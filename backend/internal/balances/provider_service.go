package balances

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/models"
)

type ProviderBalanceService struct {
	db *database.DB
}

func NewProviderBalanceService(db *database.DB) *ProviderBalanceService {
	return &ProviderBalanceService{db: db}
}

// GetOrCreateBalance получает или создает баланс провайдера
func (s *ProviderBalanceService) GetOrCreateBalance(ctx context.Context, providerID uuid.UUID, currency string) (*models.ProviderBalance, error) {
	var balance models.ProviderBalance

	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, provider_id, balance, frozen_balance, currency, created_at, updated_at
		FROM provider_balances
		WHERE provider_id = $1 AND currency = $2
	`, providerID, currency).Scan(
		&balance.ID, &balance.ProviderID, &balance.Balance, &balance.FrozenBalance,
		&balance.Currency, &balance.CreatedAt, &balance.UpdatedAt,
	)

	if err != nil {
		// Если баланс не найден, создаем новый
		balance.ID = uuid.New()
		balance.ProviderID = providerID
		balance.Balance = 0
		balance.FrozenBalance = 0
		balance.Currency = currency
		balance.CreatedAt = time.Now()
		balance.UpdatedAt = time.Now()

		_, err = s.db.Pool.Exec(ctx, `
			INSERT INTO provider_balances (id, provider_id, balance, frozen_balance, currency, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, balance.ID, balance.ProviderID, balance.Balance, balance.FrozenBalance,
			balance.Currency, balance.CreatedAt, balance.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("create provider balance: %w", err)
		}
	}

	return &balance, nil
}

