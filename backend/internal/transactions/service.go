package transactions

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/paymentsgate/paymentsgate/internal/routing"
	"github.com/paymentsgate/paymentsgate/internal/websocket"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/integrations"
	"github.com/paymentsgate/paymentsgate/pkg/models"
)

type Service struct {
	db     *database.DB
	router *routing.Engine
	hub    *websocket.Hub
}

func NewService(db *database.DB, router *routing.Engine, hub *websocket.Hub) *Service {
	return &Service{db: db, router: router, hub: hub}
}

type CreateDepositRequest struct {
	Amount              float64 `json:"amount" binding:"required,gt=0"`
	Currency            string  `json:"currency" binding:"required,len=3"`
	Country             string  `json:"country" binding:"required,len=2"`
	ExternalID          *string `json:"external_id"`
	PlayerID            *string `json:"player_id"`
	MerchantCustomerID  *string `json:"merchant_customer_id"` // For Payer Affinity
	PaymentMethod       *string `json:"payment_method"`       // auto, card, sbp, etc.
}

type DepositResponse struct {
	TransactionID uuid.UUID         `json:"transaction_id"`
	Status        models.TransactionStatus `json:"status"`
	Requisite     *RequisiteInfo    `json:"requisite,omitempty"`
	Provider      *ProviderInfo     `json:"provider,omitempty"`
}

type RequisiteInfo struct {
	BankName      string `json:"bank_name"`
	HolderName    string `json:"holder_name"`
	AccountNumber string `json:"account_number"`
}

type ProviderInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ListFilter struct {
	Page      int
	PerPage   int
	Status    string
	Country   string
	CasinoID  string
	ProviderID string
	MinAmount *float64
	MaxAmount *float64
	DateFrom  *time.Time
	DateTo    *time.Time
	IsSandbox *bool
}

