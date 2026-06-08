package balances

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/models"
)

type MerchantBalanceService struct {
	db *database.DB
}

func NewMerchantBalanceService(db *database.DB) *MerchantBalanceService {
	return &MerchantBalanceService{db: db}
}

// GetOrCreateBalance получает или создает баланс мерчанта
func (s *MerchantBalanceService) GetOrCreateBalance(ctx context.Context, casinoID uuid.UUID, currency string) (*models.MerchantBalance, error) {
	var balance models.MerchantBalance

	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, casino_id, balance, frozen_balance, currency, created_at, updated_at
		FROM merchant_balances
		WHERE casino_id = $1 AND currency = $2
	`, casinoID, currency).Scan(
		&balance.ID, &balance.CasinoID, &balance.Balance, &balance.FrozenBalance,
		&balance.Currency, &balance.CreatedAt, &balance.UpdatedAt,
	)

	if err != nil {
		// Если баланс не найден, создаем новый
		balance.ID = uuid.New()
		balance.CasinoID = casinoID
		balance.Balance = 0
		balance.FrozenBalance = 0
		balance.Currency = currency
		balance.CreatedAt = time.Now()
		balance.UpdatedAt = time.Now()

		_, err = s.db.Pool.Exec(ctx, `
			INSERT INTO merchant_balances (id, casino_id, balance, frozen_balance, currency, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, balance.ID, balance.CasinoID, balance.Balance, balance.FrozenBalance,
			balance.Currency, balance.CreatedAt, balance.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("create merchant balance: %w", err)
		}
	}

	return &balance, nil
}

// GetBalance получает баланс мерчанта
func (s *MerchantBalanceService) GetBalance(ctx context.Context, casinoID uuid.UUID, currency string) (*models.MerchantBalance, error) {
	var balance models.MerchantBalance

	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, casino_id, balance, frozen_balance, currency, created_at, updated_at
		FROM merchant_balances
		WHERE casino_id = $1 AND currency = $2
	`, casinoID, currency).Scan(
		&balance.ID, &balance.CasinoID, &balance.Balance, &balance.FrozenBalance,
		&balance.Currency, &balance.CreatedAt, &balance.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get merchant balance: %w", err)
	}

	return &balance, nil
}

// AddTransaction добавляет транзакцию и обновляет баланс
func (s *MerchantBalanceService) AddTransaction(ctx context.Context, req AddMerchantTransactionRequest) (*models.MerchantBalanceTransaction, error) {
	// Получаем или создаем баланс
	balance, err := s.GetOrCreateBalance(ctx, req.CasinoID, req.Currency)
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

	// Проверяем достаточность средств для списания
	if req.Amount < 0 && balanceAfter < 0 {
		return nil, fmt.Errorf("insufficient balance: have %.2f, need %.2f", balance.Balance, -req.Amount)
	}

	// Обновляем баланс
	_, err = tx.Exec(ctx, `
		UPDATE merchant_balances
		SET balance = $1, updated_at = NOW()
		WHERE id = $2
	`, balanceAfter, balance.ID)

	if err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	// Создаем запись транзакции
	txRecord := &models.MerchantBalanceTransaction{
		ID:                uuid.New(),
		MerchantBalanceID: balance.ID,
		CasinoID:          req.CasinoID,
		Type:              req.Type,
		Amount:            req.Amount,
		BalanceBefore:     balanceBefore,
		BalanceAfter:      balanceAfter,
		Description:       req.Description,
		ReferenceType:     req.ReferenceType,
		ReferenceID:       req.ReferenceID,
		CreatedAt:         time.Now(),
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO merchant_balance_transactions
		(id, merchant_balance_id, casino_id, type, amount, balance_before, balance_after,
		 description, reference_type, reference_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, txRecord.ID, txRecord.MerchantBalanceID, txRecord.CasinoID, txRecord.Type,
		txRecord.Amount, txRecord.BalanceBefore, txRecord.BalanceAfter, txRecord.Description,
		txRecord.ReferenceType, txRecord.ReferenceID, txRecord.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return txRecord, nil
}

// GetTransactions получает историю транзакций баланса
func (s *MerchantBalanceService) GetTransactions(ctx context.Context, casinoID uuid.UUID, limit, offset int) ([]models.MerchantBalanceTransaction, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, merchant_balance_id, casino_id, type, amount, balance_before, balance_after,
		       description, reference_type, reference_id, created_at
		FROM merchant_balance_transactions
		WHERE casino_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, casinoID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.MerchantBalanceTransaction
	for rows.Next() {
		var tx models.MerchantBalanceTransaction
		err := rows.Scan(
			&tx.ID, &tx.MerchantBalanceID, &tx.CasinoID, &tx.Type, &tx.Amount,
			&tx.BalanceBefore, &tx.BalanceAfter, &tx.Description, &tx.ReferenceType,
			&tx.ReferenceID, &tx.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// GetStats получает статистику по балансу за период
func (s *MerchantBalanceService) GetStats(ctx context.Context, casinoID uuid.UUID, currency string, from, to time.Time) (*MerchantBalanceStats, error) {
	var stats MerchantBalanceStats

	err := s.db.Pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'DEPOSIT' THEN amount ELSE 0 END), 0) as total_deposits,
			COALESCE(SUM(CASE WHEN type = 'WITHDRAWAL' THEN ABS(amount) ELSE 0 END), 0) as total_withdrawals,
			COALESCE(SUM(CASE WHEN type = 'FEE' THEN ABS(amount) ELSE 0 END), 0) as total_fees,
			COALESCE(SUM(CASE WHEN type = 'PAYOUT' THEN ABS(amount) ELSE 0 END), 0) as total_payouts,
			COALESCE(SUM(CASE WHEN type = 'REFUND' THEN amount ELSE 0 END), 0) as total_refunds,
			COUNT(*) as transaction_count
		FROM merchant_balance_transactions mbt
		JOIN merchant_balances mb ON mbt.merchant_balance_id = mb.id
		WHERE mbt.casino_id = $1 AND mb.currency = $2
		  AND mbt.created_at >= $3 AND mbt.created_at <= $4
	`, casinoID, currency, from, to).Scan(
		&stats.TotalDeposits, &stats.TotalWithdrawals, &stats.TotalFees,
		&stats.TotalPayouts, &stats.TotalRefunds, &stats.TransactionCount,
	)

	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	stats.NetAmount = stats.TotalDeposits - stats.TotalWithdrawals - stats.TotalFees - stats.TotalPayouts + stats.TotalRefunds

	return &stats, nil
}

