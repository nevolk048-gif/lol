package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/paymentsgate/paymentsgate/internal/analytics"
	"github.com/paymentsgate/paymentsgate/internal/audit"
	"github.com/paymentsgate/paymentsgate/internal/casinos"
	"github.com/paymentsgate/paymentsgate/internal/middleware"
	"github.com/paymentsgate/paymentsgate/internal/providers"
	"github.com/paymentsgate/paymentsgate/internal/requisites"
	"github.com/paymentsgate/paymentsgate/internal/routing"
	"github.com/paymentsgate/paymentsgate/internal/sandbox"
	"github.com/paymentsgate/paymentsgate/internal/transactions"
	"github.com/paymentsgate/paymentsgate/internal/users"
	"github.com/paymentsgate/paymentsgate/internal/websocket"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	"github.com/paymentsgate/paymentsgate/pkg/models"
	"github.com/paymentsgate/paymentsgate/pkg/response"
)

type AdminHandler struct {
	db           *database.DB
	users        *users.Service
	providers    *providers.Service
	casinos      *casinos.Service
	requisites   *requisites.Service
	rules        *routing.RulesService
	analytics    *analytics.Service
	audit        *audit.Service
	sandbox      *sandbox.Service
	transactions *transactions.Service
	hub          *websocket.Hub
}

func NewAdminHandler(
	db *database.DB,
	userSvc *users.Service,
	providerSvc *providers.Service,
	casinoSvc *casinos.Service,
	requisiteSvc *requisites.Service,
	ruleSvc *routing.RulesService,
	analyticsSvc *analytics.Service,
	auditSvc *audit.Service,
	sandboxSvc *sandbox.Service,
	txSvc *transactions.Service,
	hub *websocket.Hub,
) *AdminHandler {
	return &AdminHandler{
		db: db, users: userSvc, providers: providerSvc, casinos: casinoSvc,
		requisites: requisiteSvc, rules: ruleSvc, analytics: analyticsSvc,
		audit: auditSvc, sandbox: sandboxSvc, transactions: txSvc, hub: hub,
	}
}

func (h *AdminHandler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	api := rg.Group("")
	api.Use(auth)
	{
		api.GET("/dashboard", h.Dashboard)
		api.GET("/finance", h.Finance)
		api.GET("/monitoring", h.Monitoring)

		api.GET("/users", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.ListUsers)
		api.POST("/users", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.CreateUser)
		api.PATCH("/users/:id/role", middleware.RequireRoles(models.RoleSuperAdmin), h.UpdateUserRole)
		api.PATCH("/users/:id/status", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.UpdateUserStatus)
		api.DELETE("/users/:id", middleware.RequireRoles(models.RoleSuperAdmin), h.DeleteUser)

		api.GET("/providers", h.ListProviders)
		api.POST("/providers", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.CreateProvider)
		api.GET("/providers/:id", h.GetProvider)
		api.PATCH("/providers/:id/status", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.UpdateProviderStatus)
		api.DELETE("/providers/:id", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.DeleteProvider)

		api.GET("/casinos", h.ListCasinos)
		api.POST("/casinos", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.CreateCasino)
		api.GET("/casinos/:id", h.GetCasino)
		api.POST("/casinos/:id/regenerate-key", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.RegenerateCasinoKey)
		api.DELETE("/casinos/:id", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.DeleteCasino)

		api.GET("/requisites", h.ListRequisites)
		api.POST("/requisites", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.CreateRequisite)
		api.PATCH("/requisites/:id/status", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.UpdateRequisiteStatus)

		api.GET("/routing/rules", h.ListRouteRules)
		api.POST("/routing/rules", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.CreateRouteRule)
		api.PUT("/routing/rules/:id", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.UpdateRouteRule)
		api.DELETE("/routing/rules/:id", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.DeleteRouteRule)

		api.GET("/audit-logs", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.ListAuditLogs)
		api.GET("/integration-logs", h.ListIntegrationLogs)
	}

	// Debug endpoints (public for troubleshooting)
	rg.GET("/debug/last-audit-logs", h.DebugLastAuditLogs)

		sandboxGroup := api.Group("/sandbox")
		sandboxGroup.Use(middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin))
		{
			sandboxGroup.POST("/setup", h.SandboxSetup)
			sandboxGroup.POST("/deposit", h.SandboxDeposit)
			sandboxGroup.POST("/simulate-payment", h.SandboxSimulatePayment)
			sandboxGroup.POST("/generate-traffic", h.SandboxGenerateTraffic)
			sandboxGroup.POST("/generate-stats", h.SandboxGenerateStats)
		}

		api.POST("/test-transaction", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleAdmin), h.CreateTestTransaction)

		api.POST("/migrate", middleware.RequireRoles(models.RoleSuperAdmin), func(c *gin.Context) {
			response.OK(c, gin.H{"message": "migrations run via Railway"})
		})
	}
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	var isSandbox *bool
	if s := c.Query("is_sandbox"); s != "" {
		val := s == "true"
		isSandbox = &val
	}
	data, err := h.analytics.GetDashboard(c.Request.Context(), isSandbox)
	if err != nil {
		response.InternalError(c, "failed to get dashboard")
		return
	}
	response.OK(c, data)
}

