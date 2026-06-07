package users

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

type CreateUserRequest struct {
	Email    string      `json:"email" binding:"required,email"`
	Password string      `json:"password" binding:"required,min=8"`
	Role     models.Role `json:"role" binding:"required"`
}

func (s *Service) List(ctx context.Context) ([]models.User, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, email, password_hash, role, status, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Service) Create(ctx context.Context, req CreateUserRequest) (*models.User, error) {
	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	var u models.User
	err = s.db.Pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role, status)
		VALUES ($1, $2, $3, 'ACTIVE')
		RETURNING id, email, password_hash, role, status, created_at, updated_at
	`, req.Email, hash, req.Role).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (s *Service) UpdateRole(ctx context.Context, id uuid.UUID, role models.Role) error {
	tag, err := s.db.Pool.Exec(ctx, `UPDATE users SET role = $2, updated_at = NOW() WHERE id = $1`, id, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status models.EntityStatus) error {
	tag, err := s.db.Pool.Exec(ctx, `UPDATE users SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
