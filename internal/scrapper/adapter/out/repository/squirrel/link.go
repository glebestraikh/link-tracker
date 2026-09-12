package squirrel

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
)

type LinkRepository struct {
	pool    *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewLinkRepository(pool *pgxpool.Pool) *LinkRepository {
	return &LinkRepository{
		pool:    pool,
		builder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *LinkRepository) Save(ctx context.Context, link model.Link) (*model.Link, error) {
	if len(link.ChatIDs) == 0 {
		return nil, model.ErrLinkNotFound
	}
	chatID := link.ChatIDs[0]

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	linkID, lastUpdated, err := r.upsertLink(ctx, tx, link.URL)
	if err != nil {
		return nil, err
	}

	if err = r.insertSubscription(ctx, tx, linkID, chatID); err != nil {
		return nil, err
	}

	if err = r.attachTags(ctx, tx, linkID, chatID, link.Tags); err != nil {
		return nil, err
	}

	chatIDs, err := r.chatIDsForLinkTx(ctx, tx, linkID)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit save link: %w", err)
	}

	return &model.Link{
		ID:          linkID,
		URL:         link.URL,
		Tags:        link.Tags,
		ChatIDs:     chatIDs,
		LastUpdated: lastUpdated,
	}, nil
}

func (r *LinkRepository) Delete(ctx context.Context, chatID int64, url string) (*model.Link, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	findSQL, args, err := r.builder.
		Select("l.id", "COALESCE(l.last_updated, '0001-01-01'::timestamptz)").
		From("links l").
		Join("link_chat lc ON lc.link_id = l.id").
		Where(sq.Eq{"l.url": url, "lc.chat_id": chatID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find subscription: %w", err)
	}

	var linkID int64
	var lastUpdated time.Time
	if err = tx.QueryRow(ctx, findSQL, args...).Scan(&linkID, &lastUpdated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrLinkNotFound
		}
		return nil, fmt.Errorf("find subscription: %w", err)
	}

	tags, err := r.tagsForSubscriptionTx(ctx, tx, linkID, chatID)
	if err != nil {
		return nil, err
	}

	chatIDs, err := r.chatIDsForLinkTx(ctx, tx, linkID)
	if err != nil {
		return nil, err
	}

	delSubSQL, delSubArgs, err := r.builder.
		Delete("link_chat").
		Where(sq.Eq{"link_id": linkID, "chat_id": chatID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build delete subscription: %w", err)
	}
	if _, err = tx.Exec(ctx, delSubSQL, delSubArgs...); err != nil {
		return nil, fmt.Errorf("delete subscription: %w", err)
	}

	if _, err = tx.Exec(ctx, `
		DELETE FROM links WHERE id = $1
		AND NOT EXISTS (SELECT 1 FROM link_chat WHERE link_id = $1)
	`, linkID); err != nil {
		return nil, fmt.Errorf("delete orphan link: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit delete link: %w", err)
	}

	return &model.Link{
		ID:          linkID,
		URL:         url,
		Tags:        tags,
		ChatIDs:     chatIDs,
		LastUpdated: lastUpdated,
	}, nil
}

func (r *LinkRepository) FindByChatID(ctx context.Context, chatID int64, offset, limit uint64) ([]*model.Link, error) {
	query, args, err := r.builder.
		Select("l.id", "l.url", "COALESCE(l.last_updated, '0001-01-01'::timestamptz)").
		From("links l").
		Join("link_chat lc ON lc.link_id = l.id").
		Where(sq.Eq{"lc.chat_id": chatID}).
		OrderBy("l.id").
		Limit(limit).
		Offset(offset).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find by chat: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query links by chat: %w", err)
	}

	links := make([]*model.Link, 0)
	for rows.Next() {
		var l model.Link
		if err = rows.Scan(&l.ID, &l.URL, &l.LastUpdated); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan link: %w", err)
		}
		links = append(links, &l)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read links: %w", err)
	}

	for _, l := range links {
		tags, tagsErr := r.tagsForSubscription(ctx, l.ID, chatID)
		if tagsErr != nil {
			return nil, tagsErr
		}
		l.Tags = tags
		l.ChatIDs = []int64{chatID}
	}

	return links, nil
}

func (r *LinkRepository) FindAllPaged(ctx context.Context, offset, limit uint64) ([]*model.Link, error) {
	query, args, err := r.builder.
		Select("id", "url", "COALESCE(last_updated, '0001-01-01'::timestamptz)").
		From("links").
		OrderBy("id").
		Limit(limit).
		Offset(offset).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find all: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query all links: %w", err)
	}

	links := make([]*model.Link, 0)
	for rows.Next() {
		var l model.Link
		if err = rows.Scan(&l.ID, &l.URL, &l.LastUpdated); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan link: %w", err)
		}
		links = append(links, &l)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read links: %w", err)
	}

	for _, l := range links {
		ids, idsErr := r.chatIDsForLink(ctx, l.ID)
		if idsErr != nil {
			return nil, idsErr
		}
		l.ChatIDs = ids
	}

	return links, nil
}