func (h *AdminHandler) Finance(c *gin.Context) {
	data, err := h.analytics.GetFinance(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to get finance data")
		return
	}
	response.OK(c, data)
}

func (h *AdminHandler) Monitoring(c *gin.Context) {
	data, err := h.analytics.GetMonitoring(c.Request.Context(), h.hub.ClientCount())
	if err != nil {
		response.InternalError(c, "failed to get monitoring data")
		return
	}
	h.hub.Broadcast(websocket.EventMonitoring, data)
	response.OK(c, data)
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	list, err := h.users.List(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to list users")
		return
	}
	response.OK(c, list)
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req users.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, err := h.users.Create(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, "failed to create user")
		return
	}
	response.Created(c, user)
}

func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	var req struct {
		Role models.Role `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.users.UpdateRole(c.Request.Context(), id, req.Role); err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.OK(c, gin.H{"message": "role updated"})
}

func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	var req struct {
		Status models.EntityStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.users.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.OK(c, gin.H{"message": "status updated"})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	if err := h.users.Delete(c.Request.Context(), id); err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.OK(c, gin.H{"message": "user deleted"})
}

func (h *AdminHandler) ListProviders(c *gin.Context) {
	list, err := h.providers.List(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to list providers")
		return
	}
	response.OK(c, list)
}

func (h *AdminHandler) CreateProvider(c *gin.Context) {
	var req providers.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p, err := h.providers.Create(c.Request.Context(), req)
	if err != nil {
		c.Error(err) // Log the actual error
		response.InternalError(c, "failed to create provider: "+err.Error())
		return
	}
	h.hub.Broadcast(websocket.EventProviderConnected, p)
	response.Created(c, p)
}

func (h *AdminHandler) GetProvider(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	p, err := h.providers.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "provider not found")
		return
	}
	response.OK(c, p)
}

func (h *AdminHandler) UpdateProviderStatus(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	var req struct {
		Status models.EntityStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.providers.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		response.NotFound(c, "provider not found")
		return
	}
	response.OK(c, gin.H{"message": "status updated"})
}

func (h *AdminHandler) ListCasinos(c *gin.Context) {
	list, err := h.casinos.List(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to list casinos")
		return
	}
	response.OK(c, list)
}

func (h *AdminHandler) CreateCasino(c *gin.Context) {
	var req casinos.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	cas, err := h.casinos.Create(c.Request.Context(), req)
	if err != nil {
		c.Error(err) // Log the actual error
		response.InternalError(c, "failed to create casino: "+err.Error())
		return
	}
	h.hub.Broadcast(websocket.EventCasinoConnected, cas)
	response.Created(c, cas)
}

func (h *AdminHandler) GetCasino(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	cas, err := h.casinos.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "casino not found")
		return
	}
	response.OK(c, cas)
}

func (h *AdminHandler) RegenerateCasinoKey(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	key, err := h.casinos.RegenerateAPIKey(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "casino not found")
		return
	}
	response.OK(c, gin.H{"api_key": key})
}

func (h *AdminHandler) DeleteProvider(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	if err := h.providers.Delete(c.Request.Context(), id); err != nil {
		response.NotFound(c, "provider not found")
		return
	}
	response.OK(c, gin.H{"message": "provider deleted"})
}

func (h *AdminHandler) DeleteCasino(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	if err := h.casinos.Delete(c.Request.Context(), id); err != nil {
		response.NotFound(c, "casino not found")
		return
	}
	response.OK(c, gin.H{"message": "casino deleted"})
}

func (h *AdminHandler) ListRequisites(c *gin.Context) {
	var providerID *uuid.UUID
	if pid := c.Query("provider_id"); pid != "" {
		id, _ := uuid.Parse(pid)
		providerID = &id
	}
	list, err := h.requisites.List(c.Request.Context(), providerID)
	if err != nil {
		response.InternalError(c, "failed to list requisites")
		return
	}
	response.OK(c, list)
}

func (h *AdminHandler) CreateRequisite(c *gin.Context) {
	var req requisites.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	r, err := h.requisites.Create(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, "failed to create requisite")
		return
	}
	response.Created(c, r)
}

func (h *AdminHandler) UpdateRequisiteStatus(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	var req struct {
		Status models.RequisiteStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.requisites.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		response.NotFound(c, "requisite not found")
		return
	}
	response.OK(c, gin.H{"message": "status updated"})
}

func (h *AdminHandler) ListRouteRules(c *gin.Context) {
	list, err := h.rules.List(c.Request.Context())
	if err != nil {
		response.InternalError(c, "failed to list route rules")
		return
	}
	response.OK(c, list)
}

func (h *AdminHandler) CreateRouteRule(c *gin.Context) {
	var req routing.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	r, err := h.rules.Create(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, "failed to create route rule")
		return
	}
	response.Created(c, r)
}

func (h *AdminHandler) UpdateRouteRule(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	var req routing.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.rules.Update(c.Request.Context(), id, req); err != nil {
		response.NotFound(c, "route rule not found")
		return
	}
	response.OK(c, gin.H{"message": "rule updated"})
}

func (h *AdminHandler) DeleteRouteRule(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	if err := h.rules.Delete(c.Request.Context(), id); err != nil {
		response.NotFound(c, "route rule not found")
		return
	}
	response.OK(c, gin.H{"message": "rule deleted"})
}

func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	filter := audit.ListFilter{
		Page:       audit.ParsePage(c.DefaultQuery("page", "1")),
		PerPage:    audit.ParsePerPage(c.DefaultQuery("per_page", "20")),
		Action:     c.Query("action"),
		EntityType: c.Query("entity_type"),
	}
	logs, total, err := h.audit.List(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "failed to list audit logs")
		return
	}
	response.Paginated(c, logs, filter.Page, filter.PerPage, total)
}

func (h *AdminHandler) ListIntegrationLogs(c *gin.Context) {
	statusCode, _ := strconv.Atoi(c.Query("status_code"))
	filter := audit.IntegrationFilter{
		Page:       audit.ParsePage(c.DefaultQuery("page", "1")),
		PerPage:    audit.ParsePerPage(c.DefaultQuery("per_page", "20")),
		Endpoint:   c.Query("endpoint"),
		Method:     c.Query("method"),
		StatusCode: statusCode,
	}
	logs, total, err := h.audit.ListIntegrationLogs(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "failed to list integration logs")
		return
	}
	response.Paginated(c, logs, filter.Page, filter.PerPage, total)
}

func (h *AdminHandler) DebugLastAuditLogs(c *gin.Context) {
	rows, err := h.db.Pool.Query(c.Request.Context(), `
		SELECT action, entity_type, entity_id, details, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT 20
	`)
	if err != nil {
		response.InternalError(c, "failed to query audit logs")
		return
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var action, entityType, details string
		var entityID *string
		var createdAt time.Time

		if err := rows.Scan(&action, &entityType, &entityID, &details, &createdAt); err != nil {
			continue
		}

		logs = append(logs, map[string]interface{}{
			"action":      action,
			"entity_type": entityType,
			"entity_id":   entityID,
			"details":     details,
			"created_at":  createdAt,
		})
	}

	response.OK(c, logs)
}

func (h *AdminHandler) SandboxSetup(c *gin.Context) {
	data, err := h.sandbox.SetupSandbox(c.Request.Context())
	if err != nil {
		response.InternalError(c, "sandbox setup failed")
		return
	}
	response.OK(c, data)
}

func (h *AdminHandler) SandboxDeposit(c *gin.Context) {
	var req struct {
		CasinoID uuid.UUID `json:"casino_id" binding:"required"`
		Amount   float64   `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.sandbox.CreateTestDeposit(c.Request.Context(), req.CasinoID, req.Amount)
	if err != nil {
		response.InternalError(c, "failed to create test deposit")
		return
	}
	response.Created(c, result)
}

