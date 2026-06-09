package disputes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/paymentsgate/paymentsgate/pkg/crypto"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/models"
)

type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service {
	return &Service{db: db}
}

// CreateDispute создает новый спор и автоматически блокирует трафик провайдера
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

	// Начинаем транзакцию
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

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

	_, err = tx.Exec(ctx, `
		INSERT INTO disputes
		(id, transaction_id, provider_id, casino_id, status, reason, amount, currency, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, dispute.ID, dispute.TransactionID, dispute.ProviderID, dispute.CasinoID,
		dispute.Status, dispute.Reason, dispute.Amount, dispute.Currency,
		dispute.CreatedBy, dispute.CreatedAt, dispute.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("create dispute: %w", err)
	}

	fmt.Printf("[DISPUTE] created id=%s linked to transaction=%s provider=%s casino=%s amount=%.2f %s status=%s\n",
		dispute.ID, dispute.TransactionID, dispute.ProviderID, dispute.CasinoID, dispute.Amount, dispute.Currency, dispute.Status)

	// Автоматически блокируем трафик провайдера
	now := time.Now()
	blockReason := fmt.Sprintf("Автоматическая блокировка: создан спор #%s", dispute.ID.String()[:8])

	_, err = tx.Exec(ctx, `
		UPDATE providers
		SET traffic_enabled = false,
		    traffic_disabled_reason = $2,
		    traffic_disabled_at = $3,
		    traffic_disabled_by = $4,
		    updated_at = NOW()
		WHERE id = $1
	`, providerID, blockReason, now, req.CreatedBy)

	if err != nil {
		return nil, fmt.Errorf("disable provider traffic: %w", err)
	}

	// Записываем в историю трафика
	_, err = tx.Exec(ctx, `
		INSERT INTO provider_traffic_history (id, provider_id, action, reason, performed_by, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, uuid.New(), providerID, "DISABLED", blockReason, req.CreatedBy)

	if err != nil {
		return nil, fmt.Errorf("insert traffic history: %w", err)
	}

	// Записываем в историю спора
	_, err = tx.Exec(ctx, `
		INSERT INTO dispute_history (id, dispute_id, action, performed_by, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.New(), dispute.ID, "DISPUTE_CREATED", req.CreatedBy,
		map[string]interface{}{"reason": req.Reason, "traffic_blocked": true}, time.Now())

	if err != nil {
		return nil, fmt.Errorf("add history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	fmt.Printf("[DISPUTE] persisted id=%s and provider %s traffic disabled\n", dispute.ID, providerID)

	// Отправляем webhook провайдеру о создании спора (асинхронно)
	go s.notifyProviderAboutDispute(context.Background(), dispute, providerID)

	return dispute, nil
}

// HasOpenDispute проверяет, есть ли уже незакрытый спор по транзакции.
// Используется webhook-обработчиком, чтобы не создавать дубли при входящем уведомлении провайдера.
func (s *Service) HasOpenDispute(ctx context.Context, transactionID uuid.UUID) (bool, error) {
	var count int
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM disputes
		WHERE transaction_id = $1
		  AND status NOT IN ('MERCHANT_WON', 'PROVIDER_WON', 'CLOSED')
	`, transactionID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check open dispute: %w", err)
	}
	return count > 0, nil
}

