package analytics

import (
	"context"
	"time"

	"github.com/paymentsgate/paymentsgate/pkg/database"
)

type Service struct {
	db *database.DB
}

func NewService(db *database.DB) *Service {
	return &Service{db: db}
}

type DashboardStats struct {
	TurnoverDay       float64 `json:"turnover_day"`
	TurnoverWeek      float64 `json:"turnover_week"`
	TurnoverMonth     float64 `json:"turnover_month"`
	Profit            float64 `json:"profit"`
	TransactionCount  int64   `json:"transaction_count"`
	ActiveProviders   int64   `json:"active_providers"`
	ActiveRequisites  int64   `json:"active_requisites"`
	ConversionRate    float64 `json:"conversion_rate"`
	AvgProcessingMs   float64 `json:"avg_processing_ms"`
}

type ChartPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type DistributionPoint struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type RecentEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type DashboardResponse struct {
	Stats           DashboardStats      `json:"stats"`
	TurnoverHourly  []ChartPoint          `json:"turnover_hourly"`
	TurnoverDaily   []ChartPoint          `json:"turnover_daily"`
	ByProvider      []DistributionPoint  `json:"by_provider"`
	ByCasino        []DistributionPoint  `json:"by_casino"`
	ByCountry       []DistributionPoint  `json:"by_country"`
	RecentEvents    []RecentEvent         `json:"recent_events"`
}

func (s *Service) GetDashboard(ctx context.Context, isSandbox *bool) (*DashboardResponse, error) {
	sandboxFilter := ""
	args := []interface{}{}
	if isSandbox != nil {
		sandboxFilter = " AND is_sandbox = $1"
		args = append(args, *isSandbox)
	}

	stats := DashboardStats{}

	_ = s.db.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM transactions
		WHERE status = 'PAID' AND created_at >= NOW() - INTERVAL '1 day'`+sandboxFilter, args...).Scan(&stats.TurnoverDay)

	_ = s.db.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM transactions
		WHERE status = 'PAID' AND created_at >= NOW() - INTERVAL '7 days'`+sandboxFilter, args...).Scan(&stats.TurnoverWeek)

	_ = s.db.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM transactions
		WHERE status = 'PAID' AND created_at >= NOW() - INTERVAL '30 days'`+sandboxFilter, args...).Scan(&stats.TurnoverMonth)

	stats.Profit = stats.TurnoverMonth * 0.025

	_ = s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE 1=1`+sandboxFilter, args...).Scan(&stats.TransactionCount)
	_ = s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM providers WHERE status = 'ACTIVE'`+sandboxFilter, args...).Scan(&stats.ActiveProviders)
	_ = s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM requisites WHERE status = 'ACTIVE'`+sandboxFilter, args...).Scan(&stats.ActiveRequisites)

	var paid, total int64
	_ = s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE status = 'PAID'`+sandboxFilter, args...).Scan(&paid)
	_ = s.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE status IN ('PAID','EXPIRED','CANCELLED')`+sandboxFilter, args...).Scan(&total)
	if total > 0 {
		stats.ConversionRate = float64(paid) / float64(total) * 100
	}

	_ = s.db.Pool.QueryRow(ctx, `SELECT COALESCE(AVG(processing_ms), 0) FROM transactions WHERE processing_ms IS NOT NULL`+sandboxFilter, args...).Scan(&stats.AvgProcessingMs)

	resp := &DashboardResponse{Stats: stats}

	hourlyRows, _ := s.db.Pool.Query(ctx, `
		SELECT TO_CHAR(date_trunc('hour', created_at), 'HH24:00') as hour,
		       COALESCE(SUM(amount), 0)
		FROM transactions WHERE status = 'PAID' AND created_at >= NOW() - INTERVAL '24 hours'`+sandboxFilter+`
		GROUP BY 1 ORDER BY 1`, args...)
	if hourlyRows != nil {
		defer hourlyRows.Close()
		for hourlyRows.Next() {
			var p ChartPoint
			_ = hourlyRows.Scan(&p.Label, &p.Value)
			resp.TurnoverHourly = append(resp.TurnoverHourly, p)
		}
	}

	dailyRows, _ := s.db.Pool.Query(ctx, `
		SELECT TO_CHAR(date_trunc('day', created_at), 'Mon DD') as day,
		       COALESCE(SUM(amount), 0)
		FROM transactions WHERE status = 'PAID' AND created_at >= NOW() - INTERVAL '30 days'`+sandboxFilter+`
		GROUP BY date_trunc('day', created_at), 1 ORDER BY date_trunc('day', created_at)`, args...)
	if dailyRows != nil {
		defer dailyRows.Close()
		for dailyRows.Next() {
			var p ChartPoint
			_ = dailyRows.Scan(&p.Label, &p.Value)
			resp.TurnoverDaily = append(resp.TurnoverDaily, p)
		}
	}

	providerRows, _ := s.db.Pool.Query(ctx, `
		SELECT COALESCE(p.name, 'Unknown'), COALESCE(SUM(t.amount), 0)
		FROM transactions t LEFT JOIN providers p ON p.id = t.provider_id
		WHERE t.status = 'PAID'`+sandboxFilter+` GROUP BY p.name ORDER BY 2 DESC LIMIT 10`, args...)
	if providerRows != nil {
		defer providerRows.Close()
		for providerRows.Next() {
			var p DistributionPoint
			_ = providerRows.Scan(&p.Name, &p.Value)
			resp.ByProvider = append(resp.ByProvider, p)
		}
	}

	casinoRows, _ := s.db.Pool.Query(ctx, `
		SELECT COALESCE(c.name, 'Unknown'), COALESCE(SUM(t.amount), 0)
		FROM transactions t LEFT JOIN casinos c ON c.id = t.casino_id
		WHERE t.status = 'PAID'`+sandboxFilter+` GROUP BY c.name ORDER BY 2 DESC LIMIT 10`, args...)
	if casinoRows != nil {
		defer casinoRows.Close()
		for casinoRows.Next() {
			var p DistributionPoint
			_ = casinoRows.Scan(&p.Name, &p.Value)
			resp.ByCasino = append(resp.ByCasino, p)
		}
	}

	countryRows, _ := s.db.Pool.Query(ctx, `
		SELECT country, COALESCE(SUM(amount), 0)
		FROM transactions WHERE status = 'PAID'`+sandboxFilter+` GROUP BY country ORDER BY 2 DESC`, args...)
	if countryRows != nil {
		defer countryRows.Close()
		for countryRows.Next() {
			var p DistributionPoint
			_ = countryRows.Scan(&p.Name, &p.Value)
			resp.ByCountry = append(resp.ByCountry, p)
		}
	}

	eventRows, _ := s.db.Pool.Query(ctx, `
		SELECT id::text, action, entity_type, created_at
		FROM audit_logs ORDER BY created_at DESC LIMIT 20`)
	if eventRows != nil {
		defer eventRows.Close()
		for eventRows.Next() {
			var e RecentEvent
			var entityType string
			_ = eventRows.Scan(&e.ID, &e.Type, &entityType, &e.Timestamp)
			e.Message = e.Type + " on " + entityType
			resp.RecentEvents = append(resp.RecentEvents, e)
		}
	}

	return resp, nil
}

