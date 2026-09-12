package testutil

import (
	"context"
	"fmt"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"

	"github.com/stretchr/testify/mock"
)

type MockScrapperClient struct {
	mock.Mock
}

func (m *MockScrapperClient) RegisterChat(ctx context.Context, chatID int64) error {
	args := m.Called(ctx, chatID)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("register chat: %w", err)
	}
	return nil
}

func (m *MockScrapperClient) DeleteChat(ctx context.Context, chatID int64) error {
	args := m.Called(ctx, chatID)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}
	return nil
}

func (m *MockScrapperClient) AddLink(ctx context.Context, chatID int64, url string, tags []string) (*model.Link, error) {
	args := m.Called(ctx, chatID, url, tags)

	var link *model.Link
	if v := args.Get(0); v != nil {
		if l, ok := v.(*model.Link); ok {
			link = l
		}
	}

	if err := args.Error(1); err != nil {
		return link, fmt.Errorf("add link: %w", err)
	}

	return link, nil
}

func (m *MockScrapperClient) RemoveLink(ctx context.Context, chatID int64, url string) (*model.Link, error) {
	args := m.Called(ctx, chatID, url)

	var link *model.Link
	if v := args.Get(0); v != nil {
		if l, ok := v.(*model.Link); ok {
			link = l
		}
	}

	if err := args.Error(1); err != nil {
		return link, fmt.Errorf("remove link: %w", err)
	}

	return link, nil
}

func (m *MockScrapperClient) GetLinks(ctx context.Context, chatID int64) ([]*model.Link, error) {
	args := m.Called(ctx, chatID)

	var links []*model.Link
	if v := args.Get(0); v != nil {
		if l, ok := v.([]*model.Link); ok {
			links = l
		}
	}

	if err := args.Error(1); err != nil {
		return links, fmt.Errorf("get links: %w", err)
	}

	return links, nil
}

type MockStateManager struct {
	mock.Mock
}

func (m *MockStateManager) GetState(chatID, userID int64) model.UserState {
	args := m.Called(chatID, userID)

	if v, ok := args.Get(0).(model.UserState); ok {
		return v
	}
	return model.UserState{}
}

func (m *MockStateManager) SetState(chatID, userID int64, state model.UserState) {
	m.Called(chatID, userID, state)
}

func (m *MockStateManager) ClearState(chatID, userID int64) {
	m.Called(chatID, userID)
}

type MockChatSender struct {
	mock.Mock
}

func (m *MockChatSender) Send(chatID int64, text string) error {
	args := m.Called(chatID, text)
	if err := args.Error(0); err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}