// notifyProviderAboutDispute отправляет webhook провайдеру о создании спора.
// Передаёт учётные данные провайдера (merchant-id/secret + HMAC-подпись),
// идентификатор транзакции в формате провайдера, сумму в минорных единицах,
// логирует тело ответа и ретраит при сетевых/5xx-ошибках.
func (s *Service) notifyProviderAboutDispute(ctx context.Context, dispute *models.Dispute, providerID uuid.UUID) {
	// Получаем адрес, учётные данные и эндпоинт спора провайдера
	var baseURL, apiKey, secretKey, disputeEndpoint *string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT base_url, api_key, secret_key, dispute_endpoint FROM providers WHERE id = $1
	`, providerID).Scan(&baseURL, &apiKey, &secretKey, &disputeEndpoint)

	if err != nil || baseURL == nil || *baseURL == "" {
		fmt.Printf("[DISPUTE] Provider %s has no base_url configured (err=%v)\n", providerID, err)
		return
	}

	endpoint := ""
	if disputeEndpoint != nil {
		endpoint = *disputeEndpoint
	}

	// Идентификатор транзакции, известный ПРОВАЙДЕРУ (provider_transaction_id),
	// а не внутренний UUID PaymentsGate — иначе провайдер не сматчит спор.
	var providerTxID *string
	_ = s.db.Pool.QueryRow(ctx, `
		SELECT provider_transaction_id FROM transactions WHERE id = $1
	`, dispute.TransactionID).Scan(&providerTxID)

	txRef := dispute.TransactionID.String()
	if providerTxID != nil && *providerTxID != "" {
		txRef = *providerTxID
	} else {
		fmt.Printf("[DISPUTE] WARN: transaction %s has no provider_transaction_id; provider may not match the dispute\n", dispute.TransactionID)
	}

	// Формируем URL для webhook провайдера из настраиваемого dispute_endpoint.
	// Раньше тут было жёстко "/disputes", что давало .../api/disputes -> 404.
	webhookURL := buildProviderEndpointURL(*baseURL, endpoint)

	// Формируем payload. Сумма — в минорных единицах (копейки/центы), как ждёт провайдер.
	payload := map[string]interface{}{
		"type": "dispute.created",
		"object": map[string]interface{}{
			"dispute_id":     dispute.ID.String(),
			"transaction_id": txRef,
			"status":         dispute.Status,
			"reason":         dispute.Reason,
			"reason_code":    mapReasonToProviderCode(dispute.Reason),
			"amount":         int64(dispute.Amount*100 + 0.5), // мажорные → минорные единицы
			"currency":       dispute.Currency,
			"created_at":     dispute.CreatedAt.Format(time.RFC3339),
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[ERROR] Failed to marshal dispute webhook payload: %v\n", err)
		return
	}

	// HMAC-подпись сырого тела секретным ключом провайдера
	signature := ""
	if secretKey != nil {
		signature = crypto.HMACSign(string(payloadBytes), *secretKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	const maxAttempts = 4
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, reqErr := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(payloadBytes))
		if reqErr != nil {
			fmt.Printf("[ERROR] Failed to create dispute webhook request: %v\n", reqErr)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != nil {
			req.Header.Set("merchant-id", *apiKey)
		}
		if secretKey != nil {
			req.Header.Set("merchant-secret-key", *secretKey)
		}
		if signature != "" {
			req.Header.Set("X-Signature", signature)
		}
		// Идемпотентность: повтор с тем же ключом не создаст дубль на стороне провайдера
		req.Header.Set("Idempotency-Key", dispute.ID.String())

		resp, doErr := client.Do(req)
		if doErr != nil {
			fmt.Printf("[DISPUTE→PROVIDER] attempt %d/%d network error to provider %s: %v\n",
				attempt, maxAttempts, providerID, doErr)
		} else {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				fmt.Printf("[SUCCESS] Dispute %s sent to provider %s at %s (status=%d)\n",
					dispute.ID, providerID, webhookURL, resp.StatusCode)
				return
			}

			fmt.Printf("[DISPUTE→PROVIDER] attempt %d/%d status=%d url=%s body=%s\n",
				attempt, maxAttempts, resp.StatusCode, webhookURL, string(body))

			// Клиентская ошибка (кроме 409 Conflict) — ретрай не поможет
			if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusConflict {
				fmt.Printf("[DISPUTE→PROVIDER] client error %d — not retrying\n", resp.StatusCode)
				return
			}
		}

		if attempt < maxAttempts {
			time.Sleep(time.Duration(1<<attempt) * time.Second) // 2s, 4s, 8s
		}
	}

	fmt.Printf("[DISPUTE→PROVIDER] giving up after %d attempts for provider %s\n", maxAttempts, providerID)
}

// buildProviderEndpointURL собирает полный URL эндпоинта спора провайдера
// из base_url и настраиваемого пути (dispute_endpoint).
//
//	base="https://api.majorpay.io/api", endpoint="/dispute"     -> https://api.majorpay.io/api/dispute
//	base="https://api.majorpay.io/api", endpoint="/api/dispute" -> https://api.majorpay.io/api/dispute (схлопываем дубль /api)
//	base="https://api.majorpay.io/api", endpoint=""             -> https://api.majorpay.io/api/dispute (дефолт)
func buildProviderEndpointURL(baseURL, endpoint string) string {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = "/dispute"
	}
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	url := strings.TrimRight(baseURL, "/") + endpoint
	// Защита от случайного дублирования сегмента /api/api/, если base_url уже
	// заканчивается на /api, а в endpoint тоже указали /api.
	url = strings.ReplaceAll(url, "/api/api/", "/api/")
	return url
}

// mapReasonToProviderCode грубо классифицирует текстовую причину спора
// в код категории чарджбэка, который ожидают провайдеры.
func mapReasonToProviderCode(reason string) string {
	r := strings.ToLower(reason)
	switch {
	case strings.Contains(r, "fraud") || strings.Contains(r, "мошен"):
		return "fraud"
	case strings.Contains(r, "not received") || strings.Contains(r, "не получ"):
		return "product_not_received"
	case strings.Contains(r, "duplicate") || strings.Contains(r, "дубл"):
		return "duplicate"
	case strings.Contains(r, "amount") || strings.Contains(r, "сумм"):
		return "amount_mismatch"
	default:
		return "general"
	}
}

// GetDispute получает спор по ID
func (s *Service) GetDispute(ctx context.Context, disputeID uuid.UUID) (*models.Dispute, error) {
	var dispute models.Dispute

	err := s.db.Pool.QueryRow(ctx, `
		SELECT d.id, d.transaction_id, d.provider_id, d.casino_id, d.status, d.reason,
		       d.amount, d.currency, d.created_by, d.resolved_by, d.resolved_at,
		       d.created_at, d.updated_at,
		       COALESCE(p.name, '') as provider_name, COALESCE(c.name, '') as casino_name
		FROM disputes d
		LEFT JOIN providers p ON d.provider_id = p.id
		LEFT JOIN casinos c ON d.casino_id = c.id
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

// buildListDisputesQuery собирает SQL-запрос и аргументы для списка споров.
// Вынесено отдельно, чтобы покрыть логику фильтров и пагинации юнит-тестами без БД.
func buildListDisputesQuery(filter DisputeFilter) (string, []interface{}) {
	query := `
		SELECT d.id, d.transaction_id, d.provider_id, d.casino_id, d.status, d.reason,
		       d.amount, d.currency, d.created_by, d.resolved_by, d.resolved_at,
		       d.created_at, d.updated_at,
		       COALESCE(p.name, '') as provider_name, COALESCE(c.name, '') as casino_name
		FROM disputes d
		LEFT JOIN providers p ON d.provider_id = p.id
		LEFT JOIN casinos c ON d.casino_id = c.id
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

	return query, args
}

// ListDisputes получает список споров с фильтрами
func (s *Service) ListDisputes(ctx context.Context, filter DisputeFilter) ([]models.Dispute, error) {
	query, args := buildListDisputesQuery(filter)

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query disputes: %w", err)
	}
	defer rows.Close()

	// Возвращаем непустой слайс, чтобы API отдавал [] вместо null
	disputes := []models.Dispute{}
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate disputes: %w", err)
	}

	fmt.Printf("[DISPUTE] list returned %d disputes (status=%v provider=%v casino=%v)\n",
		len(disputes), filter.Status, filter.ProviderID, filter.CasinoID)

	return disputes, nil
}

// isResolvedStatus сообщает, является ли статус терминальным (спор разрешён/закрыт).
func isResolvedStatus(status models.DisputeStatus) bool {
	return status == models.DisputeClosed ||
		status == models.DisputeMerchantWon ||
		status == models.DisputeProviderWon
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
	if isResolvedStatus(status) {
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

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	fmt.Printf("[DISPUTE] updated id=%s new_status=%s\n", disputeID, status)
	return nil
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
