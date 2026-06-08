package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleSuperAdmin Role = "SUPER_ADMIN"
	RoleAdmin      Role = "ADMIN"
	RoleSupport    Role = "SUPPORT"
	RoleAnalyst    Role = "ANALYST"
)

type EntityStatus string

const (
	StatusActive   EntityStatus = "ACTIVE"
	StatusInactive EntityStatus = "INACTIVE"
	StatusBlocked  EntityStatus = "BLOCKED"
)

type TransactionStatus string

const (
	TxStatusNew            TransactionStatus = "NEW"
	TxStatusAssigned       TransactionStatus = "ASSIGNED"
	TxStatusWaitingPayment TransactionStatus = "WAITING_PAYMENT"
	TxStatusPaid           TransactionStatus = "PAID"
	TxStatusExpired        TransactionStatus = "EXPIRED"
	TxStatusCancelled      TransactionStatus = "CANCELLED"
	TxStatusPayoutSuccess  TransactionStatus = "PAYOUT_SUCCESS"
	TxStatusPayoutError    TransactionStatus = "PAYOUT_ERROR"
)

type RequisiteStatus string

const (
	RequisiteActive   RequisiteStatus = "ACTIVE"
	RequisiteInactive RequisiteStatus = "INACTIVE"
	RequisiteExhausted RequisiteStatus = "EXHAUSTED"
)

type User struct {
	ID           uuid.UUID    `json:"id"`
	Email        string       `json:"email"`
	PasswordHash string       `json:"-"`
	Role         Role         `json:"role"`
	Status       EntityStatus `json:"status"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

type Casino struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	APIKey      string       `json:"api_key"`
	SecretKey   *string      `json:"-"`
	MerchantID  *string      `json:"merchant_id,omitempty"`
	BaseURL     *string      `json:"base_url,omitempty"`
	WebhookURL  *string      `json:"webhook_url,omitempty"`
	IPWhitelist []string     `json:"ip_whitelist,omitempty"`
	Status      EntityStatus `json:"status"`
	IsSandbox   bool         `json:"is_sandbox"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type Provider struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	APIKey      string       `json:"api_key"`
	SecretKey   string       `json:"-"`
	MerchantID  *string      `json:"merchant_id,omitempty"`
	BaseURL     *string      `json:"base_url,omitempty"`
	WebhookURL  *string      `json:"webhook_url,omitempty"`
	IPWhitelist []string     `json:"ip_whitelist,omitempty"`
	Status      EntityStatus `json:"status"`
	IsSandbox   bool         `json:"is_sandbox"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type Requisite struct {
	ID            uuid.UUID       `json:"id"`
	ProviderID    uuid.UUID       `json:"provider_id"`
	BankName      string          `json:"bank_name"`
	HolderName    string          `json:"holder_name"`
	AccountNumber string          `json:"account_number"`
	Currency      string          `json:"currency"`
	Country       string          `json:"country"`
	DailyLimit    float64         `json:"daily_limit"`
	UsedLimit     float64         `json:"used_limit"`
	Status        RequisiteStatus `json:"status"`
	IsSandbox     bool            `json:"is_sandbox"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type Transaction struct {
	ID              uuid.UUID         `json:"id"`
	ExternalID      *string           `json:"external_id,omitempty"`
	CasinoID        uuid.UUID         `json:"casino_id"`
	ProviderID      *uuid.UUID        `json:"provider_id,omitempty"`
	RequisiteID     *uuid.UUID        `json:"requisite_id,omitempty"`
	Amount          float64           `json:"amount"`
	Currency        string            `json:"currency"`
	Country         string            `json:"country"`
	Status          TransactionStatus `json:"status"`
	PlayerID        *string           `json:"player_id,omitempty"`
	IsSandbox       bool              `json:"is_sandbox"`
	ProcessingMs    *int64            `json:"processing_ms,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	AssignedAt      *time.Time        `json:"assigned_at,omitempty"`
	PaidAt          *time.Time        `json:"paid_at,omitempty"`
	CasinoName      string            `json:"casino_name,omitempty"`
	ProviderName    string            `json:"provider_name,omitempty"`
	RequisiteBank   string            `json:"requisite_bank,omitempty"`
}

type RouteRule struct {
	ID         uuid.UUID    `json:"id"`
	Priority   int          `json:"priority"`
	Weight     int          `json:"weight"`
	Country    *string      `json:"country,omitempty"`
	Currency   *string      `json:"currency,omitempty"`
	ProviderID uuid.UUID    `json:"provider_id"`
	Status     EntityStatus `json:"status"`
	IsFallback bool         `json:"is_fallback"`
	IsSandbox  bool         `json:"is_sandbox"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
	ProviderName string     `json:"provider_name,omitempty"`
}

type AuditLog struct {
	ID         uuid.UUID              `json:"id"`
	UserID     *uuid.UUID             `json:"user_id,omitempty"`
	Action     string                 `json:"action"`
	EntityType string                 `json:"entity_type"`
	EntityID   *uuid.UUID             `json:"entity_id,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	IPAddress  string                 `json:"ip_address"`
	CreatedAt  time.Time              `json:"created_at"`
	UserEmail  string                 `json:"user_email,omitempty"`
}

type IntegrationLog struct {
	ID            uuid.UUID  `json:"id"`
	Endpoint      string     `json:"endpoint"`
	Method        string     `json:"method"`
	StatusCode    int        `json:"status_code"`
	DurationMs    int64      `json:"duration_ms"`
	ProviderID    *uuid.UUID `json:"provider_id,omitempty"`
	CasinoID      *uuid.UUID `json:"casino_id,omitempty"`
	TransactionID *uuid.UUID `json:"transaction_id,omitempty"`
	RequestBody   *string    `json:"request_body,omitempty"`
	ResponseBody  *string    `json:"response_body,omitempty"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
	IsSandbox     bool       `json:"is_sandbox"`
	CreatedAt     time.Time  `json:"created_at"`
}

type RefreshToken struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
