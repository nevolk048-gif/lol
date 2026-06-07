package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/paymentsgate/paymentsgate/pkg/crypto"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/jwt"
	"github.com/paymentsgate/paymentsgate/pkg/models"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserBlocked        = errors.New("user is blocked")
)

type Service struct {
	db         *database.DB
	jwtManager *jwt.Manager
}

func NewService(db *database.DB, jwtManager *jwt.Manager) *Service {
	return &Service{db: db, jwtManager: jwtManager}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type TokenResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
	User         *models.User `json:"user"`
}

func (s *Service) Login(ctx context.Context, req LoginRequest, ip string) (*TokenResponse, error) {
	var user models.User
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, email, password_hash, role, status, created_at, updated_at
		FROM users WHERE email = $1
	`, req.Email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role,
		&user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if user.Status == models.StatusBlocked {
		return nil, ErrUserBlocked
	}

	if !crypto.CheckPassword(user.PasswordHash, req.Password) {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(&user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	tokenHash := crypto.HashToken(refreshToken)
	_, err = s.db.Pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, user.ID, tokenHash, time.Now().Add(7*24*time.Hour))
	if err != nil {
		return nil, err
	}

	_, _ = s.db.Pool.Exec(ctx, `
		INSERT INTO audit_logs (user_id, action, entity_type, ip_address, details)
		VALUES ($1, 'LOGIN', 'user', $2, '{}')
	`, user.ID, ip)

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900,
		User:         &user,
	}, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	tokenHash := crypto.HashToken(refreshToken)
	var exists bool
	err = s.db.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM refresh_tokens WHERE user_id = $1 AND token_hash = $2 AND expires_at > NOW())
	`, userID, tokenHash).Scan(&exists)
	if err != nil || !exists {
		return nil, ErrInvalidCredentials
	}

	var user models.User
	err = s.db.Pool.QueryRow(ctx, `
		SELECT id, email, password_hash, role, status, created_at, updated_at
		FROM users WHERE id = $1
	`, userID).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role,
		&user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(&user)
	if err != nil {
		return nil, err
	}

	newRefresh, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	_, _ = s.db.Pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	newHash := crypto.HashToken(newRefresh)
	_, _ = s.db.Pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)
	`, user.ID, newHash, time.Now().Add(7*24*time.Hour))

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		ExpiresIn:    900,
		User:         &user,
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := crypto.HashToken(refreshToken)
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	return err
}

func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, email, password_hash, role, status, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Role,
		&user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
