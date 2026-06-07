package requisites

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

func NewService(db *database.DB) *Service {
	return &Service{db: db}
}

type CreateRequest struct {
	ProviderID    uuid.UUID `json:"provider_id" binding:"required"`
	BankName      string    `json:"bank_name" binding:"required"`
	HolderName    string    `json:"holder_name" binding:"required"`
	AccountNumber string    `json:"account_number" binding:"required"`
	Currency      string    `json:"currency" binding:"required,len=3"`
	Country       string    `json:"country" binding:"required,len=2"`
	DailyLimit    float64   `json:"daily_limit" binding:"required,gt=0"`
	IsSandbox     bool      `json:"is_sandbox"`
}

func (s *Service) List(ctx context.Context, providerID *uuid.UUID) ([]models.Requisite, error) {
	query := `
		SELECT id, provider_id, bank_name, holder_name, account_number, currency, country,
		       daily_limit, used_limit, status, is_sandbox, created_at, updated_at
		FROM requisites`
	args := []interface{}{}
	if providerID != nil {
		query += " WHERE provider_id = $1"
		args = append(args, *providerID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Requisite
	for rows.Next() {
		var r models.Requisite
		if err := rows.Scan(
			&r.ID, &r.ProviderID, &r.BankName, &r.HolderName, &r.AccountNumber,
			&r.Currency, &r.Country, &r.DailyLimit, &r.UsedLimit, &r.Status,
			&r.IsSandbox, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*models.Requisite, error) {
	var r models.Requisite
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO requisites (provider_id, bank_name, holder_name, account_number, currency, country, daily_limit, is_sandbox)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, provider_id, bank_name, holder_name, account_number, currency, country,
		          daily_limit, used_limit, status, is_sandbox, created_at, updated_at
	`, req.ProviderID, req.BankName, req.HolderName, req.AccountNumber, req.Currency, req.Country, req.DailyLimit, req.IsSandbox).Scan(
		&r.ID, &r.ProviderID, &r.BankName, &r.HolderName, &r.AccountNumber,
		&r.Currency, &r.Country, &r.DailyLimit, &r.UsedLimit, &r.Status,
		&r.IsSandbox, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create requisite: %w", err)
	}
	return &r, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status models.RequisiteStatus) error {
	tag, err := s.db.Pool.Exec(ctx, `UPDATE requisites SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Service) ResetDailyLimits(ctx context.Context) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE requisites SET used_limit = 0,
		status = CASE WHEN status = 'EXHAUSTED' THEN 'ACTIVE' ELSE status END,
		updated_at = NOW()
	`)
	return err
}
