package transactions

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/paymentsgate/paymentsgate/internal/middleware"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/response"
)

type Handler struct {
	service *Service
	db      *database.DB
}

func NewHandler(service *Service, db *database.DB) *Handler {
	return &Handler{service: service, db: db}
}

func (h *Handler) CreateDeposit(c *gin.Context) {
	casinoIDVal, exists := c.Get("casino_id")
	if !exists {
		response.Unauthorized(c, "casino not authenticated")
		return
	}
	casinoID := casinoIDVal.(uuid.UUID)
	isSandbox, _ := c.Get("is_sandbox")

	var req CreateDepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.CreateDeposit(c.Request.Context(), casinoID, req, isSandbox.(bool))
	if err != nil {
		response.InternalError(c, "failed to create deposit")
		return
	}
	response.Created(c, result)
}

func (h *Handler) GetStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid transaction id")
		return
	}

	tx, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.NotFound(c, "transaction not found")
			return
		}
		response.InternalError(c, "failed to get transaction")
		return
	}
	response.OK(c, tx)
}

func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	filter := ListFilter{
		Page:       page,
		PerPage:    perPage,
		Status:     c.Query("status"),
		Country:    c.Query("country"),
		CasinoID:   c.Query("casino_id"),
		ProviderID: c.Query("provider_id"),
	}

	if sandbox := c.Query("is_sandbox"); sandbox != "" {
		val := sandbox == "true"
		filter.IsSandbox = &val
	}

	txs, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "failed to list transactions")
		return
	}
	response.Paginated(c, txs, page, perPage, total)
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	deposits := rg.Group("/deposit")
	deposits.Use(middleware.CasinoAuth(h.db))
	{
		deposits.POST("/create", h.CreateDeposit)
		deposits.GET("/status/:id", h.GetStatus)
	}

	admin := rg.Group("/transactions")
	admin.Use(authMiddleware)
	{
		admin.GET("", h.List)
		admin.GET("/:id", h.GetStatus)
	}
}
