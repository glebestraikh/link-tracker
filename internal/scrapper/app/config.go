package app

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort            string
	GRPCPort            string
	BotURL              string
	BotGRPCAddr         string
	GithubToken         string
	HasGithubToken      bool
	StackOverflowKey    string
	HasStackOverflowKey bool
	CheckInterval       time.Duration
	Transport           string
	DatabaseURL         string
	AccessType          string
}

func NewConfig() (*Config, error) {
	if err := godotenv.Load(".env.scrapper"); err != nil {
		slog.Warn("failed to load .env.polling file", slog.Any("err", err))
	}

	cfg := &Config{}
	var errs []string

	cfg.HTTPPort = os.Getenv("APP_HTTP_PORT")
	if cfg.HTTPPort == "" {
		errs = append(errs, "APP_HTTP_PORT is required")
	}

	cfg.GRPCPort = os.Getenv("APP_GRPC_PORT")
	if cfg.GRPCPort == "" {
		errs = append(errs, "APP_GRPC_PORT is required")
	}

	cfg.Transport = os.Getenv("APP_TRANSPORT")
	if cfg.Transport != "http" && cfg.Transport != "grpc" {
		errs = append(errs, "APP_TRANSPORT must be http or grpc")
	}

	intervalStr := os.Getenv("APP_CHECK_INTERVAL")
	if intervalStr == "" {
		errs = append(errs, "APP_CHECK_INTERVAL is required")
	} else {
		parsed, err := time.ParseDuration(intervalStr)
		if err != nil {
			errs = append(errs, fmt.Sprintf("APP_CHECK_INTERVAL invalid: %v", err.Error()))
		} else {
			cfg.CheckInterval = parsed
		}
	}

	cfg.BotURL = os.Getenv("APP_BOT_HTTP_URL")
	cfg.BotGRPCAddr = os.Getenv("APP_BOT_GRPC_ADDR")

	if cfg.Transport == "grpc" && cfg.BotGRPCAddr == "" {
		errs = append(errs, "APP_BOT_GRPC_ADDR is required for APP_TRANSPORT=grpc")
	}
	if cfg.Transport == "http" && cfg.BotURL == "" {
		errs = append(errs, "APP_BOT_HTTP_URL is required for APP_TRANSPORT=http")
	}

	cfg.GithubToken = os.Getenv("APP_GITHUB_TOKEN")
	cfg.HasGithubToken = cfg.GithubToken != ""
	cfg.StackOverflowKey = os.Getenv("APP_STACKOVERFLOW_KEY")
	cfg.HasStackOverflowKey = cfg.StackOverflowKey != ""

	cfg.DatabaseURL = os.Getenv("APP_DATABASE_URL")
	if cfg.DatabaseURL == "" {
		errs = append(errs, "APP_DATABASE_URL is required")
	}

	cfg.AccessType = strings.ToLower(os.Getenv("APP_ACCESS_TYPE"))
	if cfg.AccessType != "sql" && cfg.AccessType != "squirrel" {
		errs = append(errs, "APP_ACCESS_TYPE must be sql or squirrel")
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("config validation failed:\n  - %s", joinErrors(errs))
	}

	return cfg, nil
}

func joinErrors(errs []string) string {
	var builder strings.Builder
	builder.WriteString(errs[0])
	for _, e := range errs[1:] {
		builder.WriteString("\n  - ")
		builder.WriteString(e)
	}

	return builder.String()
}
