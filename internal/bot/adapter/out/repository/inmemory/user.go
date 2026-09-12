package inmemory

import (
	"context"
	"sync"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
)

type UserRepository struct {
	mu    sync.RWMutex
	users map[int64]model.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		users: make(map[int64]model.User),
	}
}

func (r *UserRepository) Save(_ context.Context, user model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.ID] = user
	return nil
}

func (r *UserRepository) Exists(_ context.Context, id int64) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.users[id]
	return ok, nil
}
