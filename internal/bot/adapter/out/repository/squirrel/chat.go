package squirrel

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository struct {
	pool    *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewChatRepository(pool *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{
		pool:    pool,
		builder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *ChatRepository) Save(ctx context.Context, chatID int64) error {
	query, args, err := r.builder.
		Insert("chats").
		Columns("tg_chat_id").
		Values(chatID).
		Suffix("ON CONFLICT (tg_chat_id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert chat: %w", err)
	}

	if _, err = r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert chat: %w", err)
	}

	return nil
}

func (r *ChatRepository) Exists(ctx context.Context, chatID int64) (bool, error) {
	query, args, err := r.builder.
		Select("1").
		Prefix("SELECT EXISTS (").
		From("chats").
		Where(sq.Eq{"tg_chat_id": chatID}).
		Suffix(")").
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build chat exists: %w", err)
	}

	var exists bool
	if err = r.pool.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
		return false, fmt.Errorf("check chat exists: %w", err)
	}

	return exists, nil
}
