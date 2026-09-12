package messages_test

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/messages"
	testutil "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service/test_util"

	"github.com/stretchr/testify/mock"
)

func TestHandler_HandleMessage_StateWaitingURL(t *testing.T) {
	t.Parallel()

	t.Run("Success: valid url transitions to waiting tags", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockState := &testutil.MockStateManager{}
		mockScrapper := &testutil.MockScrapperClient{}
		mockSender := &testutil.MockChatSender{}

		mockState.On("GetState", int64(12345), int64(12345)).Return(model.UserState{Type: model.StateWaitingURL})
		mockState.On("SetState", int64(12345), int64(12345), model.UserState{Type: model.StateWaitingTags, URL: "https://github.com/user/repo"}).Return()
		mockSender.On("Send", int64(12345), mock.MatchedBy(func(s string) bool {
			return strings.Contains(s, "Ссылка принята")
		})).Return(nil)

		handler := messages.NewHandler(mockState, mockScrapper, mockSender, logger)
		msg := testutil.NewTestMessage("https://github.com/user/repo")

		// When
		handler.HandleMessage(t.Context(), msg)

		// Then
		mockState.AssertExpectations(t)
		mockSender.AssertExpectations(t)
	})

	t.Run("Error: invalid url scheme", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockState := &testutil.MockStateManager{}
		mockScrapper := &testutil.MockScrapperClient{}
		mockSender := &testutil.MockChatSender{}

		mockState.On("GetState", int64(12345), int64(12345)).Return(model.UserState{Type: model.StateWaitingURL})
		mockSender.On("Send", int64(12345), mock.MatchedBy(func(s string) bool {
			return strings.Contains(s, "Некорректная ссылка")
		})).Return(nil)

		handler := messages.NewHandler(mockState, mockScrapper, mockSender, logger)
		msg := testutil.NewTestMessage("tbank://github.com/user/repo")

		// When
		handler.HandleMessage(t.Context(), msg)

		// Then
		mockState.AssertExpectations(t)
		mockSender.AssertExpectations(t)
	})
}

func TestHandler_HandleMessage_StateWaitingTags(t *testing.T) {
	t.Parallel()

	t.Run("Success: Adds link successfully", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockState := &testutil.MockStateManager{}
		mockScrapper := &testutil.MockScrapperClient{}
		mockSender := &testutil.MockChatSender{}

		state := model.UserState{Type: model.StateWaitingTags, URL: "https://github.com/user/repo"}
		mockState.On("GetState", int64(12345), int64(12345)).Return(state)
		mockState.On("ClearState", int64(12345), int64(12345)).Return()
		mockScrapper.On("AddLink", mock.Anything, int64(12345), "https://github.com/user/repo", []string{"tag1", "tag2"}).Return(&model.Link{}, nil)
		mockSender.On("Send", int64(12345), mock.MatchedBy(func(s string) bool {
			return strings.Contains(s, "успешно добавлена")
		})).Return(nil)

		handler := messages.NewHandler(mockState, mockScrapper, mockSender, logger)
		msg := testutil.NewTestMessage("tag1, tag2")

		// When
		handler.HandleMessage(t.Context(), msg)

		// Then
		mockState.AssertExpectations(t)
		mockScrapper.AssertExpectations(t)
		mockSender.AssertExpectations(t)
	})

	t.Run("Error: already subscribed", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockState := &testutil.MockStateManager{}
		mockScrapper := &testutil.MockScrapperClient{}
		mockSender := &testutil.MockChatSender{}

		state := model.UserState{Type: model.StateWaitingTags, URL: "https://github.com/user/repo"}
		mockState.On("GetState", int64(12345), int64(12345)).Return(state)
		mockState.On("ClearState", int64(12345), int64(12345)).Return()

		mockScrapper.On("AddLink", mock.Anything, int64(12345), "https://github.com/user/repo", []string{"tag1"}).Return(nil, model.ErrLinkAlreadyTracked)
		mockSender.On("Send", int64(12345), mock.MatchedBy(func(s string) bool {
			return strings.Contains(s, model.ErrLinkAlreadyTracked.Error())
		})).Return(nil)

		handler := messages.NewHandler(mockState, mockScrapper, mockSender, logger)
		msg := testutil.NewTestMessage("tag1")

		// When
		handler.HandleMessage(t.Context(), msg)

		// Then
		mockState.AssertExpectations(t)
		mockScrapper.AssertExpectations(t)
		mockSender.AssertExpectations(t)
	})

	t.Run("Success: adds link without tags (skip tags)", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockState := &testutil.MockStateManager{}
		mockScrapper := &testutil.MockScrapperClient{}
		mockSender := &testutil.MockChatSender{}

		state := model.UserState{Type: model.StateWaitingTags, URL: "https://github.com/user/repo"}
		mockState.On("GetState", int64(12345), int64(12345)).Return(state)
		mockState.On("ClearState", int64(12345), int64(12345)).Return()

		mockScrapper.On("AddLink", mock.Anything, int64(12345), "https://github.com/user/repo", mock.MatchedBy(func(tags []string) bool {
			return len(tags) == 0
		})).Return(&model.Link{}, nil)
		mockSender.On("Send", int64(12345), mock.MatchedBy(func(s string) bool {
			return strings.Contains(s, "успешно добавлена")
		})).Return(nil)

		handler := messages.NewHandler(mockState, mockScrapper, mockSender, logger)
		msg := testutil.NewTestMessage("-")

		// When
		handler.HandleMessage(t.Context(), msg)

		// Then
		mockState.AssertExpectations(t)
		mockScrapper.AssertExpectations(t)
		mockSender.AssertExpectations(t)
	})

	t.Run("Error: scrapper unavailable when adding link", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockState := &testutil.MockStateManager{}
		mockScrapper := &testutil.MockScrapperClient{}
		mockSender := &testutil.MockChatSender{}

		state := model.UserState{Type: model.StateWaitingTags, URL: "https://github.com/user/repo"}
		mockState.On("GetState", int64(12345), int64(12345)).Return(state)
		mockState.On("ClearState", int64(12345), int64(12345)).Return()

		mockScrapper.On("AddLink", mock.Anything, int64(12345), "https://github.com/user/repo", mock.Anything).Return(nil, model.ErrScrapperUnavailable)
		mockSender.On("Send", int64(12345), mock.MatchedBy(func(s string) bool {
			return strings.Contains(s, model.ErrScrapperUnavailable.Error())
		})).Return(nil)

		handler := messages.NewHandler(mockState, mockScrapper, mockSender, logger)
		msg := testutil.NewTestMessage("tag1")

		// When
		handler.HandleMessage(t.Context(), msg)

		// Then
		mockState.AssertExpectations(t)
		mockScrapper.AssertExpectations(t)
		mockSender.AssertExpectations(t)
	})
}

