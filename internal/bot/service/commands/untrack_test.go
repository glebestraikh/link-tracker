package commands_test

import (
	"log/slog"
	"os"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/commands"
	testutil "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/test_util"

	"github.com/stretchr/testify/require"
)

func TestUntrackCommand(t *testing.T) {
	t.Parallel()

	t.Run("Success: sets state to waiting for untrack url", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockState := &testutil.MockStateManager{}

		mockState.On("SetState", int64(12345), int64(12345), model.UserState{Type: model.StateWaitingUntrackURL}).Return()

		cmd := commands.NewUntrackCommand(mockState, logger)
		msg := testutil.NewTestMessage("/untrack")

		// When
		result := cmd.Handle(t.Context(), msg)

		// Then
		require.Contains(t, result, "Введите ссылку для удаления из отслеживания:")
		mockState.AssertExpectations(t)
	})
}
