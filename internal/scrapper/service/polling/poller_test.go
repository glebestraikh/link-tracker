package polling

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/service"

	"github.com/stretchr/testify/mock"
)

type MockLinkRepository struct {
	mock.Mock
}

func (m *MockLinkRepository) Save(ctx context.Context, link model.Link) (*model.Link, error) {
	args := m.Called(ctx, link)
	if args.Get(0) != nil {
		if l, ok := args.Get(0).(*model.Link); ok {
			return l, args.Error(1)
		}
	}
	return nil, args.Error(1)
}

func (m *MockLinkRepository) Delete(ctx context.Context, chatID int64, url string) (*model.Link, error) {
	args := m.Called(ctx, chatID, url)
	if args.Get(0) != nil {
		if l, ok := args.Get(0).(*model.Link); ok {
			return l, args.Error(1)
		}
	}
	return nil, args.Error(1)
}

func (m *MockLinkRepository) FindByChatID(ctx context.Context, chatID int64, offset, limit uint64) ([]*model.Link, error) {
	args := m.Called(ctx, chatID, offset, limit)
	if args.Get(0) != nil {
		if links, ok := args.Get(0).([]*model.Link); ok {
			return links, args.Error(1)
		}
	}
	return nil, args.Error(1)
}

func (m *MockLinkRepository) FindAllPaged(ctx context.Context, offset, limit uint64) ([]*model.Link, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) != nil {
		if links, ok := args.Get(0).([]*model.Link); ok {
			return links, args.Error(1)
		}
	}
	return nil, args.Error(1)
}

func (m *MockLinkRepository) UpdateLastUpdated(ctx context.Context, id int64, lastUpdated time.Time) error {
	args := m.Called(ctx, id, lastUpdated)
	return args.Error(0)
}

type MockLinkChecker struct {
	mock.Mock
}

func (m *MockLinkChecker) Supports(url string) bool {
	args := m.Called(url)
	return args.Bool(0)
}

func (m *MockLinkChecker) GetLastUpdated(ctx context.Context, url string) (time.Time, error) {
	args := m.Called(ctx, url)
	if t, ok := args.Get(0).(time.Time); ok {
		return t, args.Error(1)
	}
	return time.Time{}, args.Error(1)
}

type MockBotClient struct {
	mock.Mock
}

func (m *MockBotClient) SendUpdate(ctx context.Context, linkID int64, linkURL string, message string, chatIDs []int64) error {
	args := m.Called(ctx, linkID, linkURL, message, chatIDs)
	return args.Error(0)
}

func newPoller(repo service.LinkRepository, checker service.LinkChecker, bot service.BotClient) *LinkPoller {
	return &LinkPoller{
		linkRepo:  repo,
		checkers:  []service.LinkChecker{checker},
		botClient: bot,
		logger:    slog.New(slog.NewTextHandler(os.Stdout, nil)),
		appCtx:    context.Background(),
	}
}

func expectFindAllPaged(repo *MockLinkRepository, links []*model.Link) {
	repo.On("FindAllPaged", mock.Anything, uint64(0), uint64(pollPageSize)).Return(links, nil).Once()
}

func TestPoller_checkLinks(t *testing.T) {
	t.Parallel()

	t.Run("Success: only subscribed users get updates", func(t *testing.T) {
		t.Parallel()

		mockRepo := &MockLinkRepository{}
		mockChecker := &MockLinkChecker{}
		mockBot := &MockBotClient{}

		now := time.Now()
		past := now.Add(-time.Hour)

		mockLinks := []*model.Link{
			{
				ID:          1,
				URL:         "https://github.com/user/repo",
				ChatIDs:     []int64{123, 456},
				LastUpdated: past,
			},
		}

		expectFindAllPaged(mockRepo, mockLinks)
		mockChecker.On("Supports", "https://github.com/user/repo").Return(true)
		mockChecker.On("GetLastUpdated", mock.Anything, "https://github.com/user/repo").Return(now, nil)
		mockBot.On("SendUpdate", mock.Anything, int64(1), "https://github.com/user/repo", "Обнаружено обновление", []int64{123, 456}).Return(nil)
		mockRepo.On("UpdateLastUpdated", mock.Anything, int64(1), now).Return(nil)

		newPoller(mockRepo, mockChecker, mockBot).checkLinks()

		mockRepo.AssertExpectations(t)
		mockChecker.AssertExpectations(t)
		mockBot.AssertExpectations(t)
	})

	t.Run("Error: checker fails to get last updated", func(t *testing.T) {
		t.Parallel()

		mockRepo := &MockLinkRepository{}
		mockChecker := &MockLinkChecker{}
		mockBot := &MockBotClient{}

		now := time.Now()
		past := now.Add(-time.Hour)

		mockLinks := []*model.Link{
			{
				ID:          1,
				URL:         "https://github.com/user/repo",
				ChatIDs:     []int64{123},
				LastUpdated: past,
			},
		}

		expectFindAllPaged(mockRepo, mockLinks)
		mockChecker.On("Supports", "https://github.com/user/repo").Return(true)
		mockChecker.On("GetLastUpdated", mock.Anything, "https://github.com/user/repo").Return(time.Time{}, errors.New("network error"))

		newPoller(mockRepo, mockChecker, mockBot).checkLinks()

		mockRepo.AssertExpectations(t)
		mockChecker.AssertExpectations(t)
		mockBot.AssertNotCalled(t, "SendUpdate")
	})

	t.Run("Success: no update when link hasn't changed", func(t *testing.T) {
		t.Parallel()

		mockRepo := &MockLinkRepository{}
		mockChecker := &MockLinkChecker{}
		mockBot := &MockBotClient{}

		now := time.Now()

		mockLinks := []*model.Link{
			{ID: 1, URL: "https://github.com/user/repo", ChatIDs: []int64{123}, LastUpdated: now},
		}

		expectFindAllPaged(mockRepo, mockLinks)
		mockChecker.On("Supports", "https://github.com/user/repo").Return(true)
		mockChecker.On("GetLastUpdated", mock.Anything, "https://github.com/user/repo").Return(now.Add(-time.Minute), nil)

		newPoller(mockRepo, mockChecker, mockBot).checkLinks()

		mockRepo.AssertExpectations(t)
		mockChecker.AssertExpectations(t)
		mockBot.AssertNotCalled(t, "SendUpdate")
	})

	t.Run("Success: skips checker if it doesn't support URL", func(t *testing.T) {
		t.Parallel()

		mockRepo := &MockLinkRepository{}
		mockChecker := &MockLinkChecker{}
		mockBot := &MockBotClient{}

		mockLinks := []*model.Link{
			{ID: 1, URL: "https://example.com/some-page", ChatIDs: []int64{123}, LastUpdated: time.Now()},
		}

		expectFindAllPaged(mockRepo, mockLinks)
		mockChecker.On("Supports", "https://example.com/some-page").Return(false)

		newPoller(mockRepo, mockChecker, mockBot).checkLinks()

		mockRepo.AssertExpectations(t)
		mockChecker.AssertExpectations(t)
		mockChecker.AssertNotCalled(t, "GetLastUpdated")
		mockBot.AssertNotCalled(t, "SendUpdate")
	})
}
