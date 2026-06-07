package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/paymentsgate/paymentsgate/pkg/jwt"
	"github.com/paymentsgate/paymentsgate/pkg/models"
	"github.com/paymentsgate/paymentsgate/pkg/response"
	"golang.org/x/time/rate"
)

const (
	ContextUserKey   = "user"
	ContextClaimsKey = "claims"
)

func Auth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "invalid authorization format")
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateAccessToken(parts[1])
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set(ContextClaimsKey, claims)
		c.Next()
	}
}

func RequireRoles(roles ...models.Role) gin.HandlerFunc {
	roleSet := make(map[models.Role]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(c *gin.Context) {
		claimsVal, exists := c.Get(ContextClaimsKey)
		if !exists {
			response.Unauthorized(c, "unauthorized")
			c.Abort()
			return
		}

		claims := claimsVal.(*jwt.Claims)
		if !roleSet[claims.Role] {
			response.Forbidden(c, "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}

type ipLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rps      rate.Limit
	burst    int
}

func newIPLimiter(rps int) *ipLimiter {
	return &ipLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    rps * 2,
	}
}

func (l *ipLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.RLock()
	limiter, exists := l.limiters[ip]
	l.mu.RUnlock()

	if exists {
		return limiter
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if limiter, exists = l.limiters[ip]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(l.rps, l.burst)
	l.limiters[ip] = limiter
	return limiter
}

func RateLimit(rps int) gin.HandlerFunc {
	limiter := newIPLimiter(rps)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.getLimiter(ip).Allow() {
			response.Error(c, http.StatusTooManyRequests, "RATE_LIMIT", "too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		_ = time.Since(start)
	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-API-Key, X-Signature")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func IntegrationLogger(logFn func(c *gin.Context, statusCode int, durationMs int64, errMsg *string)) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Milliseconds()
		var errMsg *string
		if len(c.Errors) > 0 {
			msg := c.Errors.String()
			errMsg = &msg
		}
		logFn(c, c.Writer.Status(), duration, errMsg)
	}
}
