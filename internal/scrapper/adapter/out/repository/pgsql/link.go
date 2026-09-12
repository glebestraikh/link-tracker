package pgsql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
)

type LinkRepository struct {
	pool *pgxpool.Pool
}

func NewLinkRepository(pool *pgxpool.Pool) *LinkRepository {
	return &LinkRepository{
		pool: pool,
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

	linkID, lastUpdated, err := upsertLink(ctx, tx, link.URL)
	if err != nil {
		return nil, err
	}

	if err = insertSubscription(ctx, tx, linkID, chatID); err != nil {
		return nil, err
	}

	if err = attachTags(ctx, tx, linkID, chatID, link.Tags); err != nil {
		return nil, err
	}

	chatIDs, err := chatIDsForLink(ctx, tx, linkID)
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

func upsertLink(ctx context.Context, tx pgx.Tx, url string) (int64, time.Time, error) {
	const query = `
		INSERT INTO links (url) VALUES ($1)
		ON CONFLICT (url) DO UPDATE SET url = EXCLUDED.url
		RETURNING id, COALESCE(last_updated, '0001-01-01'::timestamptz)
	`

	var id int64
	var lastUpdated time.Time
	if err := tx.QueryRow(ctx, query, url).Scan(&id, &lastUpdated); err != nil {
		return 0, time.Time{}, fmt.Errorf("upsert link: %w", err)
	}
	return id, lastUpdated, nil
}

func insertSubscription(ctx context.Context, tx pgx.Tx, linkID, chatID int64) error {
	const query = `
		INSERT INTO link_chat (link_id, chat_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`

	tag, err := tx.Exec(ctx, query, linkID, chatID)
	if err != nil {
		return fmt.Errorf("insert subscription: %w", err)

	}
	if tag.RowsAffected() == 0 {
		return model.ErrLinkAlreadyTracked
	}

	return nil
}

func attachTags(ctx context.Context, tx pgx.Tx, linkID, chatID int64, tags []string) error {
	for _, name := range tags {
		var tagID int64
		const upsertTag = `
			INSERT INTO tags (name) VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`
		if err := tx.QueryRow(ctx, upsertTag, name).Scan(&tagID); err != nil {
			return fmt.Errorf("upsert tag %q: %w", name, err)
		}

		const linkTag = `
			INSERT INTO link_chat_tag (link_id, chat_id, tag_id) VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING
		`
		if _, err := tx.Exec(ctx, linkTag, linkID, chatID, tagID); err != nil {
			return fmt.Errorf("attach tag %q: %w", name, err)
		}
	}
	return nil
}

func chatIDsForLink(ctx context.Context, tx pgx.Tx, linkID int64) ([]int64, error) {
	const query = `
		SELECT chat_id
		FROM link_chat
		WHERE link_id = $1
		ORDER BY chat_id
	`

	rows, err := tx.Query(ctx, query, linkID)
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

func (r *LinkRepository) Delete(ctx context.Context, chatID int64, url string) (*model.Link, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var linkID int64
	var lastUpdated time.Time
	const findLink = `
		SELECT l.id, COALESCE(l.last_updated, '0001-01-01'::timestamptz)
		FROM links l
		JOIN link_chat lc ON lc.link_id = l.id
		WHERE l.url = $1 AND lc.chat_id = $2
	`
	if err = tx.QueryRow(ctx, findLink, url, chatID).Scan(&linkID, &lastUpdated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrLinkNotFound
		}
		return nil, fmt.Errorf("find subscription: %w", err)
	}

	tags, err := tagsForSubscription(ctx, tx, linkID, chatID)
	if err != nil {
		return nil, err
	}

	chatIDs, err := chatIDsForLink(ctx, tx, linkID)
	if err != nil {
		return nil, err
	}

	const deleteSubscriptionQuery = `
		DELETE FROM link_chat
		WHERE link_id = $1 AND chat_id = $2
	`
	if _, err = tx.Exec(ctx, deleteSubscriptionQuery, linkID, chatID); err != nil {
		return nil, fmt.Errorf("delete subscription: %w", err)
	}

	const deleteOrphanLinkQuery = `
		DELETE FROM links
		WHERE id = $1
		AND NOT EXISTS (
			SELECT 1 FROM link_chat WHERE link_id = $1
		)
	`
	if _, err = tx.Exec(ctx, deleteOrphanLinkQuery, linkID); err != nil {
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

func tagsForSubscription(ctx context.Context, tx pgx.Tx, linkID, chatID int64) ([]string, error) {
	const queryTagsForSubscription = `
		SELECT t.name
		FROM link_chat_tag lct
		JOIN tags t ON t.id = lct.tag_id
		WHERE lct.link_id = $1 AND lct.chat_id = $2
		ORDER BY t.name
	`

	rows, err := tx.Query(ctx, queryTagsForSubscription, linkID, chatID)
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

func (r *LinkRepository) FindByChatID(ctx context.Context, chatID int64, offset, limit uint64) ([]*model.Link, error) {
	const linksQuery = `
		SELECT l.id, l.url, COALESCE(l.last_updated, '0001-01-01'::timestamptz)
		FROM links l
		JOIN link_chat lc ON lc.link_id = l.id
		WHERE lc.chat_id = $1
		ORDER BY l.id
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, linksQuery, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query links by chat: %w", err)
	}
	defer rows.Close()

	links := make([]*model.Link, 0)
	for rows.Next() {
		var l model.Link
		if err = rows.Scan(&l.ID, &l.URL, &l.LastUpdated); err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}
		links = append(links, &l)
	}
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
	const linksQuery = `
		SELECT id, url, COALESCE(last_updated, '0001-01-01'::timestamptz)
		FROM links
		ORDER BY id
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, linksQuery, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query all links: %w", err)
	}
	defer rows.Close()

	links := make([]*model.Link, 0)
	for rows.Next() {
		var l model.Link
		if err = rows.Scan(&l.ID, &l.URL, &l.LastUpdated); err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}
		links = append(links, &l)
	}
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
	const query = `
		UPDATE links
		SET last_updated = $2
		WHERE id = $1
	`

	tag, err := r.pool.Exec(ctx, query, id, t)
	if err != nil {
		return fmt.Errorf("update last_updated: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrLinkNotFound
	}

	return nil
}

func (r *LinkRepository) tagsForSubscription(ctx context.Context, linkID, chatID int64) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.name FROM link_chat_tag lct
		JOIN tags t ON t.id = lct.tag_id
		WHERE lct.link_id = $1 AND lct.chat_id = $2
		ORDER BY t.name
	`, linkID, chatID)
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
	const chatIDsForLinkQuery = `
		SELECT chat_id
		FROM link_chat
		WHERE link_id = $1
		ORDER BY chat_id
	`

	rows, err := r.pool.Query(ctx, chatIDsForLinkQuery, linkID)
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
