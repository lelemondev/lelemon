package testing

import (
	"context"
	"testing"

	"github.com/lelemon/server/pkg/domain/repository"
	"github.com/lelemon/server/pkg/infrastructure/store/sqlite"
)

// TestDB creates an in-memory SQLite database for testing.
// The database is automatically cleaned up when the test completes.
func TestDB(t *testing.T) repository.Store {
	t.Helper()

	// Use in-memory SQLite for tests
	s, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Run migrations
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Cleanup on test completion
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("failed to close test database: %v", err)
		}
	})

	return s
}

// TestDBWithData creates a test database with pre-populated data.
func TestDBWithData(t *testing.T, setup func(ctx context.Context, s repository.Store) error) repository.Store {
	t.Helper()

	s := TestDB(t)

	ctx := context.Background()
	if err := setup(ctx, s); err != nil {
		t.Fatalf("failed to setup test data: %v", err)
	}

	return s
}
