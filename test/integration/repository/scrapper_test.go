//go:build integration

package repository_test

import (
	"context"
	"errors"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/adapter/out/repository/pgsql"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/adapter/out/repository/squirrel"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
	scrapperservice "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/service"
)

type scrapperImpl struct {
	name     string
	chatRepo scrapperservice.ChatRepository
	linkRepo scrapperservice.LinkRepository
	tagRepo  scrapperservice.TagRepository
}

func scrapperImpls(pool *pgxpool.Pool) []scrapperImpl {
	return []scrapperImpl{
		{
			name:     "sql",
			chatRepo: pgsql.NewChatRepository(pool),
			linkRepo: pgsql.NewLinkRepository(pool),
			tagRepo:  pgsql.NewTagRepository(pool),
		},
		{
			name:     "squirrel",
			chatRepo: squirrel.NewChatRepository(pool),
			linkRepo: squirrel.NewLinkRepository(pool),
			tagRepo:  squirrel.NewTagRepository(pool),
		},
	}
}

func resetScrapper(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	truncate(t, pool, "link_chat_tag", "tags", "link_chat", "links", "chats")
}

func TestScrapperChatRepository(t *testing.T) {
	pool := scrapperPool(t)

	for _, impl := range scrapperImpls(pool) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			resetScrapper(t, pool)

			t.Run("Save then Exists returns true", func(t *testing.T) {
				require.NoError(t, impl.chatRepo.Save(ctx, 100))
				exists, err := impl.chatRepo.Exists(ctx, 100)
				require.NoError(t, err)
				require.True(t, exists)
			})

			t.Run("Save duplicate returns ErrChatAlreadyExists", func(t *testing.T) {
				require.NoError(t, impl.chatRepo.Save(ctx, 200))
				err := impl.chatRepo.Save(ctx, 200)
				require.ErrorIs(t, err, model.ErrChatAlreadyExists)
			})

			t.Run("Delete then Exists returns false", func(t *testing.T) {
				require.NoError(t, impl.chatRepo.Save(ctx, 300))
				require.NoError(t, impl.chatRepo.Delete(ctx, 300))
				exists, err := impl.chatRepo.Exists(ctx, 300)
				require.NoError(t, err)
				require.False(t, exists)
			})

			t.Run("Delete unknown returns ErrChatNotFound", func(t *testing.T) {
				err := impl.chatRepo.Delete(ctx, 99999)
				require.ErrorIs(t, err, model.ErrChatNotFound)
			})
		})
	}
}