type FinanceResponse struct {
	Turnover       float64             `json:"turnover"`
	Profit         float64             `json:"profit"`
	Commissions    float64             `json:"commissions"`
	Payouts        float64             `json:"payouts"`
	ProfitDaily    []ChartPoint        `json:"profit_daily"`
	ProfitByCasino []DistributionPoint `json:"profit_by_casino"`
	ProfitByProvider []DistributionPoint `json:"profit_by_provider"`
}

func (s *Service) GetFinance(ctx context.Context) (*FinanceResponse, error) {
	resp := &FinanceResponse{}
	_ = s.db.Pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE status = 'PAID'`).Scan(&resp.Turnover)
	resp.Commissions = resp.Turnover * 0.02
	resp.Profit = resp.Turnover * 0.025
	resp.Payouts = resp.Turnover - resp.Commissions - resp.Profit

	dailyRows, _ := s.db.Pool.Query(ctx, `
		SELECT TO_CHAR(date_trunc('day', created_at), 'Mon DD'), COALESCE(SUM(amount * 0.025), 0)
		FROM transactions WHERE status = 'PAID' AND created_at >= NOW() - INTERVAL '30 days'
		GROUP BY date_trunc('day', created_at), 1 ORDER BY date_trunc('day', created_at)`)
	if dailyRows != nil {
		defer dailyRows.Close()
		for dailyRows.Next() {
			var p ChartPoint
			_ = dailyRows.Scan(&p.Label, &p.Value)
			resp.ProfitDaily = append(resp.ProfitDaily, p)
		}
	}

	return resp, nil
}

type MonitoringStats struct {
	RPS               float64 `json:"rps"`
	ActiveConnections int     `json:"active_connections"`
	WSConnections     int     `json:"ws_connections"`
	ErrorRate         float64 `json:"error_rate"`
	AvgLatencyMs      float64 `json:"avg_latency_ms"`
	ProviderLoad      []DistributionPoint `json:"provider_load"`
}

func (s *Service) GetMonitoring(ctx context.Context, wsCount int) (*MonitoringStats, error) {
	stats := &MonitoringStats{WSConnections: wsCount}

	var total, errors int64
	_ = s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM integration_logs WHERE created_at >= NOW() - INTERVAL '1 minute'
	`).Scan(&total)
	stats.RPS = float64(total) / 60.0

	_ = s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM integration_logs WHERE status_code >= 400 AND created_at >= NOW() - INTERVAL '1 hour'
	`).Scan(&errors)
	var hourTotal int64
	_ = s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM integration_logs WHERE created_at >= NOW() - INTERVAL '1 hour'
	`).Scan(&hourTotal)
	if hourTotal > 0 {
		stats.ErrorRate = float64(errors) / float64(hourTotal) * 100
	}

	_ = s.db.Pool.QueryRow(ctx, `
		SELECT COALESCE(AVG(duration_ms), 0) FROM integration_logs WHERE created_at >= NOW() - INTERVAL '1 hour'
	`).Scan(&stats.AvgLatencyMs)

	loadRows, _ := s.db.Pool.Query(ctx, `
		SELECT COALESCE(p.name, 'Unknown'), COUNT(t.id)::float
		FROM transactions t LEFT JOIN providers p ON p.id = t.provider_id
		WHERE t.created_at >= NOW() - INTERVAL '1 hour'
		GROUP BY p.name ORDER BY 2 DESC`)
	if loadRows != nil {
		defer loadRows.Close()
		for loadRows.Next() {
			var p DistributionPoint
			_ = loadRows.Scan(&p.Name, &p.Value)
			stats.ProviderLoad = append(stats.ProviderLoad, p)
		}
	}

	return stats, nil
}
