package commands

import (
	"context"
	"errors"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
)

const startDBTimeout = 5 * time.Second

type StartCommand struct {
	userRepo       service.UserRepository
	chatRepo       service.ChatRepository
	scrapperClient service.ScrapperClient
	logger         *slog.Logger
}

func NewStartCommand(
	userRepo service.UserRepository,
	chatRepo service.ChatRepository,
	scrapperClient service.ScrapperClient,
	logger *slog.Logger,
) *StartCommand {
	return &StartCommand{
		userRepo:       userRepo,
		chatRepo:       chatRepo,
		scrapperClient: scrapperClient,
		logger:         logger,
	}
}

func (c *StartCommand) Name() string {
	return "start"
}

func (c *StartCommand) Description() string {
	return "начало работы с ботом"
}

func (c *StartCommand) Handle(ctx context.Context, msg *tgbotapi.Message) string {
	userID := msg.From.ID
	chatID := msg.Chat.ID
	username := msg.From.UserName

	ctx, cancel := context.WithTimeout(ctx, startDBTimeout)
	defer cancel()

	exists, err := c.userRepo.Exists(ctx, userID)
	if err != nil {
		c.logger.Error("failed to check user existence", slog.Int64("user_id", userID), slog.Any("error", err))
		return model.ErrInternal.Error()
	}
	isNewUser := !exists

	chatExisted := false

	if err = c.scrapperClient.RegisterChat(ctx, chatID); err != nil {
		c.logger.Error("failed to register chat with scrapper", slog.Int64("user_id", userID), slog.Int64("chat_id", chatID), slog.Any("error", err))

		if errors.Is(err, model.ErrChatAlreadyRegistered) {
			chatExisted = true
		} else {
			return model.ErrScrapperUnavailable.Error()
		}
	}

	if isNewUser {
		if err = c.userRepo.Save(ctx, model.User{
			ID:       userID,
			Username: username,
		}); err != nil {
			c.logger.Error("failed to save user", slog.Int64("user_id", userID), slog.Any("error", err))
			return model.ErrInternal.Error()
		}
	}

	if err = c.chatRepo.Save(ctx, chatID); err != nil {
		c.logger.Error("failed to save chat", slog.Int64("chat_id", chatID), slog.Any("error", err))
		return model.ErrInternal.Error()
	}

	if isNewUser && !chatExisted {
		c.logger.Info("new user registered", slog.Int64("user_id", userID), slog.Int64("chat_id", chatID), slog.String("username", username))

		return "Добро пожаловать! Используйте /help, чтобы посмотреть доступные команды."
	}

	return "Добро пожаловать! Вы уже зарегистрированы! Используйте /help, чтобы посмотреть доступные команды."
}
