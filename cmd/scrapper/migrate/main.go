package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	scrapperdb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/database/scrapper"
)

const iofsDriver = "iofs"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	direction := flag.String("direction", "up", "migration direction: up or down")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	if err := run(dbURL, *direction, logger); err != nil {
		logger.Error("migration failed", slog.Any("err", err))
		os.Exit(1)
	}
}

func run(dbURL, direction string, logger *slog.Logger) error {
	src, err := openSource(scrapperdb.Migrations)
	if err != nil {
		return err
	}

	m, err := newMigrator(src, dbURL)
	if err != nil {
		return err
	}
	defer closeMigrator(m, logger)

	if err = applyDirection(m, direction); err != nil {
		return err
	}

	logger.Info("scrapper migrations done", slog.String("direction", direction))
	return nil
}

func openSource(fs embed.FS) (source.Driver, error) {
	src, err := iofs.New(fs, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded migrations: %w", err)
	}
	return src, nil
}

func newMigrator(src source.Driver, dbURL string) (*migrate.Migrate, error) {
	m, err := migrate.NewWithSourceInstance(iofsDriver, src, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to init migrate: %w", err)
	}
	return m, nil
}

func applyDirection(m *migrate.Migrate, direction string) error {
	switch direction {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("up migration failed: %w", err)
		}
	case "down":
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("down migration failed: %w", err)
		}
	default:
		return fmt.Errorf("unknown direction %q (expected up or down)", direction)
	}
	return nil
}

func closeMigrator(m *migrate.Migrate, logger *slog.Logger) {
	if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
		logger.Warn("failed to close migrator",
			slog.Any("source_err", srcErr),
			slog.Any("db_err", dbErr),
		)
	}
}
