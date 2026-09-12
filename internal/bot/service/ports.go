package service

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
)

// Command - implementation in service/commands/
type Command interface {
	Name() string
	Description() string
	Handle(ctx context.Context, msg *tgbotapi.Message) string
}

// MessageHandler - implementation in service/messages/handler.go
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *tgbotapi.Message)
}

// CommandHandler - implementation in service/commands/handler.go
type CommandHandler interface {
	HandleCommand(ctx context.Context, msg *tgbotapi.Message)
	RegisteredCommands() []Command
}

// StateManager - implementation in service/state/manager.go
type StateManager interface {
	GetState(chatID int64, userID int64) model.UserState
	SetState(chatID int64, userID int64, state model.UserState)
	ClearState(chatID int64, userID int64)
}

// LinkUpdateNotifier - implementation in service/updates/notifier.go
type LinkUpdateNotifier interface {
	Notify(update model.LinkUpdate) error
}

// UserRepository - implementation in adapter/out/repository/user.go
type UserRepository interface {
	Save(ctx context.Context, user model.User) error
	Exists(ctx context.Context, id int64) (bool, error)
}

// ChatRepository - implementation in adapter/out/repository/chat.go
type ChatRepository interface {
	Save(ctx context.Context, chatID int64) error
	Exists(ctx context.Context, chatID int64) (bool, error)
}

// ScrapperClient - implementation in adapter/out/scrapper/
type ScrapperClient interface {
	RegisterChat(ctx context.Context, chatID int64) error
	DeleteChat(ctx context.Context, chatID int64) error
	AddLink(ctx context.Context, chatID int64, url string, tags []string) (*model.Link, error)
	RemoveLink(ctx context.Context, chatID int64, url string) (*model.Link, error)
	GetLinks(ctx context.Context, chatID int64) ([]*model.Link, error)
}

// ChatSender - implementation in adapter/out/telegram/sender.go
type ChatSender interface {
	Send(chatID int64, text string) error
}