func (r *LinkRepository) UpdateLastUpdated(ctx context.Context, id int64, t time.Time) error {
	query, args, err := r.builder.
		Update("links").
		Set("last_updated", t).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update last_updated: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update last_updated: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrLinkNotFound
	}
	return nil
}

func (r *LinkRepository) upsertLink(ctx context.Context, tx pgx.Tx, url string) (int64, time.Time, error) {
	query, args, err := r.builder.
		Insert("links").
		Columns("url").
		Values(url).
		Suffix("ON CONFLICT (url) DO UPDATE SET url = EXCLUDED.url RETURNING id, COALESCE(last_updated, '0001-01-01'::timestamptz)").
		ToSql()
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("build upsert link: %w", err)
	}

	var id int64
	var lastUpdated time.Time
	if err = tx.QueryRow(ctx, query, args...).Scan(&id, &lastUpdated); err != nil {
		return 0, time.Time{}, fmt.Errorf("upsert link: %w", err)
	}
	return id, lastUpdated, nil
}

func (r *LinkRepository) insertSubscription(ctx context.Context, tx pgx.Tx, linkID, chatID int64) error {
	query, args, err := r.builder.
		Insert("link_chat").
		Columns("link_id", "chat_id").
		Values(linkID, chatID).
		Suffix("ON CONFLICT DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert subscription: %w", err)
	}

	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert subscription: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrLinkAlreadyTracked
	}

	return nil
}

func (r *LinkRepository) attachTags(ctx context.Context, tx pgx.Tx, linkID, chatID int64, tags []string) error {
	for _, name := range tags {
		upsertTagSQL, args, err := r.builder.
			Insert("tags").
			Columns("name").
			Values(name).
			Suffix("ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id").
			ToSql()
		if err != nil {
			return fmt.Errorf("build upsert tag: %w", err)
		}

		var tagID int64
		if err = tx.QueryRow(ctx, upsertTagSQL, args...).Scan(&tagID); err != nil {
			return fmt.Errorf("upsert tag %q: %w", name, err)
		}

		linkTagSQL, lArgs, err := r.builder.
			Insert("link_chat_tag").
			Columns("link_id", "chat_id", "tag_id").
			Values(linkID, chatID, tagID).
			Suffix("ON CONFLICT DO NOTHING").
			ToSql()
		if err != nil {
			return fmt.Errorf("build attach tag: %w", err)
		}
		if _, err = tx.Exec(ctx, linkTagSQL, lArgs...); err != nil {
			return fmt.Errorf("attach tag %q: %w", name, err)
		}
	}
	return nil
}

func (r *LinkRepository) chatIDsForLinkTx(ctx context.Context, tx pgx.Tx, linkID int64) ([]int64, error) {
	query, args, err := r.builder.
		Select("chat_id").
		From("link_chat").
		Where(sq.Eq{"link_id": linkID}).
		OrderBy("chat_id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build subscribers: %w", err)
	}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read subscribers: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan subscriber: %w", err)
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read subscribers: %w", err)
	}

	return ids, nil
}

func (r *LinkRepository) tagsForSubscriptionTx(ctx context.Context, tx pgx.Tx, linkID, chatID int64) ([]string, error) {
	query, args, err := r.builder.
		Select("t.name").
		From("link_chat_tag lct").
		Join("tags t ON t.id = lct.tag_id").
		Where(sq.Eq{"lct.link_id": linkID, "lct.chat_id": chatID}).
		OrderBy("t.name").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build read tags: %w", err)
	}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, name)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read tags: %w", err)
	}

	return tags, nil
}

func (r *LinkRepository) tagsForSubscription(ctx context.Context, linkID, chatID int64) ([]string, error) {
	query, args, err := r.builder.
		Select("t.name").
		From("link_chat_tag lct").
		Join("tags t ON t.id = lct.tag_id").
		Where(sq.Eq{"lct.link_id": linkID, "lct.chat_id": chatID}).
		OrderBy("t.name").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build read tags: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, name)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read tags: %w", err)
	}
	return tags, nil
}

func (r *LinkRepository) chatIDsForLink(ctx context.Context, linkID int64) ([]int64, error) {
	query, args, err := r.builder.
		Select("chat_id").
		From("link_chat").
		Where(sq.Eq{"link_id": linkID}).
		OrderBy("chat_id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build subscribers: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read subscribers: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan subscriber: %w", err)
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read subscribers: %w", err)
	}
	return ids, nil
}
