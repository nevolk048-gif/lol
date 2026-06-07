package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/paymentsgate/paymentsgate/pkg/response"
)

type idempotencyCache struct {
	mu    sync.RWMutex
	store map[string]*cachedResponse
}

type cachedResponse struct {
	statusCode int
	body       interface{}
	timestamp  time.Time
}

var (
	cache = &idempotencyCache{
		store: make(map[string]*cachedResponse),
	}
	cacheTTL = 10 * time.Minute
)

func init() {
	go cleanupExpiredCache()
}

func cleanupExpiredCache() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		cache.mu.Lock()
		now := time.Now()
		for key, resp := range cache.store {
			if now.Sub(resp.timestamp) > cacheTTL {
				delete(cache.store, key)
			}
		}
		cache.mu.Unlock()
	}
}

// IdempotencyMiddleware ensures idempotent request processing
func IdempotencyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		idempotencyKey := c.GetHeader("Idempotency-Key")

		// If no key provided, use fallback: hash of critical request data
		if idempotencyKey == "" && c.Request.Method == "POST" {
			// Fallback: generate key from merchant-id + body hash for 10-minute window
			merchantID := c.GetHeader("merchant-id")
			if merchantID != "" {
				bodyHash := hashRequestBody(c)
				idempotencyKey = fmt.Sprintf("fallback_%s_%s", merchantID, bodyHash)
			}
		}

		if idempotencyKey == "" {
			c.Next()
			return
		}

		// Check if we have a cached response
		cache.mu.RLock()
		cached, exists := cache.store[idempotencyKey]
		cache.mu.RUnlock()

		if exists {
			// Return cached response
			c.JSON(cached.statusCode, cached.body)
			c.Abort()
			return
		}

		// Create a response writer wrapper to capture the response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           nil,
		}
		c.Writer = writer

		c.Next()

		// Cache the response if it was successful (2xx status)
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			cache.mu.Lock()
			cache.store[idempotencyKey] = &cachedResponse{
				statusCode: writer.Status(),
				body:       writer.body,
				timestamp:  time.Now(),
			}
			cache.mu.Unlock()
		}
	}
}

type responseWriter struct {
	gin.ResponseWriter
	body interface{}
}

func (w *responseWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	return w.ResponseWriter.WriteString(s)
}

func hashRequestBody(c *gin.Context) string {
	body, _ := c.GetRawData()
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])[:16]
}
