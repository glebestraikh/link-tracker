package squirrel

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
)

type UserRepository struct {
	pool    *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool:    pool,
		builder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *UserRepository) Save(ctx context.Context, user model.User) error {
	query, args, err := r.builder.
		Insert("users").
		Columns("id", "username").
		Values(user.ID, user.Username).
		Suffix("ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert user: %w", err)
	}

	if _, err = r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

func (r *UserRepository) Exists(ctx context.Context, id int64) (bool, error) {
	query, args, err := r.builder.
		Select("1").
		Prefix("SELECT EXISTS (").
		From("users").
		Where(sq.Eq{"id": id}).
		Suffix(")").
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build user exists: %w", err)
	}

	var exists bool
	if err = r.pool.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}

	return exists, nil
}
