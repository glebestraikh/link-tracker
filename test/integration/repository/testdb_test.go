//go:build integration

package repository_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	botdb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/database/bot"
	scrapperdb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/database/scrapper"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	pgUser     = "test"
	pgPassword = "test"
)

var (
	sharedHost string
	sharedPort string
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("postgres"),
		tcpostgres.WithUsername(pgUser),
		tcpostgres.WithPassword(pgPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(90*time.Second),
		),
	)
	if err != nil {
		log.Printf("failed to start postgres: %v", err)
		os.Exit(1)
	}

	host, err := container.Host(ctx)
	if err != nil {
		log.Printf("failed to get host: %v", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}
	mapped, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		log.Printf("failed to get mapped port: %v", err)
		_ = container.Terminate(ctx)
		os.Exit(1)
	}

	sharedHost = host
	sharedPort = mapped.Port()

	code := m.Run()

	termCtx, termCancel := context.WithTimeout(context.Background(), 30*time.Second)
	_ = container.Terminate(termCtx)
	termCancel()

	os.Exit(code)
}

func dbURL(database string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", pgUser, pgPassword, sharedHost, sharedPort, database)
}

func ensureDatabase(t *testing.T, name string) {
	t.Helper()
	ctx := t.Context()
	adminPool, err := pgxpool.New(ctx, dbURL("postgres"))
	require.NoError(t, err)
	defer adminPool.Close()

	var exists bool
	err = adminPool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&exists)
	require.NoError(t, err)
	if exists {
		return
	}
	_, err = adminPool.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %q`, name))
	require.NoError(t, err)
}

func applyMigrations(t *testing.T, database string, migrations fs.FS) {
	t.Helper()
	source, err := iofs.New(migrations, "migrations")
	require.NoError(t, err)

	m, err := migrate.NewWithSourceInstance("iofs", source, dbURL(database))
	require.NoError(t, err)

	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			t.Logf("close migration source: %v", srcErr)
		}
		if dbErr != nil {
			t.Logf("close migration db: %v", dbErr)
		}
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("apply migrations to %s: %v", database, err)
	}
}

func botPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ensureDatabase(t, "bot")
	applyMigrations(t, "bot", botdb.Migrations)

	pool, err := pgxpool.New(t.Context(), dbURL("bot"))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func scrapperPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ensureDatabase(t, "scrapper")
	applyMigrations(t, "scrapper", scrapperdb.Migrations)

	pool, err := pgxpool.New(t.Context(), dbURL("scrapper"))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func truncate(t *testing.T, pool *pgxpool.Pool, tables ...string) {
	t.Helper()
	stmt := "TRUNCATE TABLE " + strings.Join(tables, ", ") + " RESTART IDENTITY CASCADE"
	_, err := pool.Exec(t.Context(), stmt)
	require.NoError(t, err)
}
