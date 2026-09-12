//nolint:dupl // duplicate command setup and constructor wiring are intentional here
package commands

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
)

type TrackCommand struct {
	stateManager service.StateManager
	logger       *slog.Logger
}

func NewTrackCommand(stateManager service.StateManager, logger *slog.Logger) *TrackCommand {
	return &TrackCommand{stateManager: stateManager, logger: logger}
}

func (c *TrackCommand) Name() string {
	return "track"
}

func (c *TrackCommand) Description() string {
	return "начать отслеживание ссылки"
}

func (c *TrackCommand) Handle(_ context.Context, msg *tgbotapi.Message) string {
	userID := msg.From.ID
	chatID := msg.Chat.ID
	c.logger.Info("track command received", slog.Int64("user_id", userID), slog.String("command", "track"))

	c.stateManager.SetState(chatID, userID, model.UserState{
		Type: model.StateWaitingURL,
	})

	return "Введите ссылку для отслеживания:"
}
