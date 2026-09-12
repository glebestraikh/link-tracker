package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	tgin "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/in/telegram"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type App struct {
	logger     *slog.Logger
	bot        *tgin.Bot
	httpMux    *http.ServeMux
	grpcServer *grpc.Server
	pool       *pgxpool.Pool
}

func NewApp() *App {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return &App{logger: logger}
}

func (a *App) Run() error {
	a.logger.Info("application starting")

	cfg, err := NewConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err = a.init(cfg); err != nil {
		return fmt.Errorf("failed to initialize bot: %w", err)
	}

	if err = a.bot.RegisterCommands(); err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	httpServer := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: a.httpMux,
	}

	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error { return a.runHTTP(httpServer, cfg.HTTPPort) })
	group.Go(func() error { return a.runGRPC(cfg.GRPCPort) })
	group.Go(func() error { return a.runBot(ctx) })

	<-ctx.Done()

	a.logger.Info("shutting down bot")

	errShutdown := a.shutdown(httpServer)
	gErr := group.Wait()

	if a.pool != nil {
		a.pool.Close()
	}

	return errors.Join(errShutdown, gErr)
}

func (a *App) runHTTP(server *http.Server, port string) error {
	a.logger.Info("starting bot HTTP server", slog.String("port", port))

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP server failed: %w", err)
	}

	return nil
}

func (a *App) runGRPC(port string) error {
	var lc net.ListenConfig
	lis, err := lc.Listen(context.Background(), "tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen gRPC: %w", err)
	}

	a.logger.Info("starting bot gRPC server", slog.String("port", port))

	if serveErr := a.grpcServer.Serve(lis); serveErr != nil {
		return fmt.Errorf("gRPC server failed: %w", serveErr)
	}

	return nil
}

func (a *App) runBot(ctx context.Context) error {
	a.logger.Info("bot running")
	a.bot.Run(ctx)
	return nil
}

func (a *App) shutdown(httpServer *http.Server) error {
	a.grpcServer.GracefulStop()

	if err := httpServer.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("http shutdown failed: %w", err)
	}

	return nil
}

func (a *App) init(cfg *Config) error {
	bot, mux, grpcServer, pool, err := Init(cfg, a.logger)
	if err != nil {
		return err
	}

	a.bot = bot
	a.httpMux = mux
	a.grpcServer = grpcServer
	a.pool = pool

	return nil
}
