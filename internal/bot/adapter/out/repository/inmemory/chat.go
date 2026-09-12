package inmemory

import (
	"context"
	"sync"
)

type ChatID = int64

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

	r.chats[chatID] = struct{}{}
	return nil
}

func (r *ChatRepository) Exists(_ context.Context, chatID int64) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.chats[chatID]
	return ok, nil
}
