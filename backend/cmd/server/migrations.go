package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func runMigrations(pool *pgxpool.Pool) error {
	ctx := context.Background()

	// Create migrations table if not exists
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	// Get applied migrations
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		return fmt.Errorf("query migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return err
		}
		applied[version] = true
	}

	// Read migration files
	migrationsDir := "migrations"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var migrations []struct {
		version int
		path    string
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}

		var version int
		if _, err := fmt.Sscanf(name, "%d_", &version); err != nil {
			continue
		}

		if !applied[version] {
			migrations = append(migrations, struct {
				version int
				path    string
			}{version, filepath.Join(migrationsDir, name)})
		}
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	// Apply pending migrations
	for _, m := range migrations {
		fmt.Printf("Applying migration %d...\n", m.version)
		sql, err := os.ReadFile(m.path)
		if err != nil {
			return fmt.Errorf("read migration %d: %w", m.version, err)
		}

		// Try to apply migration, but don't fail if tables already exist
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			// Check if error is "already exists" (42P07)
			if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "42P07" {
				fmt.Printf("⚠ Migration %d already applied (skipping)\n", m.version)
			} else {
				return fmt.Errorf("apply migration %d: %w", m.version, err)
			}
		} else {
			fmt.Printf("✓ Migration %d applied\n", m.version)
		}

		// Record migration regardless
		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING`, m.version); err != nil {
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}
	}

	if len(migrations) == 0 {
		fmt.Println("No pending migrations")
	}

	return nil
}
