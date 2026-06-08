package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/models"
)

type Notifier struct {
	db         *database.DB
	httpClient *http.Client
}

func NewNotifier(db *database.DB) *Notifier {
	return &Notifier{
		db: db,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// WebhookPayload represents the payload sent to merchant webhook
type WebhookPayload struct {
	Type      string                 `json:"type"`
	Object    WebhookObject          `json:"object"`
	SecretKey string                 `json:"secret_key"`
}

type WebhookObject struct {
	UUID         string  `json:"uuid"`
	Status       string  `json:"status"`
	Amount       float64 `json:"amount"`
	IncomeAmount float64 `json:"income_amount,omitempty"`
	ExternalID   string  `json:"external_id,omitempty"`
}

// NotifyMerchant sends a webhook notification to merchant when transaction status changes to final
func (n *Notifier) NotifyMerchant(ctx context.Context, txID uuid.UUID, newStatus models.TransactionStatus) error {
	// Only notify on final statuses
	if !isFinalStatus(newStatus) {
		return nil
	}

	// Get transaction and casino details
	var casinoID uuid.UUID
	var casinoSecretKey *string
	var webhookURL *string
	var amount float64
	var externalID *string

	err := n.db.Pool.QueryRow(ctx, `
		SELECT t.casino_id, t.amount, t.external_id, c.secret_key, c.webhook_url
		FROM transactions t
		JOIN casinos c ON c.id = t.casino_id
		WHERE t.id = $1
	`, txID).Scan(&casinoID, &amount, &externalID, &casinoSecretKey, &webhookURL)

	if err != nil {
		return fmt.Errorf("fetch transaction details: %w", err)
	}

	// Skip if webhook URL is not configured
	if webhookURL == nil || *webhookURL == "" {
		fmt.Printf("[INFO] No webhook URL configured for casino %s, skipping notification\n", casinoID)
		return nil
	}

	// Skip if secret key is not configured
	if casinoSecretKey == nil || *casinoSecretKey == "" {
		fmt.Printf("[WARN] No secret key configured for casino %s, cannot sign webhook\n", casinoID)
		return nil
	}

	// Build webhook payload
	eventType := statusToEventType(newStatus)
	payload := WebhookPayload{
		Type: eventType,
		Object: WebhookObject{
			UUID:         txID.String(),
			Status:       string(newStatus),
			Amount:       amount,
			IncomeAmount: amount, // For now, same as amount
		},
		SecretKey: *casinoSecretKey,
	}

	if externalID != nil {
		payload.Object.ExternalID = *externalID
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// Generate signature: HMAC-SHA256(timestamp + "." + trade_id + "." + raw_body)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	dataToSign := timestamp + "." + txID.String() + "." + string(payloadBytes)
	mac := hmac.New(sha256.New, []byte(*casinoSecretKey))
	mac.Write([]byte(dataToSign))
	signature := hex.EncodeToString(mac.Sum(nil))

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", *webhookURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Major-Timestamp", timestamp)
	req.Header.Set("X-Major-Signature", signature)

	// Send request
	startTime := time.Now()
	resp, err := n.httpClient.Do(req)
	durationMs := time.Since(startTime).Milliseconds()

	// Log webhook attempt
	logDetails := map[string]interface{}{
		"webhook_url":  *webhookURL,
		"event_type":   eventType,
		"status_code":  0,
		"duration_ms":  durationMs,
		"transaction_id": txID.String(),
	}

	if err != nil {
		logDetails["error"] = err.Error()
		logJSON, _ := json.Marshal(logDetails)
		_, _ = n.db.Pool.Exec(ctx, `
			INSERT INTO audit_logs (action, entity_type, entity_id, details)
			VALUES ('WEBHOOK_FAILED', 'transaction', $1, $2)
		`, txID, string(logJSON))
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	logDetails["status_code"] = resp.StatusCode

	// Check response status
	if resp.StatusCode != http.StatusOK {
		logDetails["error"] = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
		logJSON, _ := json.Marshal(logDetails)
		_, _ = n.db.Pool.Exec(ctx, `
			INSERT INTO audit_logs (action, entity_type, entity_id, details)
			VALUES ('WEBHOOK_FAILED', 'transaction', $1, $2)
		`, txID, string(logJSON))
		return fmt.Errorf("webhook returned status %d, expected 200", resp.StatusCode)
	}

	// Log success
	logJSON, _ := json.Marshal(logDetails)
	_, _ = n.db.Pool.Exec(ctx, `
		INSERT INTO audit_logs (action, entity_type, entity_id, details)
		VALUES ('WEBHOOK_SENT', 'transaction', $1, $2)
	`, txID, string(logJSON))

	fmt.Printf("[SUCCESS] Webhook sent to %s for transaction %s, status: %d\n", *webhookURL, txID, resp.StatusCode)
	return nil
}

// isFinalStatus checks if the status is final and should trigger a webhook
func isFinalStatus(status models.TransactionStatus) bool {
	return status == models.TxStatusPaid ||
		status == models.TxStatusExpired ||
		status == models.TxStatusPayoutSuccess ||
		status == models.TxStatusPayoutError ||
		status == models.TxStatusCancelled
}

// statusToEventType maps transaction status to webhook event type
func statusToEventType(status models.TransactionStatus) string {
	switch status {
	case models.TxStatusPaid:
		return "payment.success"
	case models.TxStatusExpired:
		return "payment.expired"
	case models.TxStatusPayoutSuccess:
		return "payout.success"
	case models.TxStatusPayoutError:
		return "payout.error"
	case models.TxStatusCancelled:
		return "payment.cancelled"
	default:
		return "payment.status_changed"
	}
}
