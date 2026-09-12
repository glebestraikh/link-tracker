package pgsql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
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

	tag, err := r.pool.Exec(ctx, query, chatID)
	if err != nil {
		return fmt.Errorf("insert chat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrChatAlreadyExists
	}

	return nil
}

func (r *ChatRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM chats WHERE tg_chat_id = $1`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrChatNotFound
	}

	return nil
}

func (r *ChatRepository) Exists(ctx context.Context, id int64) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM chats WHERE tg_chat_id = $1)`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("check chat exists: %w", err)
	}

	return exists, nil
}
