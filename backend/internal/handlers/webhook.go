package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/paymentsgate/paymentsgate/internal/transactions"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/models"
)

type WebhookHandler struct {
	db     *database.DB
	txSvc  *transactions.Service
}

func NewWebhookHandler(db *database.DB, txSvc *transactions.Service) *WebhookHandler {
	return &WebhookHandler{db: db, txSvc: txSvc}
}

func (h *WebhookHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/majorpay", h.MajorPayWebhook)
}

type MajorPayWebhookPayload struct {
	Type      string `json:"type"`
	Object    struct {
		UUID         string  `json:"uuid"`
		Status       string  `json:"status"`
		Amount       int     `json:"amount"`
		IncomeAmount int     `json:"income_amount"`
	} `json:"object"`
	SecretKey string `json:"secret_key"`
}

func (h *WebhookHandler) MajorPayWebhook(c *gin.Context) {
	// Read raw body for signature verification
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Get headers
	timestamp := c.GetHeader("X-Major-Timestamp")
	signature := c.GetHeader("X-Major-Signature")

	if timestamp == "" || signature == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing signature headers"})
		return
	}

	// Parse payload
	var payload MajorPayWebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	// Log incoming webhook for debugging
	fmt.Printf("[WEBHOOK] Received type=%s, provider_tx_id=%s\n", payload.Type, payload.Object.UUID)

	// Get provider secret key from database
	var providerSecretKey string
	err = h.db.Pool.QueryRow(c.Request.Context(), `
		SELECT secret_key FROM providers
		WHERE name = 'MajorPay' AND status = 'ACTIVE'
		LIMIT 1
	`).Scan(&providerSecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "provider not found"})
		return
	}

	// Verify signature: HMAC-SHA256(timestamp + "." + trade_id + "." + raw_body)
	dataToSign := timestamp + "." + payload.Object.UUID + "." + string(rawBody)
	mac := hmac.New(sha256.New, []byte(providerSecretKey))
	mac.Write([]byte(dataToSign))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	// Find transaction by provider transaction ID
	var txID uuid.UUID
	err = h.db.Pool.QueryRow(c.Request.Context(), `
		SELECT id FROM transactions
		WHERE provider_transaction_id = $1
		LIMIT 1
	`, payload.Object.UUID).Scan(&txID)

	if err != nil {
		// Transaction not found - log but return 200 OK to avoid retries
		fmt.Printf("[WARN] Webhook received for unknown transaction: provider_tx_id=%s, error=%v\n", payload.Object.UUID, err)

		// Also try to find ANY transaction to help debug
		var count int
		_ = h.db.Pool.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM transactions`).Scan(&count)
		fmt.Printf("[DEBUG] Total transactions in DB: %d\n", count)

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	fmt.Printf("[SUCCESS] Found transaction %s for provider_tx_id=%s\n", txID, payload.Object.UUID)

	// Map MajorPay status to our status
	var newStatus models.TransactionStatus
	switch payload.Type {
	case "payment.success":
		newStatus = models.TxStatusPaid
	case "payment.expired":
		newStatus = models.TxStatusExpired
	case "payout.success":
		newStatus = models.TxStatusPayoutSuccess
	case "payout.error":
		newStatus = models.TxStatusPayoutError
	default:
		// Unknown event type - log and return OK
		fmt.Printf("[WARN] Unknown webhook type: %s\n", payload.Type)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Update transaction status
	if err := h.txSvc.UpdateStatus(c.Request.Context(), txID, newStatus); err != nil {
		fmt.Printf("[ERROR] Failed to update transaction %s: %v\n", txID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	// Log webhook received
	_, _ = h.db.Pool.Exec(c.Request.Context(), `
		INSERT INTO audit_logs (action, entity_type, entity_id, details)
		VALUES ('WEBHOOK_RECEIVED', 'transaction', $1, $2)
	`, txID, fmt.Sprintf(`{"type":"%s","provider_tx_id":"%s","status":"%s"}`,
		payload.Type, payload.Object.UUID, newStatus))

	fmt.Printf("[SUCCESS] Webhook processed: %s -> %s\n", txID, newStatus)

	// MUST return 200 OK
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
