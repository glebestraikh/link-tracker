//go:build integration

package e2e_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	botdb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/database/bot"
	scrapperdb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/database/scrapper"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	pgUser     = "test"
	pgPassword = "test"
)

func startPostgres(ctx context.Context, net *testcontainers.DockerNetwork) (testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     pgUser,
			"POSTGRES_PASSWORD": pgPassword,
			"POSTGRES_DB":       "postgres",
		},
		Networks: []string{net.Name},
		NetworkAliases: map[string][]string{
			net.Name: {"postgres"},
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}
	mapped, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return nil, err
	}
	hostURL := fmt.Sprintf("postgres://%s:%s@%s:%s", pgUser, pgPassword, host, mapped.Port())

	if err := bootstrapDatabases(ctx, hostURL); err != nil {
		return nil, err
	}

	return container, nil
}

func bootstrapDatabases(ctx context.Context, hostURL string) error {
	adminPool, err := pgxpool.New(ctx, hostURL+"/postgres?sslmode=disable")
	if err != nil {
		return fmt.Errorf("connect admin pool: %w", err)
	}
	defer adminPool.Close()

	if _, err := adminPool.Exec(ctx, `CREATE DATABASE bot`); err != nil {
		return fmt.Errorf("create bot db: %w", err)
	}

	if _, err := adminPool.Exec(ctx, `CREATE DATABASE scrapper`); err != nil {
		return fmt.Errorf("create scrapper db: %w", err)
	}
	if err := applyMigrations(hostURL+"/bot?sslmode=disable", botdb.Migrations); err != nil {
		return fmt.Errorf("bot migrations: %w", err)
	}
	if err := applyMigrations(hostURL+"/scrapper?sslmode=disable", scrapperdb.Migrations); err != nil {
		return fmt.Errorf("scrapper migrations: %w", err)
	}
	return nil
}

func applyMigrations(dbURL string, migrations fs.FS) error {
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("open migrations: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, dbURL)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			slog.Warn("failed to close migrator",
				slog.Any("source_err", srcErr),
				slog.Any("db_err", dbErr),
			)
		}
	}()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("up migrations: %w", err)
	}
	return nil
}

func startTelegramMock(ctx context.Context, repoRoot string, net *testcontainers.DockerNetwork) (testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    repoRoot,
			Dockerfile: "test/integration/e2e/telegrammock/Dockerfile",
		},
		ExposedPorts: []string{"8080/tcp"},
		Networks:     []string{net.Name},
		NetworkAliases: map[string][]string{
			net.Name: {"telegram-mock"},
		},
		WaitingFor: wait.ForListeningPort("8080/tcp"),
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
}

func startBot(ctx context.Context, repoRoot string, net *testcontainers.DockerNetwork) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    repoRoot,
			Dockerfile: "cmd/bot/Dockerfile",
		},
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"APP_TELEGRAM_TOKEN":     "test-token",
			"APP_TELEGRAM_API_URL":   "http://telegram-mock:8080/bot%s/%s",
			"APP_UPDATE_TIMEOUT":     "1",
			"APP_HTTP_PORT":          "8080",
			"APP_GRPC_PORT":          "9090",
			"APP_TRANSPORT":          "http",
			"APP_SCRAPPER_HTTP_URL":  "http://scrapper:8080",
			"APP_SCRAPPER_GRPC_ADDR": "scrapper:9090",
			"APP_DATABASE_URL":       fmt.Sprintf("postgres://%s:%s@postgres:5432/bot?sslmode=disable", pgUser, pgPassword),
			"APP_ACCESS_TYPE":        "sql",
		},
		Networks: []string{net.Name},
		NetworkAliases: map[string][]string{
			net.Name: {"bot"},
		},
		WaitingFor: wait.ForListeningPort("8080/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		return nil, "", err
	}

	baseURL, err := containerBaseURL(ctx, container, "8080")
	if err != nil {
		container.Terminate(ctx)

		return nil, "", err
	}

	return container, baseURL, nil
}

func startScrapper(ctx context.Context, repoRoot string, net *testcontainers.DockerNetwork) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    repoRoot,
			Dockerfile: "cmd/scrapper/Dockerfile",
		},
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"APP_HTTP_PORT":         "8080",
			"APP_GRPC_PORT":         "9090",
			"APP_TRANSPORT":         "http",
			"APP_BOT_HTTP_URL":      "http://bot:8080",
			"APP_BOT_GRPC_ADDR":     "bot:9090",
			"APP_CHECK_INTERVAL":    "1h",
			"APP_GITHUB_TOKEN":      "",
			"APP_STACKOVERFLOW_KEY": "",
			"APP_DATABASE_URL":      fmt.Sprintf("postgres://%s:%s@postgres:5432/scrapper?sslmode=disable", pgUser, pgPassword),
			"APP_ACCESS_TYPE":       "sql",
		},
		Networks: []string{net.Name},
		NetworkAliases: map[string][]string{
			net.Name: {"scrapper"},
		},
		WaitingFor: wait.ForListeningPort("8080/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		return nil, "", err
	}

	baseURL, err := containerBaseURL(ctx, container, "8080")
	if err != nil {
		container.Terminate(ctx)
		return nil, "", err
	}

	return container, baseURL, nil
}

func containerBaseURL(ctx context.Context, container testcontainers.Container, port string) (string, error) {
	host, err := container.Host(ctx)
	if err != nil {
		return "", err
	}
	mapped, err := container.MappedPort(ctx, port+"/tcp")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s:%s", host, mapped.Port()), nil
}
