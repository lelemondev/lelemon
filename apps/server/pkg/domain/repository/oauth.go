package repository

import (
	"context"

	"github.com/lelemon/server/pkg/domain/entity"
)

// OAuthStore persists the OAuth 2.1 authorization-server state for the MCP (clients, codes,
// access + refresh tokens). It is intentionally NOT part of the composed Store interface: only
// the operational backends (SQLite, PostgreSQL) implement it, never the ClickHouse analytics
// store. Callers obtain it via a type assertion on the primary store:
//
//	oauthStore, ok := primaryStore.(repository.OAuthStore)
//
// ## Atomicity (security-critical)
// ConsumeAuthCode and ConsumeRefreshToken MUST be atomic single-use operations (conditional
// UPDATE on `… WHERE …_hash/id = ? AND consumed_at IS NULL`). Without it, two concurrent
// redemptions both succeed, defeating single-use codes and refresh-rotation theft detection.
type OAuthStore interface {
	// Clients (Dynamic Client Registration)
	InsertClient(ctx context.Context, c *entity.OAuthClient) error
	GetClientByID(ctx context.Context, clientID string) (*entity.OAuthClient, error)
	// FindClientsByName returns clients with the given name (empty string means the NULL name)
	// — used for idempotent DCR dedup.
	FindClientsByName(ctx context.Context, clientName string) ([]entity.OAuthClient, error)

	// Authorization codes
	InsertAuthCode(ctx context.Context, code *entity.OAuthAuthCode) error
	// ConsumeAuthCode atomically marks the code consumed and returns it; nil if missing or
	// already consumed.
	ConsumeAuthCode(ctx context.Context, codeHash string) (*entity.OAuthAuthCode, error)

	// Access tokens
	InsertAccessToken(ctx context.Context, t *entity.OAuthAccessToken) error
	GetAccessTokenByHash(ctx context.Context, tokenHash string) (*entity.OAuthAccessToken, error)

	// Refresh tokens (rotation + theft detection)
	InsertRefreshToken(ctx context.Context, t *entity.OAuthRefreshToken) (string, error)
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*entity.OAuthRefreshToken, error)
	// ConsumeRefreshToken atomically consumes the refresh; true only if this call won the race.
	ConsumeRefreshToken(ctx context.Context, id string) (bool, error)
	SetRefreshRotatedTo(ctx context.Context, id, rotatedToID string) error
	// RevokeChain revokes every access + refresh token for the subject + client (theft response).
	RevokeChain(ctx context.Context, subjectKey, clientID string) error
}
