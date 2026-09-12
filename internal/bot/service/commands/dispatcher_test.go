package commands_test

import (
	"log/slog"
	"os"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/out/repository/inmemory"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/commands"
	testutil "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/test_util"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDispatcher(t *testing.T) {
	t.Parallel()

	setupDispatcher := func() (*commands.Dispatcher, *testutil.MockScrapperClient) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		userRepo := inmemory.NewUserRepository()
		chatRepo := inmemory.NewChatRepository()
		mockScrapper := &testutil.MockScrapperClient{}
		mockState := &testutil.MockStateManager{}

		startCmd := commands.NewStartCommand(userRepo, chatRepo, mockScrapper, logger)
		helpCmd := commands.NewHelpCommand()
		trackCmd := commands.NewTrackCommand(mockState, logger)
		untrackCmd := commands.NewUntrackCommand(mockState, logger)
		listCmd := commands.NewListCommand(mockScrapper, logger)
		cancelCmd := commands.NewCancelCommand(mockState)

		dispatcher := commands.NewDispatcher(logger, startCmd, helpCmd, trackCmd, untrackCmd, listCmd, cancelCmd)
		helpCmd.SetCommands(dispatcher.RegisteredCommands())

		return dispatcher, mockScrapper
	}

	t.Run("Success: dispatches /start", func(t *testing.T) {
		t.Parallel()

		// Given
		dispatcher, mockScrapper := setupDispatcher()
		mockScrapper.On("RegisterChat", mock.Anything, int64(12345)).Return(nil)

		msg := testutil.NewTestMessage("/start")

		// When
		result := dispatcher.Dispatch(t.Context(), msg)

		// Then
		require.Contains(t, result, "Добро пожаловать")
		mockScrapper.AssertExpectations(t)
	})

	t.Run("Success: dispatches /help", func(t *testing.T) {
		t.Parallel()

		// Given
		dispatcher, _ := setupDispatcher()

		msg := testutil.NewTestMessage("/help")

		// When
		result := dispatcher.Dispatch(t.Context(), msg)

		// Then
		require.Contains(t, result, "Доступные команды")
		require.Contains(t, result, "/start")
		require.Contains(t, result, "/help")
		require.Contains(t, result, "/track")
		require.Contains(t, result, "/untrack")
		require.Contains(t, result, "/list")
		require.Contains(t, result, "/cancel")
	})

	t.Run("Error: unknown commands", func(t *testing.T) {
		t.Parallel()

		// Given
		dispatcher, _ := setupDispatcher()

		msg := testutil.NewTestMessage("/unknown")

		// When
		result := dispatcher.Dispatch(t.Context(), msg)

		// Then
		require.Contains(t, result, "Неизвестная команда")
	})

	t.Run("Error: empty commands", func(t *testing.T) {
		t.Parallel()

		// Given
		dispatcher, _ := setupDispatcher()

		msg := testutil.NewTestMessage("/")

		// When
		result := dispatcher.Dispatch(t.Context(), msg)

		// Then
		require.Contains(t, result, "Неизвестная команда")
	})
}