// GetBalance получает баланс провайдера
func (s *ProviderBalanceService) GetBalance(ctx context.Context, providerID uuid.UUID, currency string) (*models.ProviderBalance, error) {
	var balance models.ProviderBalance

	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, provider_id, balance, frozen_balance, currency, created_at, updated_at
		FROM provider_balances
		WHERE provider_id = $1 AND currency = $2
	`, providerID, currency).Scan(
		&balance.ID, &balance.ProviderID, &balance.Balance, &balance.FrozenBalance,
		&balance.Currency, &balance.CreatedAt, &balance.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get provider balance: %w", err)
	}

	return &balance, nil
}

// AddTransaction добавляет транзакцию и обновляет баланс
func (s *ProviderBalanceService) AddTransaction(ctx context.Context, req AddTransactionRequest) (*models.ProviderBalanceTransaction, error) {
	// Получаем или создаем баланс
	balance, err := s.GetOrCreateBalance(ctx, req.ProviderID, req.Currency)
	if err != nil {
		return nil, err
	}

	// Начинаем транзакцию
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	balanceBefore := balance.Balance
	balanceAfter := balanceBefore + req.Amount

	// Обновляем баланс
	_, err = tx.Exec(ctx, `
		UPDATE provider_balances
		SET balance = $1, updated_at = NOW()
		WHERE id = $2
	`, balanceAfter, balance.ID)

	if err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	// Создаем запись транзакции
	txRecord := &models.ProviderBalanceTransaction{
		ID:                uuid.New(),
		ProviderBalanceID: balance.ID,
		ProviderID:        req.ProviderID,
		Type:              req.Type,
		Amount:            req.Amount,
		BalanceBefore:     balanceBefore,
		BalanceAfter:      balanceAfter,
		Description:       req.Description,
		ReferenceType:     req.ReferenceType,
		ReferenceID:       req.ReferenceID,
		PerformedBy:       req.PerformedBy,
		CreatedAt:         time.Now(),
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO provider_balance_transactions
		(id, provider_balance_id, provider_id, type, amount, balance_before, balance_after,
		 description, reference_type, reference_id, performed_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, txRecord.ID, txRecord.ProviderBalanceID, txRecord.ProviderID, txRecord.Type,
		txRecord.Amount, txRecord.BalanceBefore, txRecord.BalanceAfter, txRecord.Description,
		txRecord.ReferenceType, txRecord.ReferenceID, txRecord.PerformedBy, txRecord.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return txRecord, nil
}

// FreezeAmount замораживает средства на балансе
func (s *ProviderBalanceService) FreezeAmount(ctx context.Context, providerID uuid.UUID, currency string, amount float64, reason string, referenceID *uuid.UUID, performedBy *uuid.UUID) error {
	balance, err := s.GetOrCreateBalance(ctx, providerID, currency)
	if err != nil {
		return err
	}

	if balance.Balance < amount {
		return fmt.Errorf("insufficient balance: have %.2f, need %.2f", balance.Balance, amount)
	}

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Переносим средства из основного баланса в замороженный
	_, err = tx.Exec(ctx, `
		UPDATE provider_balances
		SET balance = balance - $1, frozen_balance = frozen_balance + $1, updated_at = NOW()
		WHERE id = $2
	`, amount, balance.ID)

	if err != nil {
		return fmt.Errorf("freeze balance: %w", err)
	}

	// Записываем транзакцию заморозки
	desc := reason
	_, err = tx.Exec(ctx, `
		INSERT INTO provider_balance_transactions
		(id, provider_balance_id, provider_id, type, amount, balance_before, balance_after,
		 description, reference_type, reference_id, performed_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
	`, uuid.New(), balance.ID, providerID, models.BalanceTxFreeze, -amount,
		balance.Balance, balance.Balance-amount, &desc, strPtr("DISPUTE"), referenceID, performedBy)

	if err != nil {
		return fmt.Errorf("insert freeze transaction: %w", err)
	}

	return tx.Commit(ctx)
}

// UnfreezeAmount размораживает средства
func (s *ProviderBalanceService) UnfreezeAmount(ctx context.Context, providerID uuid.UUID, currency string, amount float64, reason string, referenceID *uuid.UUID, performedBy *uuid.UUID) error {
	balance, err := s.GetBalance(ctx, providerID, currency)
	if err != nil {
		return err
	}

	if balance.FrozenBalance < amount {
		return fmt.Errorf("insufficient frozen balance: have %.2f, need %.2f", balance.FrozenBalance, amount)
	}

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Возвращаем средства из замороженного в основной баланс
	_, err = tx.Exec(ctx, `
		UPDATE provider_balances
		SET balance = balance + $1, frozen_balance = frozen_balance - $1, updated_at = NOW()
		WHERE id = $2
	`, amount, balance.ID)

	if err != nil {
		return fmt.Errorf("unfreeze balance: %w", err)
	}

	// Записываем транзакцию разморозки
	desc := reason
	_, err = tx.Exec(ctx, `
		INSERT INTO provider_balance_transactions
		(id, provider_balance_id, provider_id, type, amount, balance_before, balance_after,
		 description, reference_type, reference_id, performed_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
	`, uuid.New(), balance.ID, providerID, models.BalanceTxUnfreeze, amount,
		balance.Balance, balance.Balance+amount, &desc, strPtr("DISPUTE"), referenceID, performedBy)

	if err != nil {
		return fmt.Errorf("insert unfreeze transaction: %w", err)
	}

	return tx.Commit(ctx)
}

// GetTransactions получает историю транзакций баланса
func (s *ProviderBalanceService) GetTransactions(ctx context.Context, providerID uuid.UUID, limit, offset int) ([]models.ProviderBalanceTransaction, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, provider_balance_id, provider_id, type, amount, balance_before, balance_after,
		       description, reference_type, reference_id, performed_by, created_at
		FROM provider_balance_transactions
		WHERE provider_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, providerID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.ProviderBalanceTransaction
	for rows.Next() {
		var tx models.ProviderBalanceTransaction
		err := rows.Scan(
			&tx.ID, &tx.ProviderBalanceID, &tx.ProviderID, &tx.Type, &tx.Amount,
			&tx.BalanceBefore, &tx.BalanceAfter, &tx.Description, &tx.ReferenceType,
			&tx.ReferenceID, &tx.PerformedBy, &tx.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// GetStats получает статистику по балансу за период
func (s *ProviderBalanceService) GetStats(ctx context.Context, providerID uuid.UUID, currency string, from, to time.Time) (*BalanceStats, error) {
	var stats BalanceStats

	err := s.db.Pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'DEPOSIT' THEN amount ELSE 0 END), 0) as total_deposits,
			COALESCE(SUM(CASE WHEN type = 'WITHDRAWAL' THEN ABS(amount) ELSE 0 END), 0) as total_withdrawals,
			COALESCE(SUM(CASE WHEN type = 'FEE' THEN ABS(amount) ELSE 0 END), 0) as total_fees,
			COALESCE(SUM(CASE WHEN type = 'COMMISSION' THEN ABS(amount) ELSE 0 END), 0) as total_commissions,
			COUNT(*) as transaction_count
		FROM provider_balance_transactions pbt
		JOIN provider_balances pb ON pbt.provider_balance_id = pb.id
		WHERE pbt.provider_id = $1 AND pb.currency = $2
		  AND pbt.created_at >= $3 AND pbt.created_at <= $4
	`, providerID, currency, from, to).Scan(
		&stats.TotalDeposits, &stats.TotalWithdrawals, &stats.TotalFees,
		&stats.TotalCommissions, &stats.TransactionCount,
	)

	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	stats.NetAmount = stats.TotalDeposits - stats.TotalWithdrawals - stats.TotalFees - stats.TotalCommissions

	return &stats, nil
}

// Request/Response types
type AddTransactionRequest struct {
	ProviderID    uuid.UUID
	Currency      string
	Type          models.BalanceTransactionType
	Amount        float64
	Description   *string
	ReferenceType *string
	ReferenceID   *uuid.UUID
	PerformedBy   *uuid.UUID
}

type BalanceStats struct {
	TotalDeposits     float64 `json:"total_deposits"`
	TotalWithdrawals  float64 `json:"total_withdrawals"`
	TotalFees         float64 `json:"total_fees"`
	TotalCommissions  float64 `json:"total_commissions"`
	NetAmount         float64 `json:"net_amount"`
	TransactionCount  int     `json:"transaction_count"`
}

func strPtr(s string) *string {
	return &s
}
