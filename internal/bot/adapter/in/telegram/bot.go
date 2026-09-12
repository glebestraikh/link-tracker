package telegram

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
)

type Bot struct {
	api            *tgbotapi.BotAPI
	commandHandler service.CommandHandler
	messageHandler service.MessageHandler
	logger         *slog.Logger
	timeout        int
}

func NewBot(
	api *tgbotapi.BotAPI,
	commandHandler service.CommandHandler,
	messageHandler service.MessageHandler,
	logger *slog.Logger,
	timeout int,
) *Bot {
	return &Bot{
		api:            api,
		commandHandler: commandHandler,
		messageHandler: messageHandler,
		logger:         logger,
		timeout:        timeout,
	}
}

func (tgb *Bot) RegisterCommands() error {
	commands := tgb.commandHandler.RegisteredCommands()

	tgCommands := make([]tgbotapi.BotCommand, 0, len(commands))
	for _, cmd := range commands {
		tgCommands = append(tgCommands, tgbotapi.BotCommand{
			Command:     cmd.Name(),
			Description: cmd.Description(),
		})
	}

	cfg := tgbotapi.NewSetMyCommands(tgCommands...)
	if _, err := tgb.api.Request(cfg); err != nil {
		return fmt.Errorf("failed to register bot commands: %w", err)
	}

	tgb.logger.Info("bot commands registered", slog.Int("count", len(tgCommands)))

	return nil
}

func (tgb *Bot) Run(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = tgb.timeout

	updates := tgb.api.GetUpdatesChan(u)

	tgb.logger.Info("bot started polling for updates", slog.Int("timeout", tgb.timeout))

	for {
		select {
		case <-ctx.Done():
			tgb.logger.Info("bot context canceled, exiting run loop")
			return
		case update, ok := <-updates:
			if !ok {
				tgb.logger.Info("bot updates channel closed")
				return
			}
			if update.Message == nil {
				continue
			}

			if update.Message.IsCommand() {
				tgb.commandHandler.HandleCommand(ctx, update.Message)
			} else {
				tgb.messageHandler.HandleMessage(ctx, update.Message)
			}
		}
	}
}