func TestHandler_HandleMessage_StateWaitingUntrackURL(t *testing.T) {
	t.Parallel()

	t.Run("Success: removes link successfully", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockState := &testutil.MockStateManager{}
		mockScrapper := &testutil.MockScrapperClient{}
		mockSender := &testutil.MockChatSender{}

		state := model.UserState{Type: model.StateWaitingUntrackURL}
		mockState.On("GetState", int64(12345), int64(12345)).Return(state)
		mockState.On("ClearState", int64(12345), int64(12345)).Return()

		mockScrapper.On("RemoveLink", mock.Anything, int64(12345), "https://github.com/user/repo").Return(&model.Link{}, nil)
		mockSender.On("Send", int64(12345), mock.MatchedBy(func(s string) bool {
			return strings.Contains(s, "удалена")
		})).Return(nil)

		handler := messages.NewHandler(mockState, mockScrapper, mockSender, logger)
		msg := testutil.NewTestMessage("https://github.com/user/repo")

		// When
		handler.HandleMessage(t.Context(), msg)

		// Then
		mockState.AssertExpectations(t)
		mockScrapper.AssertExpectations(t)
		mockSender.AssertExpectations(t)
	})

	t.Run("Error: invalid url for untrack", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockState := &testutil.MockStateManager{}
		mockScrapper := &testutil.MockScrapperClient{}
		mockSender := &testutil.MockChatSender{}

		state := model.UserState{Type: model.StateWaitingUntrackURL}
		mockState.On("GetState", int64(12345), int64(12345)).Return(state)
		mockSender.On("Send", int64(12345), mock.MatchedBy(func(s string) bool {
			return strings.Contains(s, "Некорректная ссылка")
		})).Return(nil)

		handler := messages.NewHandler(mockState, mockScrapper, mockSender, logger)
		msg := testutil.NewTestMessage("tbank://github.com/user/repo")

		// When
		handler.HandleMessage(t.Context(), msg)

		// Then
		mockState.AssertExpectations(t)
		mockSender.AssertExpectations(t)
	})

	t.Run("Error: link not found when untracking", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockState := &testutil.MockStateManager{}
		mockScrapper := &testutil.MockScrapperClient{}
		mockSender := &testutil.MockChatSender{}

		state := model.UserState{Type: model.StateWaitingUntrackURL}
		mockState.On("GetState", int64(12345), int64(12345)).Return(state)
		mockState.On("ClearState", int64(12345), int64(12345)).Return()

		mockScrapper.On("RemoveLink", mock.Anything, int64(12345), "https://github.com/user/repo").Return(nil, model.ErrLinkOrChatNotFound)
		mockSender.On("Send", int64(12345), mock.MatchedBy(func(s string) bool {
			return strings.Contains(s, "ссылка не найдена")
		})).Return(nil)

		handler := messages.NewHandler(mockState, mockScrapper, mockSender, logger)
		msg := testutil.NewTestMessage("https://github.com/user/repo")

		// When
		handler.HandleMessage(t.Context(), msg)

		// Then
		mockState.AssertExpectations(t)
		mockScrapper.AssertExpectations(t)
		mockSender.AssertExpectations(t)
	})
}

func TestHandler_HandleMessage_DefaultState(t *testing.T) {
	t.Parallel()

	t.Run("Success: ignores message in unknown state", func(t *testing.T) {
		t.Parallel()

		// Given
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		mockState := &testutil.MockStateManager{}
		mockScrapper := &testutil.MockScrapperClient{}
		mockSender := &testutil.MockChatSender{}

		mockState.On("GetState", int64(12345), int64(12345)).Return(model.UserState{Type: 0})

		handler := messages.NewHandler(mockState, mockScrapper, mockSender, logger)
		msg := testutil.NewTestMessage("some random message")

		// When
		handler.HandleMessage(t.Context(), msg)

		// Then
		mockState.AssertExpectations(t)
		mockScrapper.AssertNotCalled(t, "AddLink")
		mockScrapper.AssertNotCalled(t, "RemoveLink")
		mockSender.AssertNotCalled(t, "Send")
	})
}