func TestScrapperLinkRepository(t *testing.T) {
	pool := scrapperPool(t)

	for _, impl := range scrapperImpls(pool) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()

			t.Run("Save subscribes chat with tags and returns chat-scoped view", func(t *testing.T) {
				resetScrapper(t, pool)
				require.NoError(t, impl.chatRepo.Save(ctx, 1))

				link, err := impl.linkRepo.Save(ctx, model.Link{
					URL:     "https://github.com/x/y",
					Tags:    []string{"go", "backend"},
					ChatIDs: []int64{1},
				})
				require.NoError(t, err)
				require.NotZero(t, link.ID)
				require.Equal(t, "https://github.com/x/y", link.URL)
				require.ElementsMatch(t, []string{"go", "backend"}, link.Tags)
				require.Equal(t, []int64{1}, link.ChatIDs)
			})

			t.Run("Save duplicate (same chat and url) returns ErrLinkAlreadyTracked", func(t *testing.T) {
				resetScrapper(t, pool)
				require.NoError(t, impl.chatRepo.Save(ctx, 1))

				_, err := impl.linkRepo.Save(ctx, model.Link{
					URL:     "https://github.com/x/y",
					ChatIDs: []int64{1},
				})
				require.NoError(t, err)

				_, err = impl.linkRepo.Save(ctx, model.Link{
					URL:     "https://github.com/x/y",
					ChatIDs: []int64{1},
				})
				require.ErrorIs(t, err, model.ErrLinkAlreadyTracked)
			})

			t.Run("Save same URL different chat: link is shared, both subscribed", func(t *testing.T) {
				resetScrapper(t, pool)
				require.NoError(t, impl.chatRepo.Save(ctx, 1))
				require.NoError(t, impl.chatRepo.Save(ctx, 2))

				first, err := impl.linkRepo.Save(ctx, model.Link{
					URL: "https://github.com/x/y", ChatIDs: []int64{1},
				})
				require.NoError(t, err)

				second, err := impl.linkRepo.Save(ctx, model.Link{
					URL: "https://github.com/x/y", ChatIDs: []int64{2},
				})
				require.NoError(t, err)

				require.Equal(t, first.ID, second.ID)
				require.ElementsMatch(t, []int64{1, 2}, second.ChatIDs)
			})

			t.Run("FindByChatID returns chat-scoped tags with pagination", func(t *testing.T) {
				resetScrapper(t, pool)
				require.NoError(t, impl.chatRepo.Save(ctx, 1))

				for i, url := range []string{"https://a.dev/1", "https://a.dev/2", "https://a.dev/3"} {
					_, err := impl.linkRepo.Save(ctx, model.Link{
						URL:     url,
						Tags:    []string{string(rune('a' + i))},
						ChatIDs: []int64{1},
					})
					require.NoError(t, err)
				}

				page1, err := impl.linkRepo.FindByChatID(ctx, 1, 0, 2)
				require.NoError(t, err)
				require.Len(t, page1, 2)

				page2, err := impl.linkRepo.FindByChatID(ctx, 1, 2, 2)
				require.NoError(t, err)
				require.Len(t, page2, 1)
			})

			t.Run("FindAllPaged paginates across all links", func(t *testing.T) {
				resetScrapper(t, pool)
				require.NoError(t, impl.chatRepo.Save(ctx, 1))
				for _, url := range []string{"https://a.dev/p1", "https://a.dev/p2", "https://a.dev/p3"} {
					_, err := impl.linkRepo.Save(ctx, model.Link{URL: url, ChatIDs: []int64{1}})
					require.NoError(t, err)
				}

				page1, err := impl.linkRepo.FindAllPaged(ctx, 0, 2)
				require.NoError(t, err)
				require.Len(t, page1, 2)

				page2, err := impl.linkRepo.FindAllPaged(ctx, 2, 2)
				require.NoError(t, err)
				require.Len(t, page2, 1)
			})

			t.Run("Delete removes subscription and orphan link", func(t *testing.T) {
				resetScrapper(t, pool)
				require.NoError(t, impl.chatRepo.Save(ctx, 1))

				saved, err := impl.linkRepo.Save(ctx, model.Link{
					URL: "https://orphan.dev", ChatIDs: []int64{1}, Tags: []string{"x"},
				})
				require.NoError(t, err)

				deleted, err := impl.linkRepo.Delete(ctx, 1, "https://orphan.dev")
				require.NoError(t, err)
				require.Equal(t, saved.ID, deleted.ID)

				orphanGone, err := linkExists(ctx, pool, "https://orphan.dev")
				require.NoError(t, err)
				require.False(t, orphanGone)
			})

			t.Run("Delete keeps link when other subscribers remain", func(t *testing.T) {
				resetScrapper(t, pool)
				require.NoError(t, impl.chatRepo.Save(ctx, 1))
				require.NoError(t, impl.chatRepo.Save(ctx, 2))

				_, err := impl.linkRepo.Save(ctx, model.Link{URL: "https://shared.dev", ChatIDs: []int64{1}})
				require.NoError(t, err)
				_, err = impl.linkRepo.Save(ctx, model.Link{URL: "https://shared.dev", ChatIDs: []int64{2}})
				require.NoError(t, err)

				_, err = impl.linkRepo.Delete(ctx, 1, "https://shared.dev")
				require.NoError(t, err)

				stillExists, err := linkExists(ctx, pool, "https://shared.dev")
				require.NoError(t, err)
				require.True(t, stillExists)
			})

			t.Run("Delete unknown returns ErrLinkNotFound", func(t *testing.T) {
				resetScrapper(t, pool)
				require.NoError(t, impl.chatRepo.Save(ctx, 1))

				_, err := impl.linkRepo.Delete(ctx, 1, "https://does-not-exist.dev")
				require.ErrorIs(t, err, model.ErrLinkNotFound)
			})

			t.Run("UpdateLastUpdated persists timestamp", func(t *testing.T) {
				resetScrapper(t, pool)
				require.NoError(t, impl.chatRepo.Save(ctx, 1))
				saved, err := impl.linkRepo.Save(ctx, model.Link{URL: "https://ts.dev", ChatIDs: []int64{1}})
				require.NoError(t, err)

				stamp := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
				require.NoError(t, impl.linkRepo.UpdateLastUpdated(ctx, saved.ID, stamp))

				links, err := impl.linkRepo.FindAllPaged(ctx, 0, 10)
				require.NoError(t, err)
				require.Len(t, links, 1)
				require.True(t, stamp.Equal(links[0].LastUpdated))
			})
		})
	}
}

