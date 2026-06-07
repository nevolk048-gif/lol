package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/paymentsgate/paymentsgate/internal/middleware"
	"github.com/paymentsgate/paymentsgate/pkg/jwt"
	"github.com/paymentsgate/paymentsgate/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Login godoc
// @Summary User login
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} response.APIResponse
// @Router /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.Login(c.Request.Context(), req, c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Unauthorized(c, "invalid email or password")
			return
		}
		if errors.Is(err, ErrUserBlocked) {
			response.Forbidden(c, "account is blocked")
			return
		}
		response.InternalError(c, "login failed")
		return
	}

	response.OK(c, result)
}

// Refresh godoc
// @Summary Refresh access token
// @Tags Auth
// @Accept json
// @Produce json
// @Router /api/v1/auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Unauthorized(c, "invalid refresh token")
		return
	}
	response.OK(c, result)
}

func (h *Handler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.RefreshToken != "" {
		_ = h.service.Logout(c.Request.Context(), req.RefreshToken)
	}
	response.OK(c, gin.H{"message": "logged out"})
}

func (h *Handler) Me(c *gin.Context) {
	claims := c.MustGet(middleware.ContextClaimsKey).(*jwt.Claims)
	user, err := h.service.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.OK(c, user)
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	auth := rg.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		auth.POST("/logout", h.Logout)
		auth.GET("/me", authMiddleware, h.Me)
	}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