func (s *Service) CreateDeposit(ctx context.Context, casinoID uuid.UUID, req CreateDepositRequest, isSandbox bool) (*DepositResponse, error) {
	start := time.Now()
	txID := uuid.New()

	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO transactions (id, external_id, casino_id, amount, currency, country, status, player_id, is_sandbox, merchant_customer_id, payment_method)
		VALUES ($1, $2, $3, $4, $5, $6, 'NEW', $7, $8, $9, $10)
	`, txID, req.ExternalID, casinoID, req.Amount, req.Currency, req.Country, req.PlayerID, isSandbox, req.MerchantCustomerID, req.PaymentMethod)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	s.hub.Broadcast(websocket.EventNewTransaction, map[string]interface{}{
		"id":       txID,
		"amount":   req.Amount,
		"currency": req.Currency,
		"country":  req.Country,
		"status":   models.TxStatusNew,
	})

	routeResult, err := s.router.Route(ctx, routing.RouteRequest{
		Amount:             req.Amount,
		Currency:           req.Currency,
		Country:            req.Country,
		IsSandbox:          isSandbox,
		MerchantCustomerID: req.MerchantCustomerID,
		PaymentMethod:      req.PaymentMethod,
		CasinoID:           casinoID,
	})

	resp := &DepositResponse{
		TransactionID: txID,
		Status:        models.TxStatusNew,
	}

	if err != nil {
		// Log routing error for debugging
		_, _ = s.db.Pool.Exec(ctx, `
			INSERT INTO audit_logs (action, entity_type, entity_id, details)
			VALUES ('ROUTING_ERROR', 'transaction', $1, $2)
		`, txID, fmt.Sprintf(`{"error":"%s","amount":%f,"currency":"%s","country":"%s","is_sandbox":%v}`,
			err.Error(), req.Amount, req.Currency, req.Country, isSandbox))

		_, _ = s.db.Pool.Exec(ctx, `UPDATE transactions SET status = 'CANCELLED', updated_at = NOW() WHERE id = $1`, txID)
		resp.Status = models.TxStatusCancelled
		s.hub.Broadcast(websocket.EventError, map[string]interface{}{
			"transaction_id": txID,
			"error":          err.Error(),
		})
		return resp, nil
	}

	// Log successful routing
	_, _ = s.db.Pool.Exec(ctx, `
		INSERT INTO audit_logs (action, entity_type, entity_id, details)
		VALUES ('ROUTING_SUCCESS', 'transaction', $1, $2)
	`, txID, fmt.Sprintf(`{"provider_id":"%s","requisite_id":"%s","rule_id":"%s"}`,
		routeResult.ProviderID, routeResult.RequisiteID, routeResult.RuleID))

	if err := s.router.ReserveRequisiteLimit(ctx, routeResult.RequisiteID, req.Amount); err != nil {
		_, _ = s.db.Pool.Exec(ctx, `UPDATE transactions SET status = 'CANCELLED', updated_at = NOW() WHERE id = $1`, txID)
		resp.Status = models.TxStatusCancelled
		return resp, nil
	}

	now := time.Now()
	processingMs := time.Since(start).Milliseconds()
	_, err = s.db.Pool.Exec(ctx, `
		UPDATE transactions
		SET provider_id = $2, requisite_id = $3, status = 'WAITING_PAYMENT',
		    assigned_at = $4, processing_ms = $5, updated_at = NOW()
		WHERE id = $1
	`, txID, routeResult.ProviderID, routeResult.RequisiteID, now, processingMs)
	if err != nil {
		return nil, err
	}

	var requisite models.Requisite
	var provider models.Provider
	err = s.db.Pool.QueryRow(ctx, `SELECT bank_name, holder_name, account_number FROM requisites WHERE id = $1`, routeResult.RequisiteID).
		Scan(&requisite.BankName, &requisite.HolderName, &requisite.AccountNumber)
	if err != nil {
		return nil, fmt.Errorf("fetch requisite: %w", err)
	}

	err = s.db.Pool.QueryRow(ctx, `SELECT id, name, base_url, api_key, secret_key FROM providers WHERE id = $1`, routeResult.ProviderID).
		Scan(&provider.ID, &provider.Name, &provider.BaseURL, &provider.APIKey, &provider.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("fetch provider: %w", err)
	}

	// DEBUG: Always log that we reached this point
	_, _ = s.db.Pool.Exec(ctx, `
		INSERT INTO audit_logs (action, entity_type, entity_id, details)
		VALUES ('DEBUG_PROVIDER_CHECK', 'transaction', $1, $2)
	`, txID, fmt.Sprintf(`{"provider_id":"%s","provider_name":"%s","has_base_url":%v}`, provider.ID, provider.Name, provider.BaseURL != nil && *provider.BaseURL != ""))

	// Call provider API if base_url is configured
	if provider.BaseURL != nil && *provider.BaseURL != "" {
		fmt.Printf("[DEBUG] Calling provider API: %s for transaction %s\n", *provider.BaseURL, txID)

		providerClient := integrations.NewMajorPayClient(*provider.BaseURL, provider.APIKey, provider.SecretKey)

		providerReq := integrations.MajorPayDepositRequest{
			Amount:             int(req.Amount * 100), // Convert to kopecks
			MerchantCustomerID: func() string {
				if req.MerchantCustomerID != nil {
					return *req.MerchantCustomerID
				}
				return fmt.Sprintf("customer_%s", txID.String()[:8])
			}(),
			PaymentMethod: func() string {
				if req.PaymentMethod != nil {
					return *req.PaymentMethod
				}
				return "auto"
			}(),
		}

		fmt.Printf("[DEBUG] Provider request: amount=%d, merchant_customer_id=%s\n", providerReq.Amount, providerReq.MerchantCustomerID)

		providerResp, err := providerClient.CreateDeposit(ctx, providerReq)
		if err != nil {
			fmt.Printf("[ERROR] Provider API call failed: %v\n", err)
			// Log error but don't fail the transaction
			_, _ = s.db.Pool.Exec(ctx, `
				INSERT INTO audit_logs (action, entity_type, entity_id, details)
				VALUES ('PROVIDER_API_ERROR', 'transaction', $1, $2)
			`, txID, fmt.Sprintf(`{"error":"%s","provider_url":"%s"}`, err.Error(), *provider.BaseURL))
		} else {
			fmt.Printf("[SUCCESS] Provider API response: transaction_id=%s\n", providerResp.TransactionID)
			// Log successful provider call
			_, _ = s.db.Pool.Exec(ctx, `
				INSERT INTO audit_logs (action, entity_type, entity_id, details)
				VALUES ('PROVIDER_API_CALLED', 'transaction', $1, $2)
			`, txID, fmt.Sprintf(`{"provider_transaction_id":"%s","provider_url":"%s"}`, providerResp.TransactionID, *provider.BaseURL))
		}
	} else {
		fmt.Printf("[WARN] Provider %s has no base_url configured, skipping API call\n", provider.Name)
		_, _ = s.db.Pool.Exec(ctx, `
			INSERT INTO audit_logs (action, entity_type, entity_id, details)
			VALUES ('PROVIDER_NO_URL', 'transaction', $1, $2)
		`, txID, fmt.Sprintf(`{"provider_id":"%s","provider_name":"%s"}`, provider.ID, provider.Name))
	}

	resp.Status = models.TxStatusWaitingPayment
	resp.Requisite = &RequisiteInfo{
		BankName:      requisite.BankName,
		HolderName:    requisite.HolderName,
		AccountNumber: requisite.AccountNumber,
	}
	resp.Provider = &ProviderInfo{ID: provider.ID, Name: provider.Name}

	s.hub.Broadcast(websocket.EventStatusChange, map[string]interface{}{
		"transaction_id": txID,
		"status":         models.TxStatusWaitingPayment,
		"provider_id":    routeResult.ProviderID,
	})

	_, _ = s.db.Pool.Exec(ctx, `
		INSERT INTO audit_logs (action, entity_type, entity_id, details)
		VALUES ('TRANSACTION_ROUTED', 'transaction', $1, $2)
	`, txID, fmt.Sprintf(`{"provider_id":"%s","requisite_id":"%s"}`, routeResult.ProviderID, routeResult.RequisiteID))

	return resp, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	return s.scanTransaction(ctx, `
		SELECT t.id, t.external_id, t.casino_id, t.provider_id, t.requisite_id,
		       t.amount, t.currency, t.country, t.status, t.player_id, t.is_sandbox,
		       t.processing_ms, t.created_at, t.updated_at, t.assigned_at, t.paid_at,
		       COALESCE(c.name, ''), COALESCE(p.name, ''), COALESCE(r.bank_name, '')
		FROM transactions t
		LEFT JOIN casinos c ON c.id = t.casino_id
		LEFT JOIN providers p ON p.id = t.provider_id
		LEFT JOIN requisites r ON r.id = t.requisite_id
		WHERE t.id = $1
	`, id)
}

func (s *Service) List(ctx context.Context, f ListFilter) ([]models.Transaction, int64, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 20
	}

	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if f.Status != "" {
		where += fmt.Sprintf(" AND t.status = $%d", argIdx)
		args = append(args, f.Status)
		argIdx++
	}
	if f.Country != "" {
		where += fmt.Sprintf(" AND t.country = $%d", argIdx)
		args = append(args, f.Country)
		argIdx++
	}
	if f.CasinoID != "" {
		where += fmt.Sprintf(" AND t.casino_id = $%d", argIdx)
		args = append(args, f.CasinoID)
		argIdx++
	}
	if f.ProviderID != "" {
		where += fmt.Sprintf(" AND t.provider_id = $%d", argIdx)
		args = append(args, f.ProviderID)
		argIdx++
	}
	if f.IsSandbox != nil {
		where += fmt.Sprintf(" AND t.is_sandbox = $%d", argIdx)
		args = append(args, *f.IsSandbox)
		argIdx++
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM transactions t " + where
	if err := s.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.PerPage
	query := fmt.Sprintf(`
		SELECT t.id, t.external_id, t.casino_id, t.provider_id, t.requisite_id,
		       t.amount, t.currency, t.country, t.status, t.player_id, t.is_sandbox,
		       t.processing_ms, t.created_at, t.updated_at, t.assigned_at, t.paid_at,
		       COALESCE(c.name, ''), COALESCE(p.name, ''), COALESCE(r.bank_name, '')
		FROM transactions t
		LEFT JOIN casinos c ON c.id = t.casino_id
		LEFT JOIN providers p ON p.id = t.provider_id
		LEFT JOIN requisites r ON r.id = t.requisite_id
		%s ORDER BY t.created_at DESC LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, f.PerPage, offset)

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var txs []models.Transaction
	for rows.Next() {
		tx, err := scanTransactionRow(rows)
		if err != nil {
			return nil, 0, err
		}
		txs = append(txs, *tx)
	}
	return txs, total, rows.Err()
}

