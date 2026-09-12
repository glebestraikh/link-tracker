package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	grpcin "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/in/scrapper/grpc"
	httpin "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/in/scrapper/http"
	tgin "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/in/telegram"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/out/repository/pgsql"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/out/repository/squirrel"
	grpcout "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/out/scrapper/grpc"
	httpout "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/out/scrapper/http"
	tgout "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/out/telegram"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/commands"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/messages"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/state"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/updates"
	pb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/proto/bot"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"time"
)

const (
	grpcTransport = "grpc"
	accessTypeSQL = "sql"

	testUserIDStart  = 9001
	testUserIDOffset = 2

	initTimeout = 30 * time.Second
)

func Init(cfg *Config, logger *slog.Logger) (*tgin.Bot, *http.ServeMux, *grpc.Server, *pgxpool.Pool, error) {
	logger.Info("initializing bot")

	botAPI, err := newBotAPI(cfg)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	logger.Info("authorized on telegram",
		slog.String("bot_username", botAPI.Self.UserName),
	)

	pool, err := openPool(cfg.DatabaseURL)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	logger.Info("postgres pool ready", slog.String("access_type", cfg.AccessType))

	userRepo, chatRepo := buildBotRepos(cfg.AccessType, pool)

	if cfg.TelegramAPIURL != "" {
		populateTestData(context.Background(), userRepo, chatRepo, logger)
	}

	scrapperClient, err := initScrapperOutboundClient(cfg, logger)
	if err != nil {
		pool.Close()
		return nil, nil, nil, nil, err
	}

	bot, updateNotifier := newBot(cfg, botAPI, userRepo, chatRepo, scrapperClient, logger)
	logger.Info("bot successfully initialized")

	mux, grpcServer := initScrapperInboundHandlers(updateNotifier, logger)
	logger.Info("bot HTTP and gRPC routes registered")

	return bot, mux, grpcServer, pool, nil
}

func newBotAPI(cfg *Config) (*tgbotapi.BotAPI, error) {
	if cfg.TelegramAPIURL != "" {
		botAPI, err := tgbotapi.NewBotAPIWithAPIEndpoint(cfg.TelegramToken, cfg.TelegramAPIURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create bot api: %w", err)
		}
		return botAPI, nil
	}
	botAPI, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot api: %w", err)
	}
	return botAPI, nil
}

func openPool(databaseURL string) (*pgxpool.Pool, error) {
	initCtx, cancel := context.WithTimeout(context.Background(), initTimeout)
	defer cancel()

	pool, err := pgxpool.New(initCtx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres pool: %w", err)
	}
	if err = pool.Ping(initCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}
	return pool, nil
}

func newBot(
	cfg *Config,
	botAPI *tgbotapi.BotAPI,
	userRepo service.UserRepository,
	chatRepo service.ChatRepository,
	scrapperClient service.ScrapperClient,
	logger *slog.Logger,
) (*tgin.Bot, service.LinkUpdateNotifier) {
	stateManager := state.NewManager()
	sender := tgout.NewSender(botAPI)

	startCmd := commands.NewStartCommand(userRepo, chatRepo, scrapperClient, logger)
	helpCmd := commands.NewHelpCommand()
	trackCmd := commands.NewTrackCommand(stateManager, logger)
	untrackCmd := commands.NewUntrackCommand(stateManager, logger)
	listCmd := commands.NewListCommand(scrapperClient, logger)
	cancelCmd := commands.NewCancelCommand(stateManager)

	dispatcher := commands.NewDispatcher(logger, startCmd, helpCmd, trackCmd, untrackCmd, listCmd, cancelCmd)
	helpCmd.SetCommands(dispatcher.RegisteredCommands())

	commandHandler := commands.NewHandler(dispatcher, stateManager, sender, logger)
	messageHandler := messages.NewHandler(stateManager, scrapperClient, sender, logger)

	updateNotifier := updates.NewNotifier(sender, chatRepo, logger)

	bot := tgin.NewBot(botAPI, commandHandler, messageHandler, logger, cfg.UpdateTimeout)

	return bot, updateNotifier
}

func buildBotRepos(accessType string, pool *pgxpool.Pool) (service.UserRepository, service.ChatRepository) {
	if accessType != accessTypeSQL {
		return squirrel.NewUserRepository(pool), squirrel.NewChatRepository(pool)
	}
	return pgsql.NewUserRepository(pool), pgsql.NewChatRepository(pool)
}

func initScrapperOutboundClient(cfg *Config, logger *slog.Logger) (service.ScrapperClient, error) {
	if cfg.Transport == grpcTransport {
		grpcClient, err := grpcout.NewClient(cfg.ScrapperGRPCAddr, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create scrapper gRPC client: %w", err)
		}

		logger.Info("scrapper gRPC client created", slog.String("addr", cfg.ScrapperGRPCAddr))

		return grpcClient, nil
	}

	httpClient := httpout.NewClient(cfg.ScrapperURL, logger)
	logger.Info("scrapper HTTP client created", slog.String("url", cfg.ScrapperURL))

	return httpClient, nil
}

func initScrapperInboundHandlers(updateNotifier service.LinkUpdateNotifier, logger *slog.Logger) (*http.ServeMux, *grpc.Server) {
	mux := http.NewServeMux()
	scrapperHTTPHandler := httpin.NewHandler(updateNotifier, logger)
	scrapperHTTPHandler.RegisterRoutes(mux)

	grpcServer := grpc.NewServer()
	grpcHandler := grpcin.NewHandler(updateNotifier, logger)
	pb.RegisterBotServiceServer(grpcServer, grpcHandler)

	return mux, grpcServer
}

func populateTestData(ctx context.Context, userRepo service.UserRepository, chatRepo service.ChatRepository, logger *slog.Logger) {
	testUsers := []struct {
		id       int64
		username string
	}{
		{id: testUserIDStart, username: "testuser"},
		{id: testUserIDStart + 1, username: "testuser2"},
		{id: testUserIDStart + testUserIDOffset, username: "testuser3"},
	}

	for _, user := range testUsers {
		if err := userRepo.Save(ctx, model.User{
			ID:       user.id,
			Username: user.username,
		}); err != nil {
			logger.Warn("failed to seed test user", slog.Int64("user_id", user.id), slog.Any("err", err))
		}
		if err := chatRepo.Save(ctx, user.id); err != nil {
			logger.Warn("failed to seed test chat", slog.Int64("chat_id", user.id), slog.Any("err", err))
		}
	}
}
