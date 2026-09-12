package commands_test

import (
	"log/slog"
	"os"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/out/repository/inmemory"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/commands"
	testutil "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/test_util"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStartCommand(t *testing.T) {
	t.Parallel()

	t.Run("Success: new user registers", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		userRepo := inmemory.NewUserRepository()
		chatRepo := inmemory.NewChatRepository()
		mockScrapper := &testutil.MockScrapperClient{}

		mockScrapper.On("RegisterChat", mock.Anything, int64(12345)).Return(nil)

		cmd := commands.NewStartCommand(userRepo, chatRepo, mockScrapper, logger)
		msg := testutil.NewTestMessage("/start")

		// When
		result := cmd.Handle(t.Context(), msg)

		// Then
		require.Contains(t, result, "Добро пожаловать")
		userExists, err := userRepo.Exists(t.Context(), 12345)
		require.NoError(t, err)
		chatExists, err := chatRepo.Exists(t.Context(), 12345)
		require.NoError(t, err)
		require.True(t, userExists)
		require.True(t, chatExists)
		mockScrapper.AssertExpectations(t)
	})

	t.Run("Success: existing user gets recognized", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		userRepo := inmemory.NewUserRepository()
		chatRepo := inmemory.NewChatRepository()
		mockScrapper := &testutil.MockScrapperClient{}

		require.NoError(t, userRepo.Save(t.Context(), model.User{ID: 12345, Username: "testuser"}))
		require.NoError(t, chatRepo.Save(t.Context(), 12345))

		mockScrapper.On("RegisterChat", mock.Anything, int64(12345)).Return(model.ErrChatAlreadyRegistered)

		cmd := commands.NewStartCommand(userRepo, chatRepo, mockScrapper, logger)
		msg := testutil.NewTestMessage("/start")

		// When
		result := cmd.Handle(t.Context(), msg)

		// Then
		require.Contains(t, result, "Добро пожаловать")
		require.Contains(t, result, "зарегистрированы")
		mockScrapper.AssertExpectations(t)
	})
}
