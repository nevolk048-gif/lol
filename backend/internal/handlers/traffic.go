package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/paymentsgate/paymentsgate/internal/traffic"
	"github.com/paymentsgate/paymentsgate/pkg/response"
)

type TrafficHandler struct {
	trafficService *traffic.Service
}

func NewTrafficHandler(trafficService *traffic.Service) *TrafficHandler {
	return &TrafficHandler{trafficService: trafficService}
}

// EnableTraffic включает трафик для провайдера
func (h *TrafficHandler) EnableTraffic(c *gin.Context) {
	providerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid provider id")
		return
	}

	var userID *uuid.UUID
	if uid, exists := c.Get("user_id"); exists {
		id := uid.(uuid.UUID)
		userID = &id
	}

	err = h.trafficService.EnableTraffic(c.Request.Context(), providerID, userID)
	if err != nil {
		response.InternalError(c, "failed to enable traffic")
		return
	}

	response.OK(c, gin.H{"message": "traffic enabled"})
}

// DisableTraffic отключает трафик для провайдера
func (h *TrafficHandler) DisableTraffic(c *gin.Context) {
	providerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid provider id")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
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

	err = h.trafficService.DisableTraffic(c.Request.Context(), providerID, req.Reason, userID)
	if err != nil {
		response.InternalError(c, "failed to disable traffic")
		return
	}

	response.OK(c, gin.H{"message": "traffic disabled"})
}

// UpdateTrafficStatus обновляет статус трафика (универсальный метод)
func (h *TrafficHandler) UpdateTrafficStatus(c *gin.Context) {
	providerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid provider id")
		return
	}

	var req struct {
		Enabled bool    `json:"enabled"`
		Reason  *string `json:"reason"`
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

	if req.Enabled {
		err = h.trafficService.EnableTraffic(c.Request.Context(), providerID, userID)
	} else {
		reason := "Manually disabled"
		if req.Reason != nil {
			reason = *req.Reason
		}
		err = h.trafficService.DisableTraffic(c.Request.Context(), providerID, reason, userID)
	}

	if err != nil {
		response.InternalError(c, "failed to update traffic status")
		return
	}

	response.OK(c, gin.H{"message": "traffic status updated", "enabled": req.Enabled})
}

// BulkUpdateTraffic массово обновляет трафик для нескольких провайдеров
func (h *TrafficHandler) BulkUpdateTraffic(c *gin.Context) {
	var req struct {
		ProviderIDs []string `json:"provider_ids" binding:"required"`
		Enable      bool     `json:"enable"`
		Reason      *string  `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Парсим UUID провайдеров
	providerIDs := make([]uuid.UUID, 0, len(req.ProviderIDs))
	for _, idStr := range req.ProviderIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			response.BadRequest(c, "invalid provider id: "+idStr)
			return
		}
		providerIDs = append(providerIDs, id)
	}

	var userID *uuid.UUID
	if uid, exists := c.Get("user_id"); exists {
		id := uid.(uuid.UUID)
		userID = &id
	}

	err := h.trafficService.BulkUpdateTraffic(c.Request.Context(), traffic.BulkUpdateRequest{
		ProviderIDs: providerIDs,
		Enable:      req.Enable,
		Reason:      req.Reason,
		PerformedBy: userID,
	})

	if err != nil {
		response.InternalError(c, "failed to bulk update traffic")
		return
	}

	response.OK(c, gin.H{
		"message": "traffic updated for providers",
		"count":   len(providerIDs),
		"enabled": req.Enable,
	})
}

// GetTrafficHistory возвращает историю изменений трафика провайдера
func (h *TrafficHandler) GetTrafficHistory(c *gin.Context) {
	providerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid provider id")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	history, err := h.trafficService.GetTrafficHistory(c.Request.Context(), providerID, limit, offset)
	if err != nil {
		response.InternalError(c, "failed to get traffic history")
		return
	}

	response.OK(c, history)
}

// GetTrafficStatus возвращает текущий статус трафика провайдера
func (h *TrafficHandler) GetTrafficStatus(c *gin.Context) {
	providerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid provider id")
		return
	}

	status, err := h.trafficService.GetTrafficStatus(c.Request.Context(), providerID)
	if err != nil {
		response.InternalError(c, "failed to get traffic status")
		return
	}

	response.OK(c, status)
}

// RegisterRoutes регистрирует роуты для управления трафиком
func (h *TrafficHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	traffic := rg.Group("/traffic")
	traffic.Use(authMiddleware)
	{
		// Массовое управление
		traffic.POST("/bulk", h.BulkUpdateTraffic)

		// Управление трафиком конкретного провайдера
		traffic.PUT("/providers/:id", h.UpdateTrafficStatus)
		traffic.POST("/providers/:id/enable", h.EnableTraffic)
		traffic.POST("/providers/:id/disable", h.DisableTraffic)
		traffic.GET("/providers/:id/status", h.GetTrafficStatus)
		traffic.GET("/providers/:id/history", h.GetTrafficHistory)
	}
}
