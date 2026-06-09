package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/paymentsgate/paymentsgate/config"
	"github.com/paymentsgate/paymentsgate/internal/analytics"
	"github.com/paymentsgate/paymentsgate/internal/audit"
	"github.com/paymentsgate/paymentsgate/internal/auth"
	"github.com/paymentsgate/paymentsgate/internal/casinos"
	"github.com/paymentsgate/paymentsgate/internal/disputes"
	"github.com/paymentsgate/paymentsgate/internal/handlers"
	"github.com/paymentsgate/paymentsgate/internal/middleware"
	"github.com/paymentsgate/paymentsgate/internal/providers"
	"github.com/paymentsgate/paymentsgate/internal/requisites"
	"github.com/paymentsgate/paymentsgate/internal/routing"
	"github.com/paymentsgate/paymentsgate/internal/sandbox"
	"github.com/paymentsgate/paymentsgate/internal/scheduler"
	"github.com/paymentsgate/paymentsgate/internal/traffic"
	"github.com/paymentsgate/paymentsgate/internal/transactions"
	"github.com/paymentsgate/paymentsgate/internal/users"
	"github.com/paymentsgate/paymentsgate/internal/websocket"
	"github.com/paymentsgate/paymentsgate/pkg/database"
	jwtpkg "github.com/paymentsgate/paymentsgate/pkg/jwt"
	redispkg "github.com/paymentsgate/paymentsgate/pkg/redis"
)

// @title PaymentsGate API
// @version 1.0
// @description Enterprise payment aggregator platform
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	ctx := context.Background()
	db, err := database.Connect(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	// Run migrations automatically
	log.Println("Running database migrations...")
	if err := runMigrations(db.Pool); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}
	log.Println("Migrations completed successfully")

	// CRITICAL FIX: Ensure provider_transaction_id column exists
	log.Println("Ensuring provider_transaction_id column exists...")
	_, err = db.Pool.Exec(ctx, `
		ALTER TABLE transactions ADD COLUMN IF NOT EXISTS provider_transaction_id VARCHAR(255);
		CREATE INDEX IF NOT EXISTS idx_transactions_provider_transaction_id
		ON transactions(provider_transaction_id)
		WHERE provider_transaction_id IS NOT NULL;
	`)
	if err != nil {
		log.Fatalf("failed to add provider_transaction_id column: %v", err)
	}
	log.Println("provider_transaction_id column verified")

	redisClient, err := redispkg.Connect(ctx, cfg.Redis.URL)
	if err != nil {
		log.Printf("redis connection warning: %v", err)
	} else {
		defer redisClient.Close()
	}

	jwtManager := jwtpkg.NewManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)

	hub := websocket.NewHub()
	go hub.Run()

	router := routing.NewEngine(db)
	authSvc := auth.NewService(db, jwtManager)
	userSvc := users.NewService(db)
	providerSvc := providers.NewService(db)
	casinoSvc := casinos.NewService(db)
	requisiteSvc := requisites.NewService(db)
	ruleSvc := routing.NewRulesService(db)
	analyticsSvc := analytics.NewService(db)
	auditSvc := audit.NewService(db)
	txSvc := transactions.NewService(db, router, hub)
	sandboxSvc := sandbox.NewService(db, casinoSvc, providerSvc, requisiteSvc, ruleSvc, txSvc)
	disputeSvc := disputes.NewService(db)
	trafficSvc := traffic.NewService(db)

	authHandler := auth.NewHandler(authSvc)
	txHandler := transactions.NewHandler(txSvc, db)
	adminHandler := handlers.NewAdminHandler(
		db, userSvc, providerSvc, casinoSvc, requisiteSvc, ruleSvc,
		analyticsSvc, auditSvc, sandboxSvc, txSvc, hub,
	)
	providerHandler := handlers.NewProviderAPIHandler(txSvc, db)
	webhookHandler := handlers.NewWebhookHandler(db, txSvc, disputeSvc)
	disputeHandler := handlers.NewDisputeHandler(disputeSvc)
	trafficHandler := handlers.NewTrafficHandler(trafficSvc)
	sandboxHandler := handlers.NewSandboxHandler(sandboxSvc)

	// Запускаем scheduler для фоновых задач
	schedulerSvc := scheduler.NewScheduler(db, txSvc)
	schedulerCtx, schedulerCancel := context.WithCancel(ctx)
	defer schedulerCancel()
	schedulerSvc.Start(schedulerCtx)
	log.Println("Background scheduler started")

	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.RateLimit(cfg.Security.RateLimitRPS))
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key", "X-Signature", "Idempotency-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", handlers.HealthCheck(db))
	r.GET("/ws", hub.HandleWS(jwtManager))

	v1 := r.Group("/api/v1")
	authMiddleware := middleware.Auth(jwtManager)
	casinoAuthMiddleware := middleware.CasinoAuth(db)

	authHandler.RegisterRoutes(v1, authMiddleware)
	txHandler.RegisterRoutes(v1, authMiddleware)
	adminHandler.RegisterRoutes(v1, authMiddleware)
	providerHandler.RegisterRoutes(v1)
	disputeHandler.RegisterRoutes(v1, authMiddleware, casinoAuthMiddleware)
	trafficHandler.RegisterRoutes(v1, authMiddleware)
	sandboxHandler.RegisterRoutes(v1) // Публичные sandbox endpoints

	// Webhook routes (no auth required)
	webhookGroup := v1.Group("/webhook")
	webhookHandler.RegisterRoutes(webhookGroup)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("PaymentsGate API starting on :%s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
	log.Println("server stopped")
}