// ExportTransactions экспортирует историю транзакций в CSV
func (s *MerchantBalanceService) ExportTransactions(ctx context.Context, casinoID uuid.UUID, from, to time.Time) ([]models.MerchantBalanceTransaction, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, merchant_balance_id, casino_id, type, amount, balance_before, balance_after,
		       description, reference_type, reference_id, created_at
		FROM merchant_balance_transactions
		WHERE casino_id = $1 AND created_at >= $2 AND created_at <= $3
		ORDER BY created_at DESC
	`, casinoID, from, to)

	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.MerchantBalanceTransaction
	for rows.Next() {
		var tx models.MerchantBalanceTransaction
		err := rows.Scan(
			&tx.ID, &tx.MerchantBalanceID, &tx.CasinoID, &tx.Type, &tx.Amount,
			&tx.BalanceBefore, &tx.BalanceAfter, &tx.Description, &tx.ReferenceType,
			&tx.ReferenceID, &tx.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// Request/Response types
type AddMerchantTransactionRequest struct {
	CasinoID      uuid.UUID
	Currency      string
	Type          models.BalanceTransactionType
	Amount        float64
	Description   *string
	ReferenceType *string
	ReferenceID   *uuid.UUID
}

type MerchantBalanceStats struct {
	TotalDeposits    float64 `json:"total_deposits"`
	TotalWithdrawals float64 `json:"total_withdrawals"`
	TotalFees        float64 `json:"total_fees"`
	TotalPayouts     float64 `json:"total_payouts"`
	TotalRefunds     float64 `json:"total_refunds"`
	NetAmount        float64 `json:"net_amount"`
	TransactionCount int     `json:"transaction_count"`
}
