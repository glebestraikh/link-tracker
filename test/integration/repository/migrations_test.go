//go:build integration

package repository_test

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	botdb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/database/bot"
	scrapperdb "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/database/scrapper"
)

const tableExistsQuery = `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name = $1
		)
	`

func TestMigrations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		database   string
		migrations fs.FS
		expected   []string
	}{
		{
			name:       "bot",
			database:   "bot_migration_test",
			migrations: botdb.Migrations,
			expected:   []string{"users", "chats"},
		},
		{
			name:       "scrapper",
			database:   "scrapper_migration_test",
			migrations: scrapperdb.Migrations,
			expected:   []string{"chats", "links", "link_chat", "tags", "link_chat_tag"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ensureDatabase(t, tt.database)

			source, err := iofs.New(tt.migrations, "migrations")
			require.NoError(t, err)
			m, err := migrate.NewWithSourceInstance("iofs", source, dbURL(tt.database))
			require.NoError(t, err)
			t.Cleanup(func() { _, _ = m.Close() })

			require.NoError(t, m.Up())

			pool, err := pgxpool.New(t.Context(), dbURL(tt.database))
			require.NoError(t, err)
			t.Cleanup(pool.Close)

			for _, table := range tt.expected {
				assertTableExists(t, pool, table)
			}

			err = m.Down()
			require.True(t, err == nil || errors.Is(err, migrate.ErrNoChange))

			for _, table := range tt.expected {
				assertTableMissing(t, pool, table)
			}
		})
	}
}

func assertTableExists(t *testing.T, pool *pgxpool.Pool, table string) {
	t.Helper()
	var exists bool
	err := pool.QueryRow(t.Context(), tableExistsQuery, table).Scan(&exists)
	require.NoError(t, err)
	require.True(t, exists, fmt.Sprintf("table %q expected after migrations", table))
}

func assertTableMissing(t *testing.T, pool *pgxpool.Pool, table string) {
	t.Helper()
	var exists bool
	err := pool.QueryRow(t.Context(), tableExistsQuery, table).Scan(&exists)
	require.NoError(t, err)
	require.False(t, exists, fmt.Sprintf("table %q expected to be dropped", table))
}
