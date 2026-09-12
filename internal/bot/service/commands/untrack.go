//nolint:dupl // duplicate command setup and constructor wiring are intentional here
package commands

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
)

type UntrackCommand struct {
	stateManager service.StateManager
	logger       *slog.Logger
}

func NewUntrackCommand(stateManager service.StateManager, logger *slog.Logger) *UntrackCommand {
	return &UntrackCommand{stateManager: stateManager, logger: logger}
}

func (c *UntrackCommand) Name() string {
	return "untrack"
}

func (c *UntrackCommand) Description() string {
	return "прекратить отслеживание ссылки"
}

func (c *UntrackCommand) Handle(_ context.Context, msg *tgbotapi.Message) string {
	userID := msg.From.ID
	chatID := msg.Chat.ID
	c.logger.Info("untrack command received", slog.Int64("user_id", userID), slog.String("command", "untrack"))

	c.stateManager.SetState(chatID, userID, model.UserState{
		Type: model.StateWaitingUntrackURL},
	)

	return "Введите ссылку для удаления из отслеживания:"
}