func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status models.TransactionStatus) error {
	query := `UPDATE transactions SET status = $2, updated_at = NOW()`
	args := []interface{}{id, status}

	if status == models.TxStatusPaid {
		query += `, paid_at = NOW()`
	}
	query += ` WHERE id = $1`

	tag, err := s.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	// Record successful payment for Payer Affinity
	if status == models.TxStatusPaid {
		var merchantCustomerID *string
		var casinoID uuid.UUID
		var requisiteID *uuid.UUID
		_ = s.db.Pool.QueryRow(ctx, `
			SELECT merchant_customer_id, casino_id, requisite_id
			FROM transactions WHERE id = $1
		`, id).Scan(&merchantCustomerID, &casinoID, &requisiteID)

		if merchantCustomerID != nil && *merchantCustomerID != "" && requisiteID != nil {
			_ = s.router.RecordSuccessfulPayment(ctx, *merchantCustomerID, casinoID, *requisiteID)
		}
	}

	s.hub.Broadcast(websocket.EventStatusChange, map[string]interface{}{
		"transaction_id": id,
		"status":         status,
	})
	return nil
}

func (s *Service) scanTransaction(ctx context.Context, query string, id uuid.UUID) (*models.Transaction, error) {
	row := s.db.Pool.QueryRow(ctx, query, id)
	return scanTransactionRow(row)
}

func scanTransactionRow(row pgx.Row) (*models.Transaction, error) {
	var tx models.Transaction
	err := row.Scan(
		&tx.ID, &tx.ExternalID, &tx.CasinoID, &tx.ProviderID, &tx.RequisiteID,
		&tx.Amount, &tx.Currency, &tx.Country, &tx.Status, &tx.PlayerID, &tx.IsSandbox,
		&tx.ProcessingMs, &tx.CreatedAt, &tx.UpdatedAt, &tx.AssignedAt, &tx.PaidAt,
		&tx.CasinoName, &tx.ProviderName, &tx.RequisiteBank,
	)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}
