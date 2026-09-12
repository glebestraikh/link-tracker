package squirrel

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
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

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert chat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrChatAlreadyExists
	}

	return nil
}

func (r *ChatRepository) Delete(ctx context.Context, id int64) error {
	query, args, err := r.builder.
		Delete("chats").
		Where(sq.Eq{"tg_chat_id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete chat: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrChatNotFound
	}

	return nil
}

func (r *ChatRepository) Exists(ctx context.Context, id int64) (bool, error) {
	query, args, err := r.builder.
		Select("1").
		Prefix("SELECT EXISTS (").
		From("chats").
		Where(sq.Eq{"tg_chat_id": id}).
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
