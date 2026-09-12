package inmemory

import (
	"context"
	"sync"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
)

type ChatRepository struct {
	mu    sync.RWMutex
	chats map[ChatID]struct{}
}

func NewChatRepository() *ChatRepository {
	return &ChatRepository{
		chats: make(map[ChatID]struct{}),
	}
}

func (r *ChatRepository) Save(_ context.Context, chatID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.chats[chatID]; ok {
		return model.ErrChatAlreadyExists
	}

	r.chats[chatID] = struct{}{}
	return nil
}

func (r *ChatRepository) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.chats[id]; !ok {
		return model.ErrChatNotFound
	}

	delete(r.chats, id)
	return nil
}

func (r *ChatRepository) Exists(_ context.Context, id int64) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.chats[id]
	return ok, nil
}
