package sandbox

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/paymentsgate/paymentsgate/internal/casinos"
	"github.com/paymentsgate/paymentsgate/internal/providers"
	"github.com/paymentsgate/paymentsgate/internal/requisites"
	"github.com/paymentsgate/paymentsgate/internal/routing"
	"github.com/paymentsgate/paymentsgate/internal/transactions"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/models"
)

type Service struct {
	db           *database.DB
	casinoSvc    *casinos.Service
	providerSvc  *providers.Service
	requisiteSvc *requisites.Service
	ruleSvc      *routing.RulesService
	txSvc        *transactions.Service
}

func NewService(db *database.DB, casinoSvc *casinos.Service, providerSvc *providers.Service,
	requisiteSvc *requisites.Service, ruleSvc *routing.RulesService, txSvc *transactions.Service) *Service {
	return &Service{
		db: db, casinoSvc: casinoSvc, providerSvc: providerSvc,
		requisiteSvc: requisiteSvc, ruleSvc: ruleSvc, txSvc: txSvc,
	}
}

type SimulatePaymentRequest struct {
	TransactionID uuid.UUID `json:"transaction_id" binding:"required"`
}

func (s *Service) SetupSandbox(ctx context.Context) (map[string]interface{}, error) {
	casino, err := s.casinoSvc.Create(ctx, casinos.CreateRequest{
		Name: "Sandbox Casino", IsSandbox: true,
	})
	if err != nil {
		return nil, err
	}

	provider, err := s.providerSvc.Create(ctx, providers.CreateRequest{
		Name: "Sandbox Provider", IsSandbox: true,
	})
	if err != nil {
		return nil, err
	}

	req, err := s.requisiteSvc.Create(ctx, requisites.CreateRequest{
		ProviderID: provider.ID, BankName: "Sandbox Bank", HolderName: "Test Holder",
		AccountNumber: "SB-" + uuid.New().String()[:8], Currency: "USD", Country: "US",
		DailyLimit: 1000000, IsSandbox: true,
	})
	if err != nil {
		return nil, err
	}

	rule, err := s.ruleSvc.Create(ctx, routing.CreateRequest{
		Priority: 1, Weight: 100, ProviderID: provider.ID, IsSandbox: true,
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"casino": casino, "provider": provider, "requisite": req, "route_rule": rule,
	}, nil
}

func (s *Service) CreateTestDeposit(ctx context.Context, casinoID uuid.UUID, amount float64) (*transactions.DepositResponse, error) {
	currencies := []string{"USD", "EUR", "GBP"}
	countries := []string{"US", "DE", "GB", "FR", "CA"}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	return s.txSvc.CreateDeposit(ctx, casinoID, transactions.CreateDepositRequest{
		Amount:   amount,
		Currency: currencies[rng.Intn(len(currencies))],
		Country:  countries[rng.Intn(len(countries))],
	}, true)
}

func (s *Service) SimulatePayment(ctx context.Context, txID uuid.UUID) error {
	return s.txSvc.UpdateStatus(ctx, txID, models.TxStatusPaid)
}

func (s *Service) GenerateTraffic(ctx context.Context, casinoID uuid.UUID, count int) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	amounts := []float64{50, 100, 250, 500, 1000, 2500}

	for i := 0; i < count; i++ {
		rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
		resp, err := s.CreateTestDeposit(ctx, casinoID, amounts[rng.Intn(len(amounts))])
		if err != nil {
			continue
		}
		ids = append(ids, resp.TransactionID)

		if rng.Float32() > 0.3 {
			_ = s.SimulatePayment(ctx, resp.TransactionID)
		}
	}
	return ids, nil
}

func (s *Service) GenerateStats(ctx context.Context) (string, error) {
	var casinoID uuid.UUID
	err := s.db.Pool.QueryRow(ctx, `SELECT id FROM casinos WHERE is_sandbox = true LIMIT 1`).Scan(&casinoID)
	if err != nil {
		setup, sErr := s.SetupSandbox(ctx)
		if sErr != nil {
			return "", sErr
		}
		c := setup["casino"].(*models.Casino)
		casinoID = c.ID
	}

	ids, err := s.GenerateTraffic(ctx, casinoID, 50)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Generated %d sandbox transactions", len(ids)), nil
}
