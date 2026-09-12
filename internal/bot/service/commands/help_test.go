package commands_test

import (
	"log/slog"
	"os"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/out/repository/inmemory"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/commands"
	testutil "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/test_util"

	"github.com/stretchr/testify/require"
)

func TestHelpCommand(t *testing.T) {
	t.Parallel()

	t.Run("Success: shows all registered commands", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		userRepo := inmemory.NewUserRepository()
		chatRepo := inmemory.NewChatRepository()
		mockScrapper := &testutil.MockScrapperClient{}

		startCmd := commands.NewStartCommand(userRepo, chatRepo, mockScrapper, logger)
		helpCmd := commands.NewHelpCommand()

		dispatcher := commands.NewDispatcher(logger, startCmd, helpCmd)
		helpCmd.SetCommands(dispatcher.RegisteredCommands())

		msg := testutil.NewTestMessage("/help")

		// When
		result := helpCmd.Handle(t.Context(), msg)

		// Then
		require.Contains(t, result, "Доступные команды")
		require.Contains(t, result, "/start")
		require.Contains(t, result, "/help")
	})
}
