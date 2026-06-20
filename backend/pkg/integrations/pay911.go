package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const Pay911BaseURL = "https://911pay.cc"

// Pay911Client handles integration with 911pay provider (Merchant API).
// Authentication: header "Access-Token: <merchant_id>"
type Pay911Client struct {
	baseURL    string
	merchantID string // UUID мерчанта, используется как Access-Token
	secretKey  string // Secret Key для верификации webhook-integrity
	client     *http.Client
}

// ---- Request types ----

// Pay911OrderCreateRequest — тело POST /api/merchant/order
type Pay911OrderCreateRequest struct {
	ExternalID     string  `json:"external_id"`
	Amount         int64   `json:"amount"`                    // minor units (копейки)
	MerchantID     string  `json:"merchant_id"`               // UUID мерчанта
	Currency       string  `json:"currency,omitempty"`        // "rub", "usd", …
	PaymentGateway string  `json:"payment_gateway,omitempty"` // "sberbank_rub", …
	PaymentDetail  string  `json:"payment_detail_type,omitempty"` // "card", "phone", …
	CallbackURL    string  `json:"callback_url,omitempty"`
	SuccessURL     string  `json:"success_url,omitempty"`
	FailURL        string  `json:"fail_url,omitempty"`
	IsFloating     bool    `json:"is_floating_amount,omitempty"`
}

// ---- Response types ----

// Pay911PaymentDetail — реквизиты, возвращённые в ответе на создание ордера
type Pay911PaymentDetail struct {
	Detail     string `json:"detail"`      // карта / телефон / счёт
	Initials   string `json:"initials"`    // ФИО получателя
	DetailType string `json:"detail_type"` // "card", "phone", …
	Region     string `json:"region"`
	QRCodeURL  string `json:"qr_code_url"`
	QRCodeLink string `json:"qr_code_link"`
}

// Pay911Order — объект ордера из ответа 911pay
type Pay911Order struct {
	ID              int64               `json:"id"`
	OrderID         string              `json:"order_id"`   // UUID на стороне 911pay
	ExternalID      string              `json:"external_id"`
	MerchantID      string              `json:"merchant_id"`
	Amount          string              `json:"amount"`
	Currency        string              `json:"currency"`
	Status          string              `json:"status"` // pending, success, fail, canceled, expired
	PaymentGateway  string              `json:"payment_gateway"`
	PaymentGatewayName string           `json:"payment_gateway_name"`
	PaymentLink     string              `json:"payment_link"`
	ExpiresAt       *int64              `json:"expires_at"`
	CreatedAt       *int64              `json:"created_at"`
	// Для Merchant API реквизиты лежат внутри payment_detail (через WebSocket / payment_link).
	// Для H2H API реквизиты приходят напрямую:
	PaymentDetail   *Pay911PaymentDetail `json:"payment_detail,omitempty"`
}

// Pay911OrderEnvelope — обёртка ответа
type Pay911OrderEnvelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"` // при ошибке
	Data    Pay911Order `json:"data"`
}

// Pay911WebhookPayload — колбэк от 911pay на наш callback_url.
// 911pay шлёт JSON c полем "integrity" для верификации.
type Pay911WebhookPayload struct {
	// Merchant API callback format
	OrderID    string `json:"order_id"`    // UUID ордера на стороне 911pay
	ExternalID string `json:"external_id"` // наш external_id
	Status     string `json:"status"`      // success, fail, canceled, expired
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	Integrity  string `json:"integrity"` // sha256 для верификации
}

// ---- Client ----

func NewPay911Client(merchantID, secretKey string) *Pay911Client {
	return &Pay911Client{
		baseURL:    Pay911BaseURL,
		merchantID: merchantID,
		secretKey:  secretKey,
		client:     &http.Client{Timeout: 60 * time.Second},
	}
}

// NewPay911ClientWithBase позволяет переопределить базовый URL (для тестов / staging).
func NewPay911ClientWithBase(baseURL, merchantID, secretKey string) *Pay911Client {
	if baseURL == "" {
		baseURL = Pay911BaseURL
	}
	return &Pay911Client{
		baseURL:    baseURL,
		merchantID: merchantID,
		secretKey:  secretKey,
		client:     &http.Client{Timeout: 60 * time.Second},
	}
}

// CreateOrder создаёт Merchant API payin-ордер в 911pay.
// POST https://911pay.cc/api/merchant/order
func (c *Pay911Client) CreateOrder(ctx context.Context, req Pay911OrderCreateRequest) (*Pay911Order, error) {
	// Всегда передаём merchant_id из клиента
	req.MerchantID = c.merchantID

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("pay911 marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/merchant/order", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("pay911 create request: %w", err)
	}

	c.setAuthHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("pay911 send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pay911 read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pay911 status %d: %s", resp.StatusCode, string(respBody))
	}

	var envelope Pay911OrderEnvelope
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("pay911 unmarshal (body=%s): %w", string(respBody), err)
	}

	if !envelope.Success {
		return nil, fmt.Errorf("pay911 business error: %s", envelope.Message)
	}

	return &envelope.Data, nil
}

// GetOrder получает ордер по UUID 911pay.
// GET https://911pay.cc/api/merchant/order/{order_uuid}
func (c *Pay911Client) GetOrder(ctx context.Context, orderUUID string) (*Pay911Order, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/merchant/order/"+orderUUID, nil)
	if err != nil {
		return nil, fmt.Errorf("pay911 create request: %w", err)
	}
	c.setAuthHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("pay911 send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pay911 read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pay911 status %d: %s", resp.StatusCode, string(respBody))
	}

	var envelope Pay911OrderEnvelope
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("pay911 unmarshal: %w", err)
	}
	if !envelope.Success {
		return nil, fmt.Errorf("pay911 error: %s", envelope.Message)
	}
	return &envelope.Data, nil
}

// MerchantID возвращает merchant UUID, переданный при создании клиента.
func (c *Pay911Client) MerchantID() string {
	return c.merchantID
}

// SecretKey возвращает secret key для внешней проверки подписи.
func (c *Pay911Client) SecretKey() string {
	return c.secretKey
}

// setAuthHeaders устанавливает заголовки аутентификации для 911pay.
// Merchant API использует заголовок Access-Token.
func (c *Pay911Client) setAuthHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Access-Token", c.merchantID)
}
