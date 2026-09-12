package tagging

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/service"
)

type Service struct {
	repo   service.TagRepository
	logger *slog.Logger
}

func NewService(repo service.TagRepository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) Create(ctx context.Context, name string) (*model.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, model.ErrTagNotFound
	}

	tag, err := s.repo.Create(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("create tag: %w", err)
	}

	return tag, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*model.Tag, error) {
	tag, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get tag: %w", err)
	}

	return tag, nil
}

func (s *Service) List(ctx context.Context, offset, limit uint64) ([]*model.Tag, error) {
	tags, err := s.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}

	return tags, nil
}

func (s *Service) Rename(ctx context.Context, id int64, newName string) (*model.Tag, error) {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return nil, model.ErrTagNotFound
	}

	tag, err := s.repo.Rename(ctx, id, newName)
	if err != nil {
		return nil, fmt.Errorf("rename tag: %w", err)
	}

	return tag, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	return nil
}
