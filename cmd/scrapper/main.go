package main

import (
	"log/slog"
	"os"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/app"
)

func main() {
	a := app.NewApp()
	if err := a.Run(); err != nil {
		slog.Error("Failed to run scrapper", slog.Any("err", err))
		os.Exit(1)
	}
}
