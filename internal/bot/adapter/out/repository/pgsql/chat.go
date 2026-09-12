package pgsql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository struct {
	pool *pgxpool.Pool
}

func NewChatRepository(pool *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{
		pool: pool,
	}
}

func (r *ChatRepository) Save(ctx context.Context, chatID int64) error {
	const query = `
		INSERT INTO chats (tg_chat_id)
		VALUES ($1)
		ON CONFLICT (tg_chat_id) DO NOTHING
	`

	if _, err := r.pool.Exec(ctx, query, chatID); err != nil {
		return fmt.Errorf("insert chat: %w", err)
	}

	return nil
}

func (r *ChatRepository) Exists(ctx context.Context, chatID int64) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM chats WHERE tg_chat_id = $1)`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, chatID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check chat exists: %w", err)
	}

	return exists, nil
}
