package tracking

import (
	"context"
	"fmt"
	"log/slog"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/service"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
)

const defaultLinksPageSize = 100

type Tracker struct {
	chatRepo service.ChatRepository
	linkRepo service.LinkRepository
	logger   *slog.Logger
}

func NewTracker(chatRepo service.ChatRepository, linkRepo service.LinkRepository, logger *slog.Logger) *Tracker {
	return &Tracker{
		chatRepo: chatRepo,
		linkRepo: linkRepo,
		logger:   logger,
	}
}

func (s *Tracker) RegisterChat(ctx context.Context, chatID int64) error {
	s.logger.Info("registering chat", slog.Int64("chat_id", chatID))

	if err := s.chatRepo.Save(ctx, chatID); err != nil {
		s.logger.Error("failed to register chat", slog.Int64("chat_id", chatID), slog.Any("error", err))
		return fmt.Errorf("register chat: %w", err)
	}

	return nil
}

func (s *Tracker) DeleteChat(ctx context.Context, chatID int64) error {
	s.logger.Info("deleting chat", slog.Int64("chat_id", chatID))
	if err := s.chatRepo.Delete(ctx, chatID); err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}
	return nil
}

func (s *Tracker) AddLink(ctx context.Context, chatID int64, url string, tags []string) (*model.Link, error) {
	s.logger.Info("adding link", slog.Int64("chat_id", chatID), slog.String("url", url))

	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("check chat exists: %w", err)
	}
	if !exists {
		return nil, model.ErrChatNotFound
	}

	link, err := s.linkRepo.Save(ctx, model.Link{
		URL:     url,
		Tags:    tags,
		ChatIDs: []int64{chatID},
	})
	if err != nil {
		return nil, fmt.Errorf("save link: %w", err)
	}

	return link, nil
}

func (s *Tracker) RemoveLink(ctx context.Context, chatID int64, url string) (*model.Link, error) {
	s.logger.Info("removing link", slog.Int64("chat_id", chatID), slog.String("url", url))

	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("check chat exists: %w", err)
	}
	if !exists {
		return nil, model.ErrChatNotFound
	}

	link, err := s.linkRepo.Delete(ctx, chatID, url)
	if err != nil {
		return nil, fmt.Errorf("delete link: %w", err)
	}

	return link, nil
}

func (s *Tracker) GetLinks(ctx context.Context, chatID int64) ([]*model.Link, error) {
	s.logger.Info("getting links", slog.Int64("chat_id", chatID))

	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("check chat exists: %w", err)
	}
	if !exists {
		return nil, model.ErrChatNotFound
	}

	links, err := s.linkRepo.FindByChatID(ctx, chatID, 0, defaultLinksPageSize)
	if err != nil {
		return nil, fmt.Errorf("find links by chat id: %w", err)
	}

	return links, nil
}
