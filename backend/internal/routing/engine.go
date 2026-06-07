package routing

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/models"
)

var (
	ErrNoProviderAvailable = errors.New("no provider available for routing")
	ErrNoRequisiteAvailable = errors.New("no requisite available")
)

type Engine struct {
	db *database.DB
}

func NewEngine(db *database.DB) *Engine {
	return &Engine{db: db}
}

type RouteRequest struct {
	Amount    float64
	Currency  string
	Country   string
	IsSandbox bool
}

type RouteResult struct {
	ProviderID  uuid.UUID
	RequisiteID uuid.UUID
	RuleID      uuid.UUID
}

func (e *Engine) Route(ctx context.Context, req RouteRequest) (*RouteResult, error) {
	rules, err := e.getMatchingRules(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, ErrNoProviderAvailable
	}

	selectedRule := weightedSelect(rules)
	provider, err := e.getActiveProvider(ctx, selectedRule.ProviderID, req.IsSandbox)
	if err != nil {
		return nil, err
	}

	requisite, err := e.getAvailableRequisite(ctx, provider.ID, req.Amount, req.Currency, req.Country, req.IsSandbox)
	if err != nil {
		if selectedRule.IsFallback {
			return nil, err
		}
		for _, rule := range rules {
			if rule.ID == selectedRule.ID {
				continue
			}
			altProvider, pErr := e.getActiveProvider(ctx, rule.ProviderID, req.IsSandbox)
			if pErr != nil {
				continue
			}
			altReq, rErr := e.getAvailableRequisite(ctx, altProvider.ID, req.Amount, req.Currency, req.Country, req.IsSandbox)
			if rErr == nil {
				return &RouteResult{
					ProviderID:  altProvider.ID,
					RequisiteID: altReq.ID,
					RuleID:      rule.ID,
				}, nil
			}
		}
		return nil, ErrNoRequisiteAvailable
	}

	return &RouteResult{
		ProviderID:  provider.ID,
		RequisiteID: requisite.ID,
		RuleID:      selectedRule.ID,
	}, nil
}

type ruleCandidate struct {
	ID         uuid.UUID
	ProviderID uuid.UUID
	Weight     int
	Priority   int
	IsFallback bool
}

func (e *Engine) getMatchingRules(ctx context.Context, req RouteRequest) ([]ruleCandidate, error) {
	query := `
		SELECT rr.id, rr.provider_id, rr.weight, rr.priority, rr.is_fallback
		FROM route_rules rr
		JOIN providers p ON p.id = rr.provider_id
		WHERE rr.status = 'ACTIVE'
		  AND p.status = 'ACTIVE'
		  AND rr.is_sandbox = $1
		  AND (rr.country IS NULL OR rr.country = $2)
		  AND (rr.currency IS NULL OR rr.currency = $3)
		ORDER BY rr.priority ASC, rr.weight DESC
	`
	rows, err := e.db.Pool.Query(ctx, query, req.IsSandbox, req.Country, req.Currency)
	if err != nil {
		return nil, fmt.Errorf("query rules: %w", err)
	}
	defer rows.Close()

	var rules []ruleCandidate
	for rows.Next() {
		var r ruleCandidate
		if err := rows.Scan(&r.ID, &r.ProviderID, &r.Weight, &r.Priority, &r.IsFallback); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (e *Engine) getActiveProvider(ctx context.Context, id uuid.UUID, isSandbox bool) (*models.Provider, error) {
	var p models.Provider
	err := e.db.Pool.QueryRow(ctx, `
		SELECT id, name, api_key, secret_key, webhook_url, ip_whitelist, status, is_sandbox, created_at, updated_at
		FROM providers WHERE id = $1 AND status = 'ACTIVE' AND is_sandbox = $2
	`, id, isSandbox).Scan(
		&p.ID, &p.Name, &p.APIKey, &p.SecretKey, &p.WebhookURL, &p.IPWhitelist,
		&p.Status, &p.IsSandbox, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoProviderAvailable
		}
		return nil, err
	}
	return &p, nil
}

func (e *Engine) getAvailableRequisite(ctx context.Context, providerID uuid.UUID, amount float64, currency, country string, isSandbox bool) (*models.Requisite, error) {
	var r models.Requisite
	err := e.db.Pool.QueryRow(ctx, `
		SELECT id, provider_id, bank_name, holder_name, account_number, currency, country,
		       daily_limit, used_limit, status, is_sandbox, created_at, updated_at
		FROM requisites
		WHERE provider_id = $1
		  AND status = 'ACTIVE'
		  AND is_sandbox = $2
		  AND currency = $3
		  AND country = $4
		  AND (daily_limit - used_limit) >= $5
		ORDER BY (daily_limit - used_limit) DESC
		LIMIT 1
	`, providerID, isSandbox, currency, country, amount).Scan(
		&r.ID, &r.ProviderID, &r.BankName, &r.HolderName, &r.AccountNumber,
		&r.Currency, &r.Country, &r.DailyLimit, &r.UsedLimit, &r.Status,
		&r.IsSandbox, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoRequisiteAvailable
		}
		return nil, err
	}
	return &r, nil
}

func weightedSelect(rules []ruleCandidate) ruleCandidate {
	if len(rules) == 1 {
		return rules[0]
	}

	totalWeight := 0
	for _, r := range rules {
		w := r.Weight
		if w <= 0 {
			w = 1
		}
		totalWeight += w
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	pick := rng.Intn(totalWeight)
	cumulative := 0
	for _, r := range rules {
		w := r.Weight
		if w <= 0 {
			w = 1
		}
		cumulative += w
		if pick < cumulative {
			return r
		}
	}
	return rules[0]
}

func (e *Engine) ReserveRequisiteLimit(ctx context.Context, requisiteID uuid.UUID, amount float64) error {
	tag, err := e.db.Pool.Exec(ctx, `
		UPDATE requisites
		SET used_limit = used_limit + $2,
		    status = CASE WHEN (used_limit + $2) >= daily_limit THEN 'EXHAUSTED' ELSE status END,
		    updated_at = NOW()
		WHERE id = $1 AND (daily_limit - used_limit) >= $2 AND status = 'ACTIVE'
	`, requisiteID, amount)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoRequisiteAvailable
	}
	return nil
}
