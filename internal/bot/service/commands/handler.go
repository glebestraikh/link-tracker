package commands

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
)

type Handler struct {
	dispatcher   *Dispatcher
	stateManager service.StateManager
	sender       service.ChatSender
	logger       *slog.Logger
}

func NewHandler(
	dispatcher *Dispatcher,
	stateManager service.StateManager,
	sender service.ChatSender,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		dispatcher:   dispatcher,
		stateManager: stateManager,
		sender:       sender,
		logger:       logger,
	}
}

func (h *Handler) HandleCommand(ctx context.Context, msg *tgbotapi.Message) {
	h.stateManager.ClearState(msg.Chat.ID, msg.From.ID)

	response := h.dispatcher.Dispatch(ctx, msg)
	if response == "" {
		return
	}

	if err := h.sender.Send(msg.Chat.ID, response); err != nil {
		h.logger.Error("failed to send command response", slog.Int64("chat_id", msg.Chat.ID), slog.Any("error", err))
	}
}

func (h *Handler) RegisteredCommands() []service.Command {
	return h.dispatcher.RegisteredCommands()
}
