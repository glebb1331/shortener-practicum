package migrations

import (
	"context"
	"database/sql"
	"fmt"
)

type Migration struct {
	Version int
	Name    string
	Up      string
	Down    string
}

var migrations = []Migration{
	{
		Version: 1,
		Name:    "create_urls_table",
		Up: `
        CREATE TABLE IF NOT EXISTS urls (
            id TEXT PRIMARY KEY,
            original_url TEXT NOT NULL UNIQUE,
            user_id TEXT NOT NULL DEFAULT '',
            is_deleted BOOLEAN DEFAULT FALSE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at);
        CREATE INDEX IF NOT EXISTS idx_urls_user_id ON urls(user_id);
        CREATE INDEX IF NOT EXISTS idx_urls_is_deleted ON urls(is_deleted);
    `,
		Down: `
        DROP TABLE IF EXISTS urls;
    `,
	},
	{
		Version: 2,
		Name:    "add_unique_original_url",
		Up: `
		ALTER TABLE urls ADD CONSTRAINT unique_original_url UNIQUE (original_url);
		CREATE INDEX IF NOT EXISTS idx_urls_original_url ON urls(original_url);
	`,
		Down: `
		ALTER TABLE urls DROP CONSTRAINT unique_original_url;
		DROP INDEX IF EXISTS idx_urls_original_url;
	`,
	},
	{
		Version: 3,
		Name:    "add_user_id_to_urls",
		Up: `
        DO $$ 
        BEGIN 
            IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                           WHERE table_name='urls' AND column_name='user_id') THEN
                ALTER TABLE urls ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
            END IF;
        END $$;
        CREATE INDEX IF NOT EXISTS idx_urls_user_id ON urls(user_id);
    `,
	},
	{
		Version: 4,
		Name:    "add_is_deleted_to_urls",
		Up: `
        DO $$ 
        BEGIN 
            IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                           WHERE table_name='urls' AND column_name='is_deleted') THEN
                ALTER TABLE urls ADD COLUMN is_deleted BOOLEAN DEFAULT FALSE;
            END IF;
        END $$;
        CREATE INDEX IF NOT EXISTS idx_urls_is_deleted ON urls(is_deleted);
    `,
	},
}

func RunMigrations(ctx context.Context, db *sql.DB) error {
	if err := createMigrationsTable(ctx, db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	currentVersion, err := getCurrentVersion(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	for _, migration := range migrations {
		if migration.Version > currentVersion {
			if err := runMigration(ctx, db, migration); err != nil {
				return fmt.Errorf("failed to run migration %d (%s): %w", migration.Version, migration.Name, err)
			}
		}
	}

	return nil
}

func createMigrationsTable(ctx context.Context, db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := db.ExecContext(ctx, query)
	return err
}

func getCurrentVersion(ctx context.Context, db *sql.DB) (int, error) {
	var version int
	query := `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`
	err := db.QueryRowContext(ctx, query).Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

func runMigration(ctx context.Context, db *sql.DB, migration Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, migration.Up); err != nil {
		return err
	}

	query := `INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`
	if _, err := tx.ExecContext(ctx, query, migration.Version, migration.Name); err != nil {
		return err
	}

	return tx.Commit()
}
