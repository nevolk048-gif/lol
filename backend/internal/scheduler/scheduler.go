package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/paymentsgate/paymentsgate/internal/transactions"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/models"
)

type Scheduler struct {
	db    *database.DB
	txSvc *transactions.Service
}

func NewScheduler(db *database.DB, txSvc *transactions.Service) *Scheduler {
	return &Scheduler{
		db:    db,
		txSvc: txSvc,
	}
}

// Start запускает все фоновые задачи
func (s *Scheduler) Start(ctx context.Context) {
	// Запускаем проверку истекших транзакций каждые 5 минут
	go s.expireTransactionsJob(ctx, 5*time.Minute)
}

// expireTransactionsJob проверяет и отмечает истекшие транзакции
func (s *Scheduler) expireTransactionsJob(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Запускаем сразу при старте
	s.expireOldTransactions(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Scheduler] Stopping expire transactions job")
			return
		case <-ticker.C:
			s.expireOldTransactions(ctx)
		}
	}
}

// expireOldTransactions находит и отмечает как expired транзакции, которые слишком долго в статусе WAITING_PAYMENT
func (s *Scheduler) expireOldTransactions(ctx context.Context) {
	// Конфигурируемый TTL для транзакций (по умолчанию 30 минут)
	expirationTime := 30 * time.Minute

	// Находим все транзакции в статусе WAITING_PAYMENT старше expirationTime
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id
		FROM transactions
		WHERE status = $1
		AND created_at < NOW() - $2::interval
		ORDER BY created_at ASC
		LIMIT 100
	`, models.TxStatusWaitingPayment, expirationTime)

	if err != nil {
		log.Printf("[ERROR] Failed to query expired transactions: %v", err)
		return
	}
	defer rows.Close()

	expiredCount := 0
	for rows.Next() {
		var txID string
		if err := rows.Scan(&txID); err != nil {
			log.Printf("[ERROR] Failed to scan transaction ID: %v", err)
			continue
		}

		// Парсим UUID
		txUUID, err := parseUUID(txID)
		if err != nil {
			log.Printf("[ERROR] Failed to parse UUID %s: %v", txID, err)
			continue
		}

		// Обновляем статус на EXPIRED
		if err := s.txSvc.UpdateStatus(ctx, txUUID, models.TxStatusExpired); err != nil {
			log.Printf("[ERROR] Failed to expire transaction %s: %v", txID, err)
			continue
		}

		expiredCount++
		log.Printf("[INFO] Transaction %s marked as expired", txID)
	}

	if expiredCount > 0 {
		log.Printf("[SUCCESS] Marked %d transactions as expired", expiredCount)

		// Логируем в audit_logs
		_, _ = s.db.Pool.Exec(ctx, `
			INSERT INTO audit_logs (action, entity_type, details)
			VALUES ('BATCH_EXPIRE_TRANSACTIONS', 'transaction', $1)
		`, fmt.Sprintf(`{"count":%d,"expiration_minutes":%d}`, expiredCount, int(expirationTime.Minutes())))
	}
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
