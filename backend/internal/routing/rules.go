package routing

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/models"
)

type Service struct {
	db *database.DB
}

type RulesService = Service

func NewRulesService(db *database.DB) *RulesService {
	return &RulesService{db: db}
}

type CreateRequest struct {
	Priority   int       `json:"priority"`
	Weight     int       `json:"weight" binding:"required,gt=0"`
	Country    *string   `json:"country"`
	Currency   *string   `json:"currency"`
	ProviderID uuid.UUID `json:"provider_id" binding:"required"`
	IsFallback bool      `json:"is_fallback"`
	IsSandbox  bool      `json:"is_sandbox"`
}

func (s *Service) List(ctx context.Context) ([]models.RouteRule, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT rr.id, rr.priority, rr.weight, rr.country, rr.currency, rr.provider_id,
		       rr.status, rr.is_fallback, rr.is_sandbox, rr.created_at, rr.updated_at,
		       COALESCE(p.name, '')
		FROM route_rules rr LEFT JOIN providers p ON p.id = rr.provider_id
		ORDER BY rr.priority ASC, rr.weight DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]models.RouteRule, 0)
	for rows.Next() {
		var r models.RouteRule
		if err := rows.Scan(
			&r.ID, &r.Priority, &r.Weight, &r.Country, &r.Currency, &r.ProviderID,
			&r.Status, &r.IsFallback, &r.IsSandbox, &r.CreatedAt, &r.UpdatedAt, &r.ProviderName,
		); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*models.RouteRule, error) {
	var r models.RouteRule
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO route_rules (priority, weight, country, currency, provider_id, is_fallback, is_sandbox)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, priority, weight, country, currency, provider_id, status, is_fallback, is_sandbox, created_at, updated_at
	`, req.Priority, req.Weight, req.Country, req.Currency, req.ProviderID, req.IsFallback, req.IsSandbox).Scan(
		&r.ID, &r.Priority, &r.Weight, &r.Country, &r.Currency, &r.ProviderID,
		&r.Status, &r.IsFallback, &r.IsSandbox, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create route rule: %w", err)
	}
	return &r, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Pool.Exec(ctx, `DELETE FROM route_rules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req CreateRequest) error {
	tag, err := s.db.Pool.Exec(ctx, `
		UPDATE route_rules SET priority = $2, weight = $3, country = $4, currency = $5,
		provider_id = $6, is_fallback = $7, updated_at = NOW() WHERE id = $1
	`, id, req.Priority, req.Weight, req.Country, req.Currency, req.ProviderID, req.IsFallback)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
