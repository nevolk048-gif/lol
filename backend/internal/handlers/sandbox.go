package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/paymentsgate/paymentsgate/internal/sandbox"
	"github.com/paymentsgate/paymentsgate/pkg/response"
)

type SandboxHandler struct {
	sandboxSvc *sandbox.Service
}

func NewSandboxHandler(sandboxSvc *sandbox.Service) *SandboxHandler {
	return &SandboxHandler{sandboxSvc: sandboxSvc}
}

// RegisterRoutes регистрирует публичные sandbox маршруты (без авторизации для тестирования)
func (h *SandboxHandler) RegisterRoutes(rg *gin.RouterGroup) {
	sandbox := rg.Group("/sandbox")
	{
		sandbox.POST("/deposit", h.CreateDeposit)
		sandbox.POST("/simulate-payment", h.SimulatePayment)
	}
}

// CreateDeposit создает тестовый депозит
func (h *SandboxHandler) CreateDeposit(c *gin.Context) {
	var req struct {
		CasinoID uuid.UUID `json:"casino_id" binding:"required"`
		Amount   float64   `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.sandboxSvc.CreateTestDeposit(c.Request.Context(), req.CasinoID, req.Amount)
	if err != nil {
		response.InternalError(c, "failed to create test deposit")
		return
	}

	response.Created(c, result)
}

// SimulatePayment симулирует оплату транзакции
func (h *SandboxHandler) SimulatePayment(c *gin.Context) {
	var req sandbox.SimulatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.sandboxSvc.SimulatePayment(c.Request.Context(), req.TransactionID); err != nil {
		if err == pgx.ErrNoRows {
			response.NotFound(c, "transaction not found")
			return
		}
		response.InternalError(c, "simulation failed")
		return
	}

	response.OK(c, gin.H{"message": "payment simulated"})
}
