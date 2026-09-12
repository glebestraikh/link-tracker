package commands_test

import (
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/commands"
	testutil "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/test_util"

	"github.com/stretchr/testify/require"
)

func TestCancelCommand(t *testing.T) {
	t.Parallel()

	t.Run("Success: clears state", func(t *testing.T) {
		t.Parallel()

		// Given
		mockState := &testutil.MockStateManager{}

		mockState.On("ClearState", int64(12345), int64(12345)).Return()

		cmd := commands.NewCancelCommand(mockState)
		msg := testutil.NewTestMessage("/cancel")

		// When
		result := cmd.Handle(t.Context(), msg)

		// Then
		require.Contains(t, result, "Действие отменено.")
		mockState.AssertExpectations(t)
	})
}
