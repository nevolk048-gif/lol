package disputes

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/models"
)

type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service {
	return &Service{db: db}
}

// CreateDispute создает новый спор
func (s *Service) CreateDispute(ctx context.Context, req CreateDisputeRequest) (*models.Dispute, error) {
	// Получаем информацию о транзакции
	var transactionID, providerID, casinoID uuid.UUID
	var amount float64
	var currency string

	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, provider_id, casino_id, amount, currency
		FROM transactions
		WHERE id = $1
	`, req.TransactionID).Scan(&transactionID, &providerID, &casinoID, &amount, &currency)

	if err != nil {
		return nil, fmt.Errorf("get transaction: %w", err)
	}

	// Создаем спор
	dispute := &models.Dispute{
		ID:            uuid.New(),
		TransactionID: transactionID,
		ProviderID:    providerID,
		CasinoID:      casinoID,
		Status:        models.DisputeNew,
		Reason:        req.Reason,
		Amount:        amount,
		Currency:      currency,
		CreatedBy:     req.CreatedBy,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, err = s.db.Pool.Exec(ctx, `
		INSERT INTO disputes
		(id, transaction_id, provider_id, casino_id, status, reason, amount, currency, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, dispute.ID, dispute.TransactionID, dispute.ProviderID, dispute.CasinoID,
		dispute.Status, dispute.Reason, dispute.Amount, dispute.Currency,
		dispute.CreatedBy, dispute.CreatedAt, dispute.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("create dispute: %w", err)
	}

	// Записываем в историю
	err = s.addHistory(ctx, dispute.ID, "DISPUTE_CREATED", req.CreatedBy, map[string]interface{}{
		"reason": req.Reason,
	})
	if err != nil {
		return nil, fmt.Errorf("add history: %w", err)
	}

	return dispute, nil
}

// GetDispute получает спор по ID
func (s *Service) GetDispute(ctx context.Context, disputeID uuid.UUID) (*models.Dispute, error) {
	var dispute models.Dispute

	err := s.db.Pool.QueryRow(ctx, `
		SELECT d.id, d.transaction_id, d.provider_id, d.casino_id, d.status, d.reason,
		       d.amount, d.currency, d.created_by, d.resolved_by, d.resolved_at,
		       d.created_at, d.updated_at, p.name as provider_name, c.name as casino_name
		FROM disputes d
		JOIN providers p ON d.provider_id = p.id
		JOIN casinos c ON d.casino_id = c.id
		WHERE d.id = $1
	`, disputeID).Scan(
		&dispute.ID, &dispute.TransactionID, &dispute.ProviderID, &dispute.CasinoID,
		&dispute.Status, &dispute.Reason, &dispute.Amount, &dispute.Currency,
		&dispute.CreatedBy, &dispute.ResolvedBy, &dispute.ResolvedAt,
		&dispute.CreatedAt, &dispute.UpdatedAt, &dispute.ProviderName, &dispute.CasinoName,
	)

	if err != nil {
		return nil, fmt.Errorf("get dispute: %w", err)
	}

	return &dispute, nil
}

// ListDisputes получает список споров с фильтрами
func (s *Service) ListDisputes(ctx context.Context, filter DisputeFilter) ([]models.Dispute, error) {
	query := `
		SELECT d.id, d.transaction_id, d.provider_id, d.casino_id, d.status, d.reason,
		       d.amount, d.currency, d.created_by, d.resolved_by, d.resolved_at,
		       d.created_at, d.updated_at, p.name as provider_name, c.name as casino_name
		FROM disputes d
		JOIN providers p ON d.provider_id = p.id
		JOIN casinos c ON d.casino_id = c.id
		WHERE 1=1
	`

	args := []interface{}{}
	argNum := 1

	if filter.Status != nil {
		query += fmt.Sprintf(" AND d.status = $%d", argNum)
		args = append(args, *filter.Status)
		argNum++
	}

	if filter.ProviderID != nil {
		query += fmt.Sprintf(" AND d.provider_id = $%d", argNum)
		args = append(args, *filter.ProviderID)
		argNum++
	}

	if filter.CasinoID != nil {
		query += fmt.Sprintf(" AND d.casino_id = $%d", argNum)
		args = append(args, *filter.CasinoID)
		argNum++
	}

	query += " ORDER BY d.created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
		args = append(args, filter.Limit, filter.Offset)
	}

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query disputes: %w", err)
	}
	defer rows.Close()

	var disputes []models.Dispute
	for rows.Next() {
		var d models.Dispute
		err := rows.Scan(
			&d.ID, &d.TransactionID, &d.ProviderID, &d.CasinoID,
			&d.Status, &d.Reason, &d.Amount, &d.Currency,
			&d.CreatedBy, &d.ResolvedBy, &d.ResolvedAt,
			&d.CreatedAt, &d.UpdatedAt, &d.ProviderName, &d.CasinoName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan dispute: %w", err)
		}
		disputes = append(disputes, d)
	}

	return disputes, nil
}

