package state

import (
	"sync"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
)

type Manager struct {
	mu     sync.RWMutex
	states map[key]model.UserState
}

type key struct {
	chatID int64
	userID int64
}

func NewManager() *Manager {
	return &Manager{
		states: make(map[key]model.UserState),
	}
}

func (m *Manager) GetState(chatID int64, userID int64) model.UserState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.states[key{chatID: chatID, userID: userID}]
}

func (m *Manager) SetState(chatID int64, userID int64, state model.UserState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.states[key{chatID: chatID, userID: userID}] = state
}

func (m *Manager) ClearState(chatID int64, userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.states, key{chatID: chatID, userID: userID})
}
