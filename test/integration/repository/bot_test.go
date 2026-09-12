//go:build integration

package repository_test

import (
	"context"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/out/repository/pgsql"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/adapter/out/repository/squirrel"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	botservice "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
)

type botImpl struct {
	name     string
	userRepo botservice.UserRepository
	chatRepo botservice.ChatRepository
}

func botImpls(pool *pgxpool.Pool) []botImpl {
	return []botImpl{
		{
			name:     "sql",
			userRepo: pgsql.NewUserRepository(pool),
			chatRepo: pgsql.NewChatRepository(pool),
		},
		{
			name:     "squirrel",
			userRepo: squirrel.NewUserRepository(pool),
			chatRepo: squirrel.NewChatRepository(pool),
		},
	}
}

func TestBotUserRepository(t *testing.T) {
	pool := botPool(t)

	for _, impl := range botImpls(pool) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			truncate(t, pool, "users", "chats")

			t.Run("Save then Exists returns true", func(t *testing.T) {
				require.NoError(t, impl.userRepo.Save(ctx, model.User{ID: 1, Username: "alice"}))
				exists, err := impl.userRepo.Exists(ctx, 1)
				require.NoError(t, err)
				require.True(t, exists)
			})

			t.Run("Exists returns false for unknown id", func(t *testing.T) {
				exists, err := impl.userRepo.Exists(ctx, 99999)
				require.NoError(t, err)
				require.False(t, exists)
			})

			t.Run("Save twice upserts username", func(t *testing.T) {
				require.NoError(t, impl.userRepo.Save(ctx, model.User{ID: 2, Username: "old"}))
				require.NoError(t, impl.userRepo.Save(ctx, model.User{ID: 2, Username: "new"}))

				name, err := readUsername(ctx, pool, 2)
				require.NoError(t, err)
				require.Equal(t, "new", name)
			})
		})
	}
}

func TestBotChatRepository(t *testing.T) {
	pool := botPool(t)

	for _, impl := range botImpls(pool) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			truncate(t, pool, "users", "chats")

			t.Run("Save then Exists returns true", func(t *testing.T) {
				require.NoError(t, impl.chatRepo.Save(ctx, 12345))
				exists, err := impl.chatRepo.Exists(ctx, 12345)
				require.NoError(t, err)
				require.True(t, exists)
			})

			t.Run("Save duplicate is idempotent", func(t *testing.T) {
				require.NoError(t, impl.chatRepo.Save(ctx, 67890))
				require.NoError(t, impl.chatRepo.Save(ctx, 67890))
			})

			t.Run("Exists returns false for unknown chat", func(t *testing.T) {
				exists, err := impl.chatRepo.Exists(ctx, 11111)
				require.NoError(t, err)
				require.False(t, exists)
			})
		})
	}
}

func readUsername(ctx context.Context, pool *pgxpool.Pool, id int64) (string, error) {
	var name string
	err := pool.QueryRow(ctx, `SELECT username FROM users WHERE id = $1`, id).Scan(&name)
	return name, err
}
