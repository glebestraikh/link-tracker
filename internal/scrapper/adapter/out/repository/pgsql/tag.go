package pgsql

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
)

type TagRepository struct {
	pool *pgxpool.Pool
}

func NewTagRepository(pool *pgxpool.Pool) *TagRepository {
	return &TagRepository{
		pool: pool,
	}
}

func (r *TagRepository) Create(ctx context.Context, name string) (*model.Tag, error) {
	const query = `
		INSERT INTO tags (name)
		VALUES ($1)
		ON CONFLICT (name) DO NOTHING
		RETURNING id
	`

	var id int64
	if err := r.pool.QueryRow(ctx, query, name).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrTagAlreadyExists
		}
		return nil, fmt.Errorf("insert tag: %w", err)
	}

	return &model.Tag{ID: id, Name: name}, nil
}

func (r *TagRepository) GetByID(ctx context.Context, id int64) (*model.Tag, error) {
	const query = `
		SELECT id, name
		FROM tags
		WHERE id = $1
	`

	var tag model.Tag
	if err := r.pool.QueryRow(ctx, query, id).Scan(&tag.ID, &tag.Name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrTagNotFound
		}
		return nil, fmt.Errorf("get tag by id: %w", err)
	}
	return &tag, nil
}

func (r *TagRepository) GetByName(ctx context.Context, name string) (*model.Tag, error) {
	const query = `
		SELECT id, name
		FROM tags
		WHERE name = $1
	`

	var tag model.Tag
	if err := r.pool.QueryRow(ctx, query, name).Scan(&tag.ID, &tag.Name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrTagNotFound
		}
		return nil, fmt.Errorf("get tag by name: %w", err)
	}
	return &tag, nil
}

func (r *TagRepository) List(ctx context.Context, offset, limit uint64) ([]*model.Tag, error) {
	const query = `
		SELECT id, name
		FROM tags
		ORDER BY id
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	tags := make([]*model.Tag, 0)
	for rows.Next() {
		var tag model.Tag
		if err = rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, &tag)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read tags: %w", err)
	}
	return tags, nil
}

func (r *TagRepository) Rename(ctx context.Context, id int64, newName string) (*model.Tag, error) {
	const query = `
		UPDATE tags
		SET name = $2
		WHERE id = $1
		RETURNING id, name
	`

	var tag model.Tag
	if err := r.pool.QueryRow(ctx, query, id, newName).Scan(&tag.ID, &tag.Name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrTagNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, model.ErrTagAlreadyExists
		}
		return nil, fmt.Errorf("rename tag: %w", err)
	}
	return &tag, nil
}

func (r *TagRepository) Delete(ctx context.Context, id int64) error {
	const query = `
		DELETE FROM tags
		WHERE id = $1
	`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrTagNotFound
	}
	return nil
}
