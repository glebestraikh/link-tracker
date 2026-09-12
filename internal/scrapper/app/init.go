package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/adapter/out/repository/pgsql"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/adapter/out/repository/squirrel"

	grpcin "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/adapter/in/bot/grpc"
	httpin "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/adapter/in/bot/http"
	grpcout "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/adapter/out/bot/grpc"
	httpout "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/adapter/out/bot/http"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/adapter/out/github"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/adapter/out/stackoverflow"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/service"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/service/polling"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/service/tracking"
	pb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/proto/scrapper"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"time"
)

const (
	accessTypeSQL = "sql"

	initTimeout = 30 * time.Second
)

func Init(cfg *Config, logger *slog.Logger) (*polling.LinkPoller, *http.ServeMux, *grpc.Server, *pgxpool.Pool, error) {
	logger.Info("initializing polling")

	// init в graceful shutdown не добавлен, не знаю, насколько нужно
	// сейчас graceful shutdown появляется в app.run() до init()
	initCtx, cancel := context.WithTimeout(context.Background(), initTimeout)
	defer cancel()

	pool, err := pgxpool.New(initCtx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to open postgres pool: %w", err)
	}
	if err = pool.Ping(initCtx); err != nil {
		pool.Close()
		return nil, nil, nil, nil, fmt.Errorf("failed to ping postgres: %w", err)
	}
	logger.Info("postgres pool ready", slog.String("access_type", cfg.AccessType))

	chatRepo, linkRepo := buildScrapperRepos(cfg.AccessType, pool)

	tracker := tracking.NewTracker(chatRepo, linkRepo, logger)

	githubChecker := github.NewClient(cfg.GithubToken, cfg.HasGithubToken, logger)
	soChecker := stackoverflow.NewClient(cfg.StackOverflowKey, cfg.HasStackOverflowKey, logger)
	checkers := []service.LinkChecker{githubChecker, soChecker}

	botClient, err := initBotOutboundClient(cfg, logger)
	if err != nil {
		pool.Close()
		return nil, nil, nil, nil, err
	}

	poller, err := polling.NewLinkPoller(linkRepo, checkers, botClient, cfg.CheckInterval, logger)
	if err != nil {
		pool.Close()
		return nil, nil, nil, nil, fmt.Errorf("failed to create polling service: %w", err)
	}
	logger.Info("polling service created", slog.String("check_interval", cfg.CheckInterval.String()))

	mux, grpcServer := initBotInboundHandlers(tracker, logger)

	logger.Info("polling HTTP and gRPC routes registered")

	return poller, mux, grpcServer, pool, nil
}

func buildScrapperRepos(accessType string, pool *pgxpool.Pool) (service.ChatRepository, service.LinkRepository) {
	if accessType != accessTypeSQL {
		return squirrel.NewChatRepository(pool), squirrel.NewLinkRepository(pool)
	}
	return pgsql.NewChatRepository(pool), pgsql.NewLinkRepository(pool)
}

const grpcTransport = "grpc"

func initBotOutboundClient(cfg *Config, logger *slog.Logger) (service.BotClient, error) {
	if cfg.Transport == grpcTransport {
		grpcBotClient, err := grpcout.NewClient(cfg.BotGRPCAddr, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create bot gRPC client: %w", err)
		}

		logger.Info("bot gRPC client created", slog.String("addr", cfg.BotGRPCAddr))

		return grpcBotClient, nil
	}

	httpClient := httpout.NewClient(cfg.BotURL, logger)
	logger.Info("bot HTTP client created", slog.String("url", cfg.BotURL))

	return httpClient, nil
}

func initBotInboundHandlers(tracker service.Tracker, logger *slog.Logger) (*http.ServeMux, *grpc.Server) {
	mux := http.NewServeMux()
	httpHandler := httpin.NewHandler(tracker, logger)
	httpHandler.RegisterRoutes(mux)

	grpcServer := grpc.NewServer()
	grpcHandler := grpcin.NewHandler(tracker, logger)
	pb.RegisterScrapperServiceServer(grpcServer, grpcHandler)

	return mux, grpcServer
}