func TestScrapperTagRepository(t *testing.T) {
	pool := scrapperPool(t)

	for _, impl := range scrapperImpls(pool) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()

			t.Run("Create then GetByID returns the same tag", func(t *testing.T) {
				resetScrapper(t, pool)
				created, err := impl.tagRepo.Create(ctx, "first")
				require.NoError(t, err)
				require.NotZero(t, created.ID)

				got, err := impl.tagRepo.GetByID(ctx, created.ID)
				require.NoError(t, err)
				require.Equal(t, created.Name, got.Name)
			})

			t.Run("Create duplicate returns ErrTagAlreadyExists", func(t *testing.T) {
				resetScrapper(t, pool)
				_, err := impl.tagRepo.Create(ctx, "dup")
				require.NoError(t, err)
				_, err = impl.tagRepo.Create(ctx, "dup")
				require.ErrorIs(t, err, model.ErrTagAlreadyExists)
			})

			t.Run("List paginates", func(t *testing.T) {
				resetScrapper(t, pool)
				for _, name := range []string{"a", "b", "c"} {
					_, err := impl.tagRepo.Create(ctx, name)
					require.NoError(t, err)
				}
				page1, err := impl.tagRepo.List(ctx, 0, 2)
				require.NoError(t, err)
				require.Len(t, page1, 2)
				page2, err := impl.tagRepo.List(ctx, 2, 2)
				require.NoError(t, err)
				require.Len(t, page2, 1)
			})

			t.Run("Rename returns updated tag", func(t *testing.T) {
				resetScrapper(t, pool)
				created, err := impl.tagRepo.Create(ctx, "old")
				require.NoError(t, err)
				renamed, err := impl.tagRepo.Rename(ctx, created.ID, "new")
				require.NoError(t, err)
				require.Equal(t, "new", renamed.Name)
			})

			t.Run("Rename to existing name returns ErrTagAlreadyExists", func(t *testing.T) {
				resetScrapper(t, pool)
				_, err := impl.tagRepo.Create(ctx, "taken")
				require.NoError(t, err)
				other, err := impl.tagRepo.Create(ctx, "free")
				require.NoError(t, err)
				_, err = impl.tagRepo.Rename(ctx, other.ID, "taken")
				require.ErrorIs(t, err, model.ErrTagAlreadyExists)
			})

			t.Run("Delete removes tag", func(t *testing.T) {
				resetScrapper(t, pool)
				created, err := impl.tagRepo.Create(ctx, "doomed")
				require.NoError(t, err)
				require.NoError(t, impl.tagRepo.Delete(ctx, created.ID))

				_, err = impl.tagRepo.GetByID(ctx, created.ID)
				require.ErrorIs(t, err, model.ErrTagNotFound)
			})

			t.Run("Delete unknown returns ErrTagNotFound", func(t *testing.T) {
				resetScrapper(t, pool)
				err := impl.tagRepo.Delete(ctx, 99999)
				require.ErrorIs(t, err, model.ErrTagNotFound)
			})
		})
	}
}

func linkExists(ctx context.Context, pool *pgxpool.Pool, url string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM links WHERE url = $1)`, url).Scan(&exists)
	if err != nil && !errors.Is(err, context.Canceled) {
		return false, err
	}
	return exists, err
}
