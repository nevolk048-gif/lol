package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/paymentsgate/paymentsgate/internal/disputes"
	"github.com/paymentsgate/paymentsgate/pkg/models"
	"github.com/paymentsgate/paymentsgate/pkg/response"
)

type DisputeHandler struct {
	disputeService *disputes.Service
}

func NewDisputeHandler(disputeService *disputes.Service) *DisputeHandler {
	return &DisputeHandler{disputeService: disputeService}
}

// CreateDispute создает новый спор
func (h *DisputeHandler) CreateDispute(c *gin.Context) {
	var req struct {
		TransactionID string `json:"transaction_id" binding:"required"`
		Reason        string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	txID, err := uuid.Parse(req.TransactionID)
	if err != nil {
		response.BadRequest(c, "invalid transaction_id")
		return
	}

	// Получаем ID пользователя из контекста (если авторизован)
	var createdBy *uuid.UUID
	if userID, exists := c.Get("user_id"); exists {
		uid := userID.(uuid.UUID)
		createdBy = &uid
	}

	dispute, err := h.disputeService.CreateDispute(c.Request.Context(), disputes.CreateDisputeRequest{
		TransactionID: txID,
		Reason:        req.Reason,
		CreatedBy:     createdBy,
	})

	if err != nil {
		response.InternalError(c, "failed to create dispute")
		return
	}

	response.Created(c, dispute)
}

// ListDisputes возвращает список споров с фильтрами
func (h *DisputeHandler) ListDisputes(c *gin.Context) {
	var filter disputes.DisputeFilter

	if status := c.Query("status"); status != "" {
		s := models.DisputeStatus(status)
		filter.Status = &s
	}

	if providerID := c.Query("provider_id"); providerID != "" {
		id, err := uuid.Parse(providerID)
		if err == nil {
			filter.ProviderID = &id
		}
	}

	if casinoID := c.Query("casino_id"); casinoID != "" {
		id, err := uuid.Parse(casinoID)
		if err == nil {
			filter.CasinoID = &id
		}
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	filter.Limit = limit
	filter.Offset = offset

	disputeList, err := h.disputeService.ListDisputes(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "failed to list disputes")
		return
	}

	response.OK(c, disputeList)
}

// GetDispute возвращает детали спора
func (h *DisputeHandler) GetDispute(c *gin.Context) {
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid dispute id")
		return
	}

	dispute, err := h.disputeService.GetDispute(c.Request.Context(), disputeID)
	if err != nil {
		response.NotFound(c, "dispute not found")
		return
	}

	response.OK(c, dispute)
}

// UpdateDisputeStatus обновляет статус спора
func (h *DisputeHandler) UpdateDisputeStatus(c *gin.Context) {
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid dispute id")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var userID *uuid.UUID
	if uid, exists := c.Get("user_id"); exists {
		id := uid.(uuid.UUID)
		userID = &id
	}

	err = h.disputeService.UpdateStatus(c.Request.Context(), disputeID, models.DisputeStatus(req.Status), userID)
	if err != nil {
		response.InternalError(c, "failed to update dispute status")
		return
	}

	response.OK(c, gin.H{"message": "dispute status updated"})
}

// AddDisputeMessage добавляет сообщение в спор
func (h *DisputeHandler) AddDisputeMessage(c *gin.Context) {
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid dispute id")
		return
	}

	var req struct {
		Message     string                 `json:"message" binding:"required"`
		Attachments map[string]interface{} `json:"attachments"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Получаем ID и тип отправителя
	senderID, _ := c.Get("user_id")
	senderType := "ADMIN" // По умолчанию админ

	message, err := h.disputeService.AddMessage(c.Request.Context(), disputes.AddMessageRequest{
		DisputeID:   disputeID,
		SenderType:  senderType,
		SenderID:    senderID.(uuid.UUID),
		Message:     req.Message,
		Attachments: req.Attachments,
	})

	if err != nil {
		response.InternalError(c, "failed to add message")
		return
	}

	response.Created(c, message)
}

// GetDisputeMessages возвращает сообщения спора
func (h *DisputeHandler) GetDisputeMessages(c *gin.Context) {
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid dispute id")
		return
	}

	messages, err := h.disputeService.GetMessages(c.Request.Context(), disputeID)
	if err != nil {
		response.InternalError(c, "failed to get messages")
		return
	}

	response.OK(c, messages)
}

// GetDisputeHistory возвращает историю изменений спора
func (h *DisputeHandler) GetDisputeHistory(c *gin.Context) {
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid dispute id")
		return
	}

	history, err := h.disputeService.GetHistory(c.Request.Context(), disputeID)
	if err != nil {
		response.InternalError(c, "failed to get history")
		return
	}

	response.OK(c, history)
}

// GetDisputeStats возвращает статистику по спорам
func (h *DisputeHandler) GetDisputeStats(c *gin.Context) {
	var filter disputes.StatsFilter

	// Парсим даты если переданы
	// from и to можно добавить позже

	stats, err := h.disputeService.GetStats(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "failed to get stats")
		return
	}

	response.OK(c, stats)
}

// RegisterRoutes регистрирует роуты для споров
func (h *DisputeHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc, casinoAuth gin.HandlerFunc) {
	disputes := rg.Group("/disputes")

	// Споры доступны как админам (JWT), так и казино (API ключ)
	disputes.POST("", h.CreateDispute)
	disputes.GET("", h.ListDisputes)
	disputes.GET("/:id", h.GetDispute)

	// Обновление статуса только для админов
	adminDisputes := disputes.Group("")
	adminDisputes.Use(authMiddleware)
	{
		adminDisputes.PUT("/:id/status", h.UpdateDisputeStatus)
		adminDisputes.GET("/stats", h.GetDisputeStats)
	}

	// Сообщения доступны всем авторизованным
	disputes.POST("/:id/messages", h.AddDisputeMessage)
	disputes.GET("/:id/messages", h.GetDisputeMessages)
	disputes.GET("/:id/history", h.GetDisputeHistory)
}
