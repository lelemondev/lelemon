package entity

import "time"

// OAuth 2.1 authorization-server persistence entities. The Lelemon MCP server (built on mcify)
// is the authorization server; the Go backend only stores its clients/codes/tokens on its behalf.
// Tokens and authorization codes are stored HASHED (SHA-256, base64url) — the plaintext never
// reaches the backend. The authenticated principal is opaque here: a canonical JSON `SubjectKey`
// (e.g. `{"projectId":"...","userId":"..."}`) the MCP binds to the token.

// OAuthClient is a client registered via Dynamic Client Registration (RFC 7591). Public client:
// no secret. `ClientName` and `Scope` are nullable.
type OAuthClient struct {
	ClientID                string
	ClientName              *string
	RedirectURIs            []string
	GrantTypes              []string
	TokenEndpointAuthMethod string
	Scope                   *string
	CreatedAt               time.Time
}

// OAuthAuthCode is a single-use authorization code (hashed). `ConsumedAt` flips exactly once.
type OAuthAuthCode struct {
	CodeHash            string
	ClientID            string
	SubjectKey          string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               *string
	ExpiresAt           time.Time
	ConsumedAt          *time.Time
}

// OAuthAccessToken is an issued access token (hashed). `RevokedAt` set on chain revocation.
type OAuthAccessToken struct {
	TokenHash  string
	ClientID   string
	SubjectKey string
	Scope      *string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
}

// OAuthRefreshToken is an issued refresh token (hashed). Rotated single-use via `ConsumedAt`;
// `RotatedToID` links to its successor for audit/theft tracing.
type OAuthRefreshToken struct {
	ID          string
	TokenHash   string
	ClientID    string
	SubjectKey  string
	Scope       *string
	ExpiresAt   time.Time
	ConsumedAt  *time.Time
	RotatedToID *string
}
