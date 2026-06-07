package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/paymentsgate/paymentsgate/config"
	"github.com/paymentsgate/paymentsgate/pkg/crypto"
	"github.com/paymentsgate/paymentsgate/pkg/database"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	ctx := context.Background()
	db, err := database.Connect(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	password := "Admin123!"
	if p := os.Getenv("ADMIN_PASSWORD"); p != "" {
		password = p
	}

	hash, err := crypto.HashPassword(password)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO users (email, password_hash, role, status)
		VALUES ('admin@paymentsgate.io', $1, 'SUPER_ADMIN', 'ACTIVE')
		ON CONFLICT (email) DO UPDATE SET password_hash = $1, updated_at = NOW()
	`, hash)
	if err != nil {
		log.Fatalf("seed admin: %v", err)
	}

	fmt.Println("Admin user seeded: admin@paymentsgate.io /", password)
}
