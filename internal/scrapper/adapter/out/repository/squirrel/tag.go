package squirrel

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
)

type TagRepository struct {
	pool    *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewTagRepository(pool *pgxpool.Pool) *TagRepository {
	return &TagRepository{
		pool:    pool,
		builder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *TagRepository) Create(ctx context.Context, name string) (*model.Tag, error) {
	query, args, err := r.builder.
		Insert("tags").
		Columns("name").
		Values(name).
		Suffix("ON CONFLICT (name) DO NOTHING RETURNING id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert tag: %w", err)
	}

	var id int64
	if err = r.pool.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrTagAlreadyExists
		}
		return nil, fmt.Errorf("insert tag: %w", err)
	}

	return &model.Tag{ID: id, Name: name}, nil
}

func (r *TagRepository) GetByID(ctx context.Context, id int64) (*model.Tag, error) {
	return r.getOne(ctx, sq.Eq{"id": id})
}

func (r *TagRepository) GetByName(ctx context.Context, name string) (*model.Tag, error) {
	return r.getOne(ctx, sq.Eq{"name": name})
}

func (r *TagRepository) List(ctx context.Context, offset, limit uint64) ([]*model.Tag, error) {
	query, args, err := r.builder.
		Select("id", "name").
		From("tags").
		OrderBy("id").
		Limit(limit).
		Offset(offset).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list tags: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
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
	query, args, err := r.builder.
		Update("tags").
		Set("name", newName).
		Where(sq.Eq{"id": id}).
		Suffix("RETURNING id, name").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build rename tag: %w", err)
	}

	var tag model.Tag
	if err = r.pool.QueryRow(ctx, query, args...).Scan(&tag.ID, &tag.Name); err != nil {
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
	query, args, err := r.builder.
		Delete("tags").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete tag: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrTagNotFound
	}
	return nil
}

func (r *TagRepository) getOne(ctx context.Context, where sq.Eq) (*model.Tag, error) {
	query, args, err := r.builder.
		Select("id", "name").
		From("tags").
		Where(where).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get tag: %w", err)
	}

	var tag model.Tag
	if err = r.pool.QueryRow(ctx, query, args...).Scan(&tag.ID, &tag.Name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrTagNotFound
		}
		return nil, fmt.Errorf("get tag: %w", err)
	}
	return &tag, nil
}
