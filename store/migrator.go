package store

import (
	"context"
	"embed"
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migration represents a database migration
type Migration struct {
	Version string
	UpSQL   string
	DownSQL string
}

// LoadMigrations loads all migration files from the embed filesystem
func LoadMigrations() ([]Migration, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		parts := parseMigrationFilename(entry.Name())
		if parts == nil {
			continue
		}

		version := parts[0]
		direction := parts[1]

		migrationPath := filepath.Join("migrations", entry.Name())
		content, err := migrationFS.ReadFile(migrationPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", entry.Name(), err)
		}

		// Find or create migration for this version
		var m *Migration
		for i := range migrations {
			if migrations[i].Version == version {
				m = &migrations[i]
				break
			}
		}

		if m == nil {
			m = &Migration{Version: version}
			migrations = append(migrations, *m)
		}

		if direction == "up" {
			m.UpSQL = string(content)
		} else {
			m.DownSQL = string(content)
		}
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFilename parses migration filename and returns [version, direction]
// Example: "000001_init.up.sql" -> ["000001_init", "up"]
func parseMigrationFilename(filename string) []string {
	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]
	parts := filepath.Base(base)

	// Split by last underscore
	lastUnderscore := -1
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == '_' {
			lastUnderscore = i
			break
		}
	}

	if lastUnderscore == -1 {
		return nil
	}

	return []string{
		parts[:lastUnderscore],
		parts[lastUnderscore+1:],
	}
}

// RunMigrations runs all pending migrations
func RunMigrations(ctx context.Context, conn *pgx.Conn) error {
	migrations, err := LoadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Create schema_migrations table if not exists
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get applied migrations
	rows, err := conn.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[version] = true
	}

	// Run pending migrations
	for _, migration := range migrations {
		if applied[migration.Version] {
			log.Printf("Migration %s already applied, skipping", migration.Version)
			continue
		}

		log.Printf("Applying migration %s", migration.Version)

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		_, err = tx.Exec(ctx, migration.UpSQL)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to apply migration %s: %w", migration.Version, err)
		}

		// Record migration as applied
		_, err = tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", migration.Version)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.Version, err)
		}

		log.Printf("Migration %s applied successfully", migration.Version)
	}

	log.Printf("All migrations applied successfully")
	return nil
}