func (h *AdminHandler) SandboxSimulatePayment(c *gin.Context) {
	var req sandbox.SimulatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.sandbox.SimulatePayment(c.Request.Context(), req.TransactionID); err != nil {
		if err == pgx.ErrNoRows {
			response.NotFound(c, "transaction not found")
			return
		}
		response.InternalError(c, "simulation failed")
		return
	}
	response.OK(c, gin.H{"message": "payment simulated"})
}

func (h *AdminHandler) SandboxGenerateTraffic(c *gin.Context) {
	var req struct {
		CasinoID uuid.UUID `json:"casino_id" binding:"required"`
		Count    int       `json:"count" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ids, err := h.sandbox.GenerateTraffic(c.Request.Context(), req.CasinoID, req.Count)
	if err != nil {
		response.InternalError(c, "traffic generation failed")
		return
	}
	response.OK(c, gin.H{"transaction_ids": ids, "count": len(ids)})
}

func (h *AdminHandler) SandboxGenerateStats(c *gin.Context) {
	msg, err := h.sandbox.GenerateStats(c.Request.Context())
	if err != nil {
		response.InternalError(c, "stats generation failed")
		return
	}
	response.OK(c, gin.H{"message": msg})
}

func (h *AdminHandler) CreateTestTransaction(c *gin.Context) {
	var req struct {
		CasinoID   uuid.UUID `json:"casino_id" binding:"required"`
		ProviderID uuid.UUID `json:"provider_id"`
		Amount     float64   `json:"amount" binding:"required,gt=0"`
		Currency   string    `json:"currency" binding:"required"`
		Country    string    `json:"country" binding:"required"`
		PlayerID   string    `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	externalID := fmt.Sprintf("test_%s", uuid.New().String()[:8])

	// Create transaction using the transactions service
	tx, err := h.transactions.CreateDeposit(c.Request.Context(), req.CasinoID, transactions.CreateDepositRequest{
		Amount:     req.Amount,
		Currency:   req.Currency,
		Country:    req.Country,
		PlayerID:   &req.PlayerID,
		ExternalID: &externalID,
	}, true) // true = is_sandbox
	if err != nil {
		c.Error(err)
		response.InternalError(c, "failed to create test transaction: "+err.Error())
		return
	}

	response.Created(c, tx)
}

type ProviderAPIHandler struct {
	txSvc *transactions.Service
	db    *database.DB
}

func NewProviderAPIHandler(txSvc *transactions.Service, db *database.DB) *ProviderAPIHandler {
	return &ProviderAPIHandler{txSvc: txSvc, db: db}
}

func (h *ProviderAPIHandler) RegisterRoutes(rg *gin.RouterGroup) {
	provider := rg.Group("/provider")
	provider.Use(middleware.ProviderAuth(h.db))
	{
		provider.GET("/transaction/:id", h.GetTransaction)
		provider.POST("/transaction/:id/status", h.UpdateStatus)
	}
}

func (h *ProviderAPIHandler) GetTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	tx, err := h.txSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "transaction not found")
		return
	}
	response.OK(c, tx)
}

func (h *ProviderAPIHandler) UpdateStatus(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	var req struct {
		Status models.TransactionStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.txSvc.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		response.NotFound(c, "transaction not found")
		return
	}
	response.OK(c, gin.H{"message": "status updated"})
}

func HealthCheck(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := db.Health(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "database": "down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "paymentsgate"})
	}
}
