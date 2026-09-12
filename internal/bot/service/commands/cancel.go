package commands

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
)

type CancelCommand struct {
	stateManager service.StateManager
}

func NewCancelCommand(stateManager service.StateManager) *CancelCommand {
	return &CancelCommand{stateManager: stateManager}
}

func (c *CancelCommand) Name() string {
	return "cancel"
}

func (c *CancelCommand) Description() string {
	return "отменить текущее действие"
}

func (c *CancelCommand) Handle(_ context.Context, msg *tgbotapi.Message) string {
	c.stateManager.ClearState(msg.Chat.ID, msg.From.ID)

	return "Действие отменено."
}
