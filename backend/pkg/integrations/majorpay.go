package integrations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// MajorPayClient handles integration with MajorPay provider
type MajorPayClient struct {
	baseURL   string
	apiKey    string
	secretKey string
	client    *http.Client
}

type MajorPayDepositRequest struct {
	Amount             int               `json:"amount"`
	MerchantCustomerID string            `json:"merchant_customer_id"`
	PaymentMethod      string            `json:"payment_method,omitempty"`
	Description        string            `json:"description,omitempty"`
	ReturnURL          string            `json:"return_url,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

type MajorPayDepositResponse struct {
	TransactionID          string        `json:"transaction_id"`
	Status                 string        `json:"status"`
	HostedPaymentPageURL   string        `json:"hosted_payment_page_url"`
	Amount                 int           `json:"amount"`
	MerchantCustomerID     string        `json:"merchant_customer_id"`
	PaymentMethod          PaymentMethod `json:"payment_method"`
	Requisite              Requisite     `json:"requisite"`
	CreatedAt              time.Time     `json:"created_at"`
	ExpiresAt              time.Time     `json:"expires_at"`
	UUID                   string        `json:"uuid"` // Provider uses 'uuid' field
	RedirectURL            string        `json:"redirect_url"`
}

type PaymentMethod struct {
	Bank  string `json:"bank"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type Requisite struct {
	BankName      string `json:"bank_name"`
	HolderName    string `json:"holder_name"`
	AccountNumber string `json:"account_number"`
}

func NewMajorPayClient(baseURL, apiKey, secretKey string) *MajorPayClient {
	return &MajorPayClient{
		baseURL:   baseURL,
		apiKey:    apiKey,
		secretKey: secretKey,
		client:    &http.Client{Timeout: 60 * time.Second},
	}
}

// CreateDeposit sends deposit request to MajorPay
func (c *MajorPayClient) CreateDeposit(ctx context.Context, req MajorPayDepositRequest) (*MajorPayDepositResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/payments", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add headers as per MajorPay documentation
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("merchant-id", c.apiKey)
	httpReq.Header.Set("merchant-secret-key", c.secretKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("provider returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Try to parse response directly (MajorPay returns data directly, not wrapped)
	var directResponse MajorPayDepositResponse
	if err := json.Unmarshal(respBody, &directResponse); err != nil {
		return nil, fmt.Errorf("unmarshal response (body: %s): %w", string(respBody), err)
	}

	// Provider uses 'uuid' field, copy it to TransactionID
	if directResponse.UUID != "" {
		directResponse.TransactionID = directResponse.UUID
	}

	return &directResponse, nil
}

// signRequest creates HMAC SHA256 signature
func (c *MajorPayClient) signRequest(body []byte) string {
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// NotifyPayment notifies provider about payment completion
func (c *MajorPayClient) NotifyPayment(ctx context.Context, transactionID uuid.UUID, status string) error {
	payload := map[string]interface{}{
		"transaction_id": transactionID,
		"status":         status,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/transaction/"+transactionID.String()+"/status", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("merchant-id", c.apiKey)
	httpReq.Header.Set("merchant-secret-key", c.secretKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("provider returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetActiveRequisites fetches list of active requisites from provider
func (c *MajorPayClient) GetActiveRequisites(ctx context.Context) ([]Requisite, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/requisites", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("merchant-id", c.apiKey)
	httpReq.Header.Set("merchant-secret-key", c.secretKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var requisites []Requisite
	if err := json.Unmarshal(respBody, &requisites); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return requisites, nil
}
