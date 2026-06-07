package audit

import (
	"context"
	"fmt"
	"strconv"

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

type ListFilter struct {
	Page       int
	PerPage    int
	Action     string
	EntityType string
}

func (s *Service) List(ctx context.Context, f ListFilter) ([]models.AuditLog, int64, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 20
	}

	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if f.Action != "" {
		where += fmt.Sprintf(" AND a.action = $%d", idx)
		args = append(args, f.Action)
		idx++
	}
	if f.EntityType != "" {
		where += fmt.Sprintf(" AND a.entity_type = $%d", idx)
		args = append(args, f.EntityType)
		idx++
	}

	var total int64
	_ = s.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs a "+where, args...).Scan(&total)

	offset := (f.Page - 1) * f.PerPage
	query := fmt.Sprintf(`
		SELECT a.id, a.user_id, a.action, a.entity_type, a.entity_id, a.details, a.ip_address, a.created_at,
		       COALESCE(u.email, '')
		FROM audit_logs a LEFT JOIN users u ON u.id = a.user_id
		%s ORDER BY a.created_at DESC LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)
	args = append(args, f.PerPage, offset)

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.Action, &l.EntityType, &l.EntityID,
			&l.Details, &l.IPAddress, &l.CreatedAt, &l.UserEmail,
		); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}

func (s *Service) Log(ctx context.Context, userID *uuid.UUID, action, entityType string, entityID *uuid.UUID, ip string, details map[string]interface{}) {
	_, _ = s.db.Pool.Exec(ctx, `
		INSERT INTO audit_logs (user_id, action, entity_type, entity_id, ip_address, details)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, action, entityType, entityID, ip, details)
}

type IntegrationFilter struct {
	Page          int
	PerPage       int
	Endpoint      string
	Method        string
	StatusCode    int
	ProviderID    string
	CasinoID      string
	TransactionID string
}

func (s *Service) ListIntegrationLogs(ctx context.Context, f IntegrationFilter) ([]models.IntegrationLog, int64, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 20
	}

	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if f.Endpoint != "" {
		where += fmt.Sprintf(" AND endpoint ILIKE $%d", idx)
		args = append(args, "%"+f.Endpoint+"%")
		idx++
	}
	if f.Method != "" {
		where += fmt.Sprintf(" AND method = $%d", idx)
		args = append(args, f.Method)
		idx++
	}
	if f.StatusCode > 0 {
		where += fmt.Sprintf(" AND status_code = $%d", idx)
		args = append(args, f.StatusCode)
		idx++
	}

	var total int64
	_ = s.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM integration_logs "+where, args...).Scan(&total)

	offset := (f.Page - 1) * f.PerPage
	query := fmt.Sprintf(`SELECT id, endpoint, method, status_code, duration_ms, provider_id, casino_id,
		transaction_id, request_body, response_body, error_message, is_sandbox, created_at
		FROM integration_logs %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, idx, idx+1)
	args = append(args, f.PerPage, offset)

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []models.IntegrationLog
	for rows.Next() {
		var l models.IntegrationLog
		if err := rows.Scan(
			&l.ID, &l.Endpoint, &l.Method, &l.StatusCode, &l.DurationMs,
			&l.ProviderID, &l.CasinoID, &l.TransactionID, &l.RequestBody,
			&l.ResponseBody, &l.ErrorMessage, &l.IsSandbox, &l.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}

func ParsePage(s string) int {
	p, _ := strconv.Atoi(s)
	if p < 1 {
		return 1
	}
	return p
}

func ParsePerPage(s string) int {
	p, _ := strconv.Atoi(s)
	if p < 1 || p > 100 {
		return 20
	}
	return p
}
