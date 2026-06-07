package casinos

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

type CasinoStats struct {
	models.Casino
	Turnover         float64 `json:"turnover"`
	TransactionCount int64   `json:"transaction_count"`
	ConversionRate   float64 `json:"conversion_rate"`
}

func (s *Service) List(ctx context.Context) ([]CasinoStats, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT c.id, c.name, c.api_key, c.webhook_url, c.ip_whitelist, c.status, c.is_sandbox,
		       c.created_at, c.updated_at,
		       COALESCE(SUM(CASE WHEN t.status = 'PAID' THEN t.amount ELSE 0 END), 0),
		       COUNT(t.id)
		FROM casinos c LEFT JOIN transactions t ON t.casino_id = c.id
		GROUP BY c.id ORDER BY c.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []CasinoStats
	for rows.Next() {
		var cs CasinoStats
		if err := rows.Scan(
			&cs.ID, &cs.Name, &cs.APIKey, &cs.WebhookURL, &cs.IPWhitelist, &cs.Status, &cs.IsSandbox,
			&cs.CreatedAt, &cs.UpdatedAt, &cs.Turnover, &cs.TransactionCount,
		); err != nil {
			return nil, err
		}
		list = append(list, cs)
	}
	return list, rows.Err()
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*CasinoStats, error) {
	var cs CasinoStats
	err := s.db.Pool.QueryRow(ctx, `
		SELECT c.id, c.name, c.api_key, c.webhook_url, c.ip_whitelist, c.status, c.is_sandbox,
		       c.created_at, c.updated_at,
		       COALESCE(SUM(CASE WHEN t.status = 'PAID' THEN t.amount ELSE 0 END), 0),
		       COUNT(t.id)
		FROM casinos c LEFT JOIN transactions t ON t.casino_id = c.id
		WHERE c.id = $1 GROUP BY c.id
	`, id).Scan(
		&cs.ID, &cs.Name, &cs.APIKey, &cs.WebhookURL, &cs.IPWhitelist, &cs.Status, &cs.IsSandbox,
		&cs.CreatedAt, &cs.UpdatedAt, &cs.Turnover, &cs.TransactionCount,
	)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*models.Casino, error) {
	apiKey := crypto.GenerateAPIKey()

	var c models.Casino
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO casinos (name, api_key, webhook_url, status, is_sandbox)
		VALUES ($1, $2, $3, 'ACTIVE', $4)
		RETURNING id, name, api_key, webhook_url, ip_whitelist, status, is_sandbox, created_at, updated_at
	`, req.Name, apiKey, req.WebhookURL, req.IsSandbox).Scan(
		&c.ID, &c.Name, &c.APIKey, &c.WebhookURL, &c.IPWhitelist, &c.Status, &c.IsSandbox, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create casino: %w", err)
	}
	return &c, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status models.EntityStatus) error {
	tag, err := s.db.Pool.Exec(ctx, `UPDATE casinos SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Service) RegenerateAPIKey(ctx context.Context, id uuid.UUID) (string, error) {
	apiKey := crypto.GenerateAPIKey()
	tag, err := s.db.Pool.Exec(ctx, `UPDATE casinos SET api_key = $2, updated_at = NOW() WHERE id = $1`, id, apiKey)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", pgx.ErrNoRows
	}
	return apiKey, nil
}
