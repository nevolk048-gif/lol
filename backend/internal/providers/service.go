package providers

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

type CreateRequest struct {
	Name       string  `json:"name" binding:"required"`
	WebhookURL *string `json:"webhook_url"`
	IsSandbox  bool    `json:"is_sandbox"`
}

type ProviderStats struct {
	models.Provider
	Turnover       float64 `json:"turnover"`
	TransactionCount int64 `json:"transaction_count"`
	ConversionRate float64 `json:"conversion_rate"`
	AvgResponseMs  float64 `json:"avg_response_ms"`
}

func (s *Service) List(ctx context.Context) ([]ProviderStats, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT p.id, p.name, p.api_key, p.secret_key, p.webhook_url, p.ip_whitelist,
		       p.status, p.is_sandbox, p.created_at, p.updated_at,
		       COALESCE(SUM(CASE WHEN t.status = 'PAID' THEN t.amount ELSE 0 END), 0),
		       COUNT(t.id),
		       COALESCE(AVG(t.processing_ms), 0)
		FROM providers p
		LEFT JOIN transactions t ON t.provider_id = p.id
		GROUP BY p.id ORDER BY p.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ProviderStats
	for rows.Next() {
		var ps ProviderStats
		if err := rows.Scan(
			&ps.ID, &ps.Name, &ps.APIKey, &ps.SecretKey, &ps.WebhookURL, &ps.IPWhitelist,
			&ps.Status, &ps.IsSandbox, &ps.CreatedAt, &ps.UpdatedAt,
			&ps.Turnover, &ps.TransactionCount, &ps.AvgResponseMs,
		); err != nil {
			return nil, err
		}
		list = append(list, ps)
	}
	return list, rows.Err()
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*ProviderStats, error) {
	var ps ProviderStats
	err := s.db.Pool.QueryRow(ctx, `
		SELECT p.id, p.name, p.api_key, p.secret_key, p.webhook_url, p.ip_whitelist,
		       p.status, p.is_sandbox, p.created_at, p.updated_at,
		       COALESCE(SUM(CASE WHEN t.status = 'PAID' THEN t.amount ELSE 0 END), 0),
		       COUNT(t.id), COALESCE(AVG(t.processing_ms), 0)
		FROM providers p LEFT JOIN transactions t ON t.provider_id = p.id
		WHERE p.id = $1 GROUP BY p.id
	`, id).Scan(
		&ps.ID, &ps.Name, &ps.APIKey, &ps.SecretKey, &ps.WebhookURL, &ps.IPWhitelist,
		&ps.Status, &ps.IsSandbox, &ps.CreatedAt, &ps.UpdatedAt,
		&ps.Turnover, &ps.TransactionCount, &ps.AvgResponseMs,
	)
	if err != nil {
		return nil, err
	}
	return &ps, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*models.Provider, error) {
	apiKey := crypto.GenerateAPIKey()
	secretKey := crypto.GenerateSecretKey()

	var p models.Provider
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO providers (name, api_key, secret_key, webhook_url, status, is_sandbox)
		VALUES ($1, $2, $3, $4, 'ACTIVE', $5)
		RETURNING id, name, api_key, secret_key, webhook_url, ip_whitelist, status, is_sandbox, created_at, updated_at
	`, req.Name, apiKey, secretKey, req.WebhookURL, req.IsSandbox).Scan(
		&p.ID, &p.Name, &p.APIKey, &p.SecretKey, &p.WebhookURL, &p.IPWhitelist,
		&p.Status, &p.IsSandbox, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}
	return &p, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status models.EntityStatus) error {
	tag, err := s.db.Pool.Exec(ctx, `UPDATE providers SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
