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
	Amount              float64
	Currency            string
	Country             string
	IsSandbox           bool
	MerchantCustomerID  *string // For Payer Affinity
	PaymentMethod       *string // Payment method hint
	CasinoID            uuid.UUID
}

type RouteResult struct {
	ProviderID  uuid.UUID
	RequisiteID uuid.UUID
	RuleID      uuid.UUID
}

func (e *Engine) Route(ctx context.Context, req RouteRequest) (*RouteResult, error) {
	// Step 1: Try Payer Affinity - find requisite used successfully by this customer before
	if req.MerchantCustomerID != nil && *req.MerchantCustomerID != "" {
		requisite, err := e.getAffinityRequisite(ctx, *req.MerchantCustomerID, req.CasinoID, req.Amount, req.Currency, req.Country, req.IsSandbox)
		if err == nil && requisite != nil {
			// Found preferred requisite, use its provider
			return &RouteResult{
				ProviderID:  requisite.ProviderID,
				RequisiteID: requisite.ID,
				RuleID:      uuid.Nil, // Affinity-based, no rule
			}, nil
		}
	}

	// Step 2: Standard Smart Routing
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
		SELECT id, name, api_key, secret_key, merchant_id, base_url, webhook_url, ip_whitelist, status, is_sandbox, created_at, updated_at
		FROM providers WHERE id = $1 AND status = 'ACTIVE' AND is_sandbox = $2
	`, id, isSandbox).Scan(
		&p.ID, &p.Name, &p.APIKey, &p.SecretKey, &p.MerchantID, &p.BaseURL, &p.WebhookURL, &p.IPWhitelist,
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

// getAffinityRequisite implements Payer Affinity: returns requisite used successfully by customer before
func (e *Engine) getAffinityRequisite(ctx context.Context, merchantCustomerID string, casinoID uuid.UUID, amount float64, currency, country string, isSandbox bool) (*models.Requisite, error) {
	var r models.Requisite
	err := e.db.Pool.QueryRow(ctx, `
		SELECT r.id, r.provider_id, r.bank_name, r.holder_name, r.account_number,
		       r.currency, r.country, r.daily_limit, r.used_limit, r.status,
		       r.is_sandbox, r.created_at, r.updated_at
		FROM requisites r
		INNER JOIN customer_requisite_history crh
		  ON crh.requisite_id = r.id
		  AND crh.merchant_customer_id = $1
		  AND crh.casino_id = $2
		WHERE r.status = 'ACTIVE'
		  AND r.is_sandbox = $3
		  AND r.currency = $4
		  AND r.country = $5
		  AND (r.daily_limit - r.used_limit) >= $6
		ORDER BY crh.last_success_at DESC, crh.success_count DESC
		LIMIT 1
	`, merchantCustomerID, casinoID, isSandbox, currency, country, amount).Scan(
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

// RecordSuccessfulPayment records successful customer-requisite pair for future Payer Affinity
func (e *Engine) RecordSuccessfulPayment(ctx context.Context, merchantCustomerID string, casinoID, requisiteID uuid.UUID) error {
	if merchantCustomerID == "" {
		return nil // No customer ID, skip affinity tracking
	}

	_, err := e.db.Pool.Exec(ctx, `
		INSERT INTO customer_requisite_history
		  (merchant_customer_id, requisite_id, casino_id, last_success_at, success_count)
		VALUES ($1, $2, $3, NOW(), 1)
		ON CONFLICT (merchant_customer_id, requisite_id, casino_id)
		DO UPDATE SET
		  last_success_at = NOW(),
		  success_count = customer_requisite_history.success_count + 1,
		  updated_at = NOW()
	`, merchantCustomerID, requisiteID, casinoID)
	return err
}
