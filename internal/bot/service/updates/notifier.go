package updates

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
)

const notifyDBTimeout = 5 * time.Second

type Notifier struct {
	sender   service.ChatSender
	chatRepo service.ChatRepository
	logger   *slog.Logger
}

func NewNotifier(sender service.ChatSender, chatRepo service.ChatRepository, logger *slog.Logger) *Notifier {
	return &Notifier{
		sender:   sender,
		chatRepo: chatRepo,
		logger:   logger,
	}
}

func (n *Notifier) Notify(update model.LinkUpdate) error {
	if update.URL == "" || len(update.TgChatIDs) == 0 {
		return model.ErrInvalidUpdate
	}

	text := fmt.Sprintf("Обновление по ссылке: %s\n%s", update.URL, update.Description)
	var successCount int

	for _, chatID := range update.TgChatIDs {
		ctx, cancel := context.WithTimeout(context.Background(), notifyDBTimeout)
		exists, err := n.chatRepo.Exists(ctx, chatID)
		cancel()
		if err != nil {
			n.logger.Error("failed to check chat existence", slog.Int64("chat_id", chatID), slog.Any("error", err))
			continue
		}
		if !exists {
			n.logger.Warn("skip update for unknown chat", slog.Int64("chat_id", chatID))
			continue
		}

		if err = n.sender.Send(chatID, text); err != nil {
			n.logger.Error("failed to send update to chat", slog.Int64("chat_id", chatID), slog.Any("error", err))
			continue
		}
		successCount++
	}

	if successCount == 0 {
		return model.ErrDeliveryFailed
	}

	if successCount < len(update.TgChatIDs) {
		n.logger.Warn("partial update delivery", slog.Int("delivered", successCount), slog.Int("total", len(update.TgChatIDs)))
	}

	return nil
}
