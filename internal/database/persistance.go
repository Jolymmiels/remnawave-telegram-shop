package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
	"log/slog"
	"os"
	"path/filepath"
	"remnawave-tg-shop-bot/internal/config"
)

type MigrationConfig struct {
	MigrationsPath string
	Direction      string
	Steps          int
}

func RunMigrations(ctx context.Context, migrationConfig *MigrationConfig, pool *pgxpool.Pool) error {
	if err := pool.Ping(ctx); err != nil {
		return err
	}

	absPath, err := filepath.Abs(migrationConfig.MigrationsPath)
	if err != nil {
		return fmt.Errorf("invalid migrations path: %w", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory does not exist: %s", absPath)
	}

	db, err := sql.Open("postgres", config.DadaBaseUrl())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = db.Close() }()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("could not create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", absPath),
		"postgres", driver,
	)
	if err != nil {
		return fmt.Errorf("migration initialization failed: %w", err)
	}

	version, dirty, verErr := m.Version()
	if verErr != nil && verErr != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", verErr)
	}

	if dirty && version == 3 {
		slog.Warn("Detected dirty migration at version 3: forcing pointer and running down script")

		if err := m.Force(int(version)); err != nil {
			return fmt.Errorf("failed to force migration version: %w", err)
		}

		if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("failed to run down migration: %w", err)
		}

		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("failed to re-apply migrations: %w", err)
		}

		slog.Info("Down + re-Up completed successfully")
		return nil
	}

	var migErr error
	switch migrationConfig.Direction {
	case "up":
		if migrationConfig.Steps > 0 {
			migErr = m.Steps(migrationConfig.Steps)
		} else {
			migErr = m.Up()
		}
	case "down":
		if migrationConfig.Steps > 0 {
			migErr = m.Steps(-migrationConfig.Steps)
		} else {
			migErr = m.Down()
		}
	case "force":
		if migrationConfig.Steps < 0 {
			return errors.New("version cannot be negative for force command")
		}
		migErr = m.Force(migrationConfig.Steps)
	default:
		v, d, dbErr := m.Version()
		if dbErr != nil && dbErr != migrate.ErrNilVersion {
			return fmt.Errorf("failed to get migration version: %w", dbErr)
		}
		slog.Info("Current migration version", "version", v, "dirty", d)
		return nil
	}

	if migErr != nil && migErr != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", migErr)
	}
	if errors.Is(migErr, migrate.ErrNoChange) {
		slog.Info("No migrations to apply")
	} else {
		slog.Info("Migrations completed successfully")
	}
	return nil
}
func GetMigrationVersion(migrationsPath string) (uint, bool, error) {
	db, err := sql.Open("postgres", config.DadaBaseUrl())
	if err != nil {
		return 0, false, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = db.Close() }()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return 0, false, fmt.Errorf("could not create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres", driver)
	if err != nil {
		return 0, false, fmt.Errorf("migration initialization failed: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	if err == migrate.ErrNilVersion {
		return 0, false, nil
	}

	return version, dirty, nil
}
