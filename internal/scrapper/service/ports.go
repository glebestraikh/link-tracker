package service

import (
	"context"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
)

// Tracker - implementation in internal/scrapper/service/tracking/tracker.go
type Tracker interface {
	RegisterChat(ctx context.Context, chatID int64) error
	DeleteChat(ctx context.Context, chatID int64) error
	AddLink(ctx context.Context, chatID int64, url string, tags []string) (*model.Link, error)
	RemoveLink(ctx context.Context, chatID int64, url string) (*model.Link, error)
	GetLinks(ctx context.Context, chatID int64) ([]*model.Link, error)
}

// ChatRepository - implementation in internal/scrapper/adapter/out/repository/chat.go
type ChatRepository interface {
	Save(ctx context.Context, chatID int64) error
	Delete(ctx context.Context, id int64) error
	Exists(ctx context.Context, id int64) (bool, error)
}

// LinkRepository - implementation in internal/scrapper/adapter/out/repository/link.go
type LinkRepository interface {
	Save(ctx context.Context, link model.Link) (*model.Link, error)
	Delete(ctx context.Context, chatID int64, url string) (*model.Link, error)
	FindByChatID(ctx context.Context, chatID int64, offset, limit uint64) ([]*model.Link, error)
	FindAllPaged(ctx context.Context, offset, limit uint64) ([]*model.Link, error)
	UpdateLastUpdated(ctx context.Context, id int64, t time.Time) error
}

// TagService - implementation in internal/scrapper/service/tagging/service.go
type TagService interface {
	Create(ctx context.Context, name string) (*model.Tag, error)
	GetByID(ctx context.Context, id int64) (*model.Tag, error)
	List(ctx context.Context, offset, limit uint64) ([]*model.Tag, error)
	Rename(ctx context.Context, id int64, newName string) (*model.Tag, error)
	Delete(ctx context.Context, id int64) error
}

// TagRepository - implementation in internal/scrapper/adapter/out/repository/postgres/{sql,squirrel}/tag.go
type TagRepository interface {
	Create(ctx context.Context, name string) (*model.Tag, error)
	GetByID(ctx context.Context, id int64) (*model.Tag, error)
	GetByName(ctx context.Context, name string) (*model.Tag, error)
	List(ctx context.Context, offset, limit uint64) ([]*model.Tag, error)
	Rename(ctx context.Context, id int64, newName string) (*model.Tag, error)
	Delete(ctx context.Context, id int64) error
}

// BotClient - implementation in internal/scrapper/internal/bot/adapter/in/scrapper/
type BotClient interface {
	SendUpdate(ctx context.Context, id int64, url string, description string, tgChatIDs []int64) error
}

// LinkChecker - implementation in internal/scrapper/adapter/out/github/
// and internal/scrapper/adapter/out/stackoverflow/
type LinkChecker interface {
	Supports(url string) bool
	GetLastUpdated(ctx context.Context, url string) (time.Time, error)
}
