package middleware

import (
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/paymentsgate/paymentsgate/pkg/crypto"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/models"
	"github.com/paymentsgate/paymentsgate/pkg/response"
)

func CasinoAuth(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "ApiKey ") {
				apiKey = strings.TrimPrefix(auth, "ApiKey ")
			}
		}
		if apiKey == "" {
			response.Unauthorized(c, "missing API key")
			c.Abort()
			return
		}

		var id uuid.UUID
		var status models.EntityStatus
		var ipWhitelist []string
		var isSandbox bool
		err := db.Pool.QueryRow(c.Request.Context(), `
			SELECT id, status, ip_whitelist, is_sandbox FROM casinos WHERE api_key = $1
		`, apiKey).Scan(&id, &status, &ipWhitelist, &isSandbox)
		if err != nil {
			if err == pgx.ErrNoRows {
				response.Unauthorized(c, "invalid API key")
			} else {
				response.InternalError(c, "auth error")
			}
			c.Abort()
			return
		}

		if status != models.StatusActive {
			response.Forbidden(c, "casino is not active")
			c.Abort()
			return
		}

		if len(ipWhitelist) > 0 && !isIPAllowed(c.ClientIP(), ipWhitelist) {
			response.Forbidden(c, "IP not whitelisted")
			c.Abort()
			return
		}

		c.Set("casino_id", id)
		c.Set("is_sandbox", isSandbox)
		c.Next()
	}
}

func ProviderAuth(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			response.Unauthorized(c, "missing API key")
			c.Abort()
			return
		}

		signature := c.GetHeader("X-Signature")
		var id uuid.UUID
		var secretKey string
		var status models.EntityStatus
		var ipWhitelist []string
		var isSandbox bool

		err := db.Pool.QueryRow(c.Request.Context(), `
			SELECT id, secret_key, status, ip_whitelist, is_sandbox FROM providers WHERE api_key = $1
		`, apiKey).Scan(&id, &secretKey, &status, &ipWhitelist, &isSandbox)
		if err != nil {
			response.Unauthorized(c, "invalid API key")
			c.Abort()
			return
		}

		if status != models.StatusActive {
			response.Forbidden(c, "provider is not active")
			c.Abort()
			return
		}

		if len(ipWhitelist) > 0 && !isIPAllowed(c.ClientIP(), ipWhitelist) {
			response.Forbidden(c, "IP not whitelisted")
			c.Abort()
			return
		}

		if signature != "" {
			body, _ := c.GetRawData()
			if len(body) > 0 {
				c.Request.Body = io.NopCloser(strings.NewReader(string(body)))
				if !crypto.VerifyHMAC(string(body), signature, secretKey) {
					response.Unauthorized(c, "invalid signature")
					c.Abort()
					return
				}
			}
		}

		c.Set("provider_id", id)
		c.Set("is_sandbox", isSandbox)
		c.Next()
	}
}

func isIPAllowed(clientIP string, whitelist []string) bool {
	for _, ip := range whitelist {
		if ip == clientIP || ip == "*" {
			return true
		}
	}
	return len(whitelist) == 0
}
