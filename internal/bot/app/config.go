package app

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken    string
	TelegramAPIURL   string
	UpdateTimeout    int
	HTTPPort         string
	GRPCPort         string
	ScrapperURL      string
	ScrapperGRPCAddr string
	Transport        string
	DatabaseURL      string
	AccessType       string
}

func NewConfig() (*Config, error) {
	if err := godotenv.Load(".env.bot"); err != nil {
		slog.Warn("failed to load .env.bot file", slog.Any("err", err))
	}

	cfg := &Config{}
	var errs []string

	cfg.TelegramToken = os.Getenv("APP_TELEGRAM_TOKEN")
	if cfg.TelegramToken == "" {
		errs = append(errs, "APP_TELEGRAM_TOKEN is required")
	}

	cfg.TelegramAPIURL = os.Getenv("APP_TELEGRAM_API_URL")

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

	timeoutStr := os.Getenv("APP_UPDATE_TIMEOUT")
	if timeoutStr == "" {
		errs = append(errs, "APP_UPDATE_TIMEOUT is required")
	} else {
		parsed, err := strconv.Atoi(timeoutStr)
		if err != nil {
			errs = append(errs, fmt.Sprintf("APP_UPDATE_TIMEOUT invalid: %v", err))
		} else {
			cfg.UpdateTimeout = parsed
		}
	}

	cfg.ScrapperURL = os.Getenv("APP_SCRAPPER_HTTP_URL")
	cfg.ScrapperGRPCAddr = os.Getenv("APP_SCRAPPER_GRPC_ADDR")

	if cfg.Transport == "grpc" && cfg.ScrapperGRPCAddr == "" {
		errs = append(errs, "APP_SCRAPPER_GRPC_ADDR is required for APP_TRANSPORT=grpc")
	}
	if cfg.Transport == "http" && cfg.ScrapperURL == "" {
		errs = append(errs, "APP_SCRAPPER_HTTP_URL is required for APP_TRANSPORT=http")
	}

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