// UpdateStatus обновляет статус спора
func (s *Service) UpdateStatus(ctx context.Context, disputeID uuid.UUID, status models.DisputeStatus, userID *uuid.UUID) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	var resolvedAt *time.Time
	var resolvedBy *uuid.UUID

	// Если спор разрешен, сохраняем время и пользователя
	if status == models.DisputeClosed || status == models.DisputeMerchantWon || status == models.DisputeProviderWon {
		resolvedAt = &now
		resolvedBy = userID
	}

	_, err = tx.Exec(ctx, `
		UPDATE disputes
		SET status = $1, resolved_at = $2, resolved_by = $3, updated_at = $4
		WHERE id = $5
	`, status, resolvedAt, resolvedBy, now, disputeID)

	if err != nil {
		return fmt.Errorf("update dispute status: %w", err)
	}

	// Записываем в историю
	_, err = tx.Exec(ctx, `
		INSERT INTO dispute_history (id, dispute_id, action, performed_by, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.New(), disputeID, "STATUS_CHANGED", userID,
		map[string]interface{}{"new_status": status}, now)

	if err != nil {
		return fmt.Errorf("add history: %w", err)
	}

	return tx.Commit(ctx)
}

// AddMessage добавляет сообщение в спор
func (s *Service) AddMessage(ctx context.Context, req AddMessageRequest) (*models.DisputeMessage, error) {
	message := &models.DisputeMessage{
		ID:          uuid.New(),
		DisputeID:   req.DisputeID,
		SenderType:  req.SenderType,
		SenderID:    req.SenderID,
		Message:     req.Message,
		Attachments: req.Attachments,
		CreatedAt:   time.Now(),
	}

	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO dispute_messages (id, dispute_id, sender_type, sender_id, message, attachments, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, message.ID, message.DisputeID, message.SenderType, message.SenderID,
		message.Message, message.Attachments, message.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	// Обновляем время изменения спора
	_, err = s.db.Pool.Exec(ctx, `
		UPDATE disputes SET updated_at = $1 WHERE id = $2
	`, time.Now(), req.DisputeID)

	if err != nil {
		return nil, fmt.Errorf("update dispute: %w", err)
	}

	// Записываем в историю
	err = s.addHistory(ctx, req.DisputeID, "MESSAGE_ADDED", &req.SenderID, map[string]interface{}{
		"sender_type": req.SenderType,
	})

	return message, err
}

// GetMessages получает сообщения спора
func (s *Service) GetMessages(ctx context.Context, disputeID uuid.UUID) ([]models.DisputeMessage, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, dispute_id, sender_type, sender_id, message, attachments, created_at
		FROM dispute_messages
		WHERE dispute_id = $1
		ORDER BY created_at ASC
	`, disputeID)

	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var messages []models.DisputeMessage
	for rows.Next() {
		var m models.DisputeMessage
		err := rows.Scan(
			&m.ID, &m.DisputeID, &m.SenderType, &m.SenderID,
			&m.Message, &m.Attachments, &m.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, m)
	}

	return messages, nil
}

// GetHistory получает историю изменений спора
func (s *Service) GetHistory(ctx context.Context, disputeID uuid.UUID) ([]models.DisputeHistory, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, dispute_id, action, performed_by, details, created_at
		FROM dispute_history
		WHERE dispute_id = $1
		ORDER BY created_at DESC
	`, disputeID)

	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var history []models.DisputeHistory
	for rows.Next() {
		var h models.DisputeHistory
		err := rows.Scan(
			&h.ID, &h.DisputeID, &h.Action, &h.PerformedBy, &h.Details, &h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		history = append(history, h)
	}

	return history, nil
}

// GetStats получает статистику по спорам
func (s *Service) GetStats(ctx context.Context, filter StatsFilter) (*DisputeStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'NEW' THEN 1 END) as new_count,
			COUNT(CASE WHEN status = 'UNDER_REVIEW' THEN 1 END) as under_review_count,
			COUNT(CASE WHEN status = 'MERCHANT_WON' THEN 1 END) as merchant_won_count,
			COUNT(CASE WHEN status = 'PROVIDER_WON' THEN 1 END) as provider_won_count,
			COUNT(CASE WHEN status = 'CLOSED' THEN 1 END) as closed_count,
			COALESCE(SUM(amount), 0) as total_amount
		FROM disputes
		WHERE 1=1
	`

	args := []interface{}{}
	argNum := 1

	if filter.From != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argNum)
		args = append(args, *filter.From)
		argNum++
	}

	if filter.To != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argNum)
		args = append(args, *filter.To)
		argNum++
	}

	var stats DisputeStats
	err := s.db.Pool.QueryRow(ctx, query, args...).Scan(
		&stats.Total, &stats.NewCount, &stats.UnderReviewCount,
		&stats.MerchantWonCount, &stats.ProviderWonCount, &stats.ClosedCount,
		&stats.TotalAmount,
	)

	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	return &stats, nil
}

func (s *Service) addHistory(ctx context.Context, disputeID uuid.UUID, action string, performedBy *uuid.UUID, details map[string]interface{}) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO dispute_history (id, dispute_id, action, performed_by, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.New(), disputeID, action, performedBy, details, time.Now())

	return err
}

// Request/Response types
type CreateDisputeRequest struct {
	TransactionID uuid.UUID
	Reason        string
	CreatedBy     *uuid.UUID
}

type DisputeFilter struct {
	Status     *models.DisputeStatus
	ProviderID *uuid.UUID
	CasinoID   *uuid.UUID
	Limit      int
	Offset     int
}

type AddMessageRequest struct {
	DisputeID   uuid.UUID
	SenderType  string
	SenderID    uuid.UUID
	Message     string
	Attachments map[string]interface{}
}

type StatsFilter struct {
	From *time.Time
	To   *time.Time
}

type DisputeStats struct {
	Total             int     `json:"total"`
	NewCount          int     `json:"new_count"`
	UnderReviewCount  int     `json:"under_review_count"`
	MerchantWonCount  int     `json:"merchant_won_count"`
	ProviderWonCount  int     `json:"provider_won_count"`
	ClosedCount       int     `json:"closed_count"`
	TotalAmount       float64 `json:"total_amount"`
}
