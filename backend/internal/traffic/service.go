package traffic

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

// EnableTraffic включает трафик для провайдера
func (s *Service) EnableTraffic(ctx context.Context, providerID uuid.UUID, performedBy *uuid.UUID) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Обновляем провайдера
	_, err = tx.Exec(ctx, `
		UPDATE providers
		SET traffic_enabled = true,
		    traffic_disabled_reason = NULL,
		    traffic_disabled_at = NULL,
		    traffic_disabled_by = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`, providerID)

	if err != nil {
		return fmt.Errorf("update provider: %w", err)
	}

	// Записываем в историю
	_, err = tx.Exec(ctx, `
		INSERT INTO provider_traffic_history (id, provider_id, action, reason, performed_by, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, uuid.New(), providerID, "ENABLED", nil, performedBy)

	if err != nil {
		return fmt.Errorf("insert history: %w", err)
	}

	return tx.Commit(ctx)
}

// DisableTraffic отключает трафик для провайдера
func (s *Service) DisableTraffic(ctx context.Context, providerID uuid.UUID, reason string, performedBy *uuid.UUID) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	// Обновляем провайдера
	_, err = tx.Exec(ctx, `
		UPDATE providers
		SET traffic_enabled = false,
		    traffic_disabled_reason = $2,
		    traffic_disabled_at = $3,
		    traffic_disabled_by = $4,
		    updated_at = NOW()
		WHERE id = $1
	`, providerID, reason, now, performedBy)

	if err != nil {
		return fmt.Errorf("update provider: %w", err)
	}

	// Записываем в историю
	_, err = tx.Exec(ctx, `
		INSERT INTO provider_traffic_history (id, provider_id, action, reason, performed_by, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, uuid.New(), providerID, "DISABLED", reason, performedBy)

	if err != nil {
		return fmt.Errorf("insert history: %w", err)
	}

	return tx.Commit(ctx)
}

// BulkUpdateTraffic массово обновляет трафик для нескольких провайдеров
func (s *Service) BulkUpdateTraffic(ctx context.Context, req BulkUpdateRequest) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	for _, providerID := range req.ProviderIDs {
		if req.Enable {
			// Включаем трафик
			_, err = tx.Exec(ctx, `
				UPDATE providers
				SET traffic_enabled = true,
				    traffic_disabled_reason = NULL,
				    traffic_disabled_at = NULL,
				    traffic_disabled_by = NULL,
				    updated_at = NOW()
				WHERE id = $1
			`, providerID)
		} else {
			// Отключаем трафик
			_, err = tx.Exec(ctx, `
				UPDATE providers
				SET traffic_enabled = false,
				    traffic_disabled_reason = $2,
				    traffic_disabled_at = $3,
				    traffic_disabled_by = $4,
				    updated_at = NOW()
				WHERE id = $1
			`, providerID, req.Reason, now, req.PerformedBy)
		}

		if err != nil {
			return fmt.Errorf("update provider %s: %w", providerID, err)
		}

		// Записываем в историю
		action := "DISABLED"
		if req.Enable {
			action = "ENABLED"
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO provider_traffic_history (id, provider_id, action, reason, performed_by, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`, uuid.New(), providerID, action, req.Reason, req.PerformedBy)

		if err != nil {
			return fmt.Errorf("insert history for provider %s: %w", providerID, err)
		}
	}

	return tx.Commit(ctx)
}

// GetTrafficHistory получает историю изменений трафика провайдера
func (s *Service) GetTrafficHistory(ctx context.Context, providerID uuid.UUID, limit, offset int) ([]models.ProviderTrafficHistory, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, provider_id, action, reason, performed_by, created_at
		FROM provider_traffic_history
		WHERE provider_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, providerID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var history []models.ProviderTrafficHistory
	for rows.Next() {
		var h models.ProviderTrafficHistory
		err := rows.Scan(&h.ID, &h.ProviderID, &h.Action, &h.Reason, &h.PerformedBy, &h.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		history = append(history, h)
	}

	return history, nil
}

// GetTrafficStatus получает текущий статус трафика провайдера
func (s *Service) GetTrafficStatus(ctx context.Context, providerID uuid.UUID) (*TrafficStatus, error) {
	var status TrafficStatus

	err := s.db.Pool.QueryRow(ctx, `
		SELECT traffic_enabled, traffic_disabled_reason, traffic_disabled_at, traffic_disabled_by
		FROM providers
		WHERE id = $1
	`, providerID).Scan(&status.Enabled, &status.DisabledReason, &status.DisabledAt, &status.DisabledBy)

	if err != nil {
		return nil, fmt.Errorf("get traffic status: %w", err)
	}

	status.ProviderID = providerID
	return &status, nil
}

// Request/Response types
type BulkUpdateRequest struct {
	ProviderIDs []uuid.UUID
	Enable      bool
	Reason      *string
	PerformedBy *uuid.UUID
}

type TrafficStatus struct {
	ProviderID     uuid.UUID  `json:"provider_id"`
	Enabled        bool       `json:"enabled"`
	DisabledReason *string    `json:"disabled_reason,omitempty"`
	DisabledAt     *time.Time `json:"disabled_at,omitempty"`
	DisabledBy     *uuid.UUID `json:"disabled_by,omitempty"`
}
