package commands_test

import (
	"errors"
	"log/slog"
	"os"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/commands"
	testutil "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/test_util"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListCommand(t *testing.T) {
	t.Parallel()

	t.Run("Success: user has active subscriptions", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockScrapper := &testutil.MockScrapperClient{}

		mockLinks := []*model.Link{
			{ID: 1, URL: "https://github.com/user/repo1", Tags: []string{}},
			{ID: 2, URL: "https://github.com/user/repo2", Tags: []string{"test_tag"}},
		}

		mockScrapper.On("GetLinks", mock.Anything, int64(12345)).Return(mockLinks, nil)

		cmd := commands.NewListCommand(mockScrapper, logger)
		msg := testutil.NewTestMessage("/list")

		// When
		result := cmd.Handle(t.Context(), msg)

		// Then
		require.Contains(t, result, "Отслеживаемые ссылки:")
		require.Contains(t, result, "github.com/user/repo1")
		require.Contains(t, result, "github.com/user/repo2")
		require.Contains(t, result, "[теги: test_tag]")
		mockScrapper.AssertExpectations(t)
	})

	t.Run("Success: user has active subscriptions with specific tag filter", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockScrapper := &testutil.MockScrapperClient{}

		mockLinks := []*model.Link{
			{ID: 1, URL: "https://github.com/user/repo1", Tags: []string{}},
			{ID: 2, URL: "https://github.com/user/repo2", Tags: []string{"important"}},
		}

		mockScrapper.On("GetLinks", mock.Anything, int64(12345)).Return(mockLinks, nil)

		cmd := commands.NewListCommand(mockScrapper, logger)
		msg := testutil.NewTestMessage("/list important")

		// When
		result := cmd.Handle(t.Context(), msg)

		// Then
		require.Contains(t, result, "github.com/user/repo2")
		require.NotContains(t, result, "github.com/user/repo1")
		mockScrapper.AssertExpectations(t)
	})

	t.Run("Success: user has no active subscriptions", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockScrapper := &testutil.MockScrapperClient{}

		mockScrapper.On("GetLinks", mock.Anything, int64(12345)).Return([]*model.Link{}, nil)

		cmd := commands.NewListCommand(mockScrapper, logger)
		msg := testutil.NewTestMessage("/list")

		// When
		result := cmd.Handle(t.Context(), msg)

		// Then
		require.Contains(t, result, "Нет отслеживаемых ссылок")
		mockScrapper.AssertExpectations(t)
	})

	t.Run("Error: scrapper unavailable", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockScrapper := &testutil.MockScrapperClient{}

		mockScrapper.On("GetLinks", mock.Anything, int64(12345)).Return(nil, errors.New("connection failed"))

		cmd := commands.NewListCommand(mockScrapper, logger)
		msg := testutil.NewTestMessage("/list")

		// When
		result := cmd.Handle(t.Context(), msg)

		// Then
		require.Contains(t, result, model.ErrScrapperUnavailable.Error())
		mockScrapper.AssertExpectations(t)
	})
}
