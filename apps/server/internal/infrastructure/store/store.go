package store

import (
	"strings"

	"github.com/lelemon/server/internal/domain/repository"
	"github.com/lelemon/server/internal/infrastructure/store/clickhouse"
	"github.com/lelemon/server/internal/infrastructure/store/postgres"
	"github.com/lelemon/server/internal/infrastructure/store/sqlite"
)

// New creates a new store based on the database URL
func New(databaseURL string) (repository.Store, error) {
	switch {
	case strings.HasPrefix(databaseURL, "sqlite://"):
		path := strings.TrimPrefix(databaseURL, "sqlite://")
		return sqlite.New(path)

	case strings.HasPrefix(databaseURL, "postgres://"),
		strings.HasPrefix(databaseURL, "postgresql://"):
		return postgres.New(databaseURL)

	case strings.HasPrefix(databaseURL, "clickhouse://"),
		strings.HasPrefix(databaseURL, "clickhouses://"):
		return clickhouse.New(databaseURL)

	default:
		// Default to SQLite with the provided path
		return sqlite.New(databaseURL)
	}
}
