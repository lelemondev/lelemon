package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lelemon/server/pkg/domain/entity"
)

// SQLite implementation of repository.OAuthStore — the MCP's OAuth 2.1 authorization-server state.
// Tokens/codes are stored hashed; `subject_key` is the opaque canonical-JSON identity the MCP binds.

// migrateOAuth creates the four OAuth tables. Called from Migrate; idempotent.
func (s *Store) migrateOAuth(ctx context.Context) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS mcp_clients (
			client_id TEXT PRIMARY KEY,
			client_name TEXT,
			redirect_uris TEXT NOT NULL DEFAULT '[]',
			grant_types TEXT NOT NULL DEFAULT '[]',
			token_endpoint_auth_method TEXT NOT NULL DEFAULT 'none',
			scope TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS mcp_authorization_codes (
			code_hash TEXT PRIMARY KEY,
			client_id TEXT NOT NULL,
			subject_key TEXT NOT NULL,
			redirect_uri TEXT NOT NULL,
			code_challenge TEXT NOT NULL,
			code_challenge_method TEXT NOT NULL DEFAULT 'S256',
			scope TEXT,
			expires_at DATETIME NOT NULL,
			consumed_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS mcp_access_tokens (
			token_hash TEXT PRIMARY KEY,
			client_id TEXT NOT NULL,
			subject_key TEXT NOT NULL,
			scope TEXT,
			expires_at DATETIME NOT NULL,
			revoked_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS mcp_refresh_tokens (
			id TEXT PRIMARY KEY,
			token_hash TEXT UNIQUE NOT NULL,
			client_id TEXT NOT NULL,
			subject_key TEXT NOT NULL,
			scope TEXT,
			expires_at DATETIME NOT NULL,
			consumed_at DATETIME,
			rotated_to_id TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mcp_clients_name ON mcp_clients(client_name)`,
		`CREATE INDEX IF NOT EXISTS idx_mcp_access_subject ON mcp_access_tokens(subject_key, client_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mcp_refresh_subject ON mcp_refresh_tokens(subject_key, client_id)`,
	}
	for _, m := range migrations {
		if _, err := s.db.ExecContext(ctx, m); err != nil {
			return fmt.Errorf("oauth migration failed: %w", err)
		}
	}
	return nil
}

// ── Clients ─────────────────────────────────────────────────────────────────────

func (s *Store) InsertClient(ctx context.Context, c *entity.OAuthClient) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	redirects, _ := json.Marshal(c.RedirectURIs)
	grants, _ := json.Marshal(c.GrantTypes)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mcp_clients (client_id, client_name, redirect_uris, grant_types, token_endpoint_auth_method, scope, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, c.ClientID, c.ClientName, string(redirects), string(grants), c.TokenEndpointAuthMethod, c.Scope, c.CreatedAt)
	if err != nil {
		return fmt.Errorf("InsertClient: %w", err)
	}
	return nil
}

func scanClient(row interface{ Scan(...any) error }) (*entity.OAuthClient, error) {
	var c entity.OAuthClient
	var clientName, scope sql.NullString
	var redirects, grants string
	if err := row.Scan(&c.ClientID, &clientName, &redirects, &grants, &c.TokenEndpointAuthMethod, &scope, &c.CreatedAt); err != nil {
		return nil, err
	}
	if clientName.Valid {
		c.ClientName = &clientName.String
	}
	if scope.Valid {
		c.Scope = &scope.String
	}
	json.Unmarshal([]byte(redirects), &c.RedirectURIs)
	json.Unmarshal([]byte(grants), &c.GrantTypes)
	return &c, nil
}

func (s *Store) GetClientByID(ctx context.Context, clientID string) (*entity.OAuthClient, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT client_id, client_name, redirect_uris, grant_types, token_endpoint_auth_method, scope, created_at
		FROM mcp_clients WHERE client_id = ?
	`, clientID)
	c, err := scanClient(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetClientByID: %w", err)
	}
	return c, nil
}

func (s *Store) FindClientsByName(ctx context.Context, clientName string) ([]entity.OAuthClient, error) {
	// Empty string means the NULL client_name bucket.
	var rows *sql.Rows
	var err error
	const base = `SELECT client_id, client_name, redirect_uris, grant_types, token_endpoint_auth_method, scope, created_at FROM mcp_clients `
	if clientName == "" {
		rows, err = s.db.QueryContext(ctx, base+`WHERE client_name IS NULL`)
	} else {
		rows, err = s.db.QueryContext(ctx, base+`WHERE client_name = ?`, clientName)
	}
	if err != nil {
		return nil, fmt.Errorf("FindClientsByName: %w", err)
	}
	defer rows.Close()

	var clients []entity.OAuthClient
	for rows.Next() {
		c, err := scanClient(rows)
		if err != nil {
			return nil, fmt.Errorf("FindClientsByName scan: %w", err)
		}
		clients = append(clients, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("FindClientsByName iteration: %w", err)
	}
	return clients, nil
}

// ── Authorization codes ──────────────────────────────────────────────────────────

func (s *Store) InsertAuthCode(ctx context.Context, code *entity.OAuthAuthCode) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mcp_authorization_codes (code_hash, client_id, subject_key, redirect_uri, code_challenge, code_challenge_method, scope, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, code.CodeHash, code.ClientID, code.SubjectKey, code.RedirectURI, code.CodeChallenge, code.CodeChallengeMethod, code.Scope, code.ExpiresAt)
	if err != nil {
		return fmt.Errorf("InsertAuthCode: %w", err)
	}
	return nil
}

func (s *Store) ConsumeAuthCode(ctx context.Context, codeHash string) (*entity.OAuthAuthCode, error) {
	// Atomic single-use: only the caller that flips consumed_at from NULL proceeds.
	res, err := s.db.ExecContext(ctx, `
		UPDATE mcp_authorization_codes SET consumed_at = ? WHERE code_hash = ? AND consumed_at IS NULL
	`, time.Now(), codeHash)
	if err != nil {
		return nil, fmt.Errorf("ConsumeAuthCode update: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, nil // missing or already consumed
	}
	var c entity.OAuthAuthCode
	var scope sql.NullString
	var consumedAt sql.NullTime
	err = s.db.QueryRowContext(ctx, `
		SELECT code_hash, client_id, subject_key, redirect_uri, code_challenge, code_challenge_method, scope, expires_at, consumed_at
		FROM mcp_authorization_codes WHERE code_hash = ?
	`, codeHash).Scan(&c.CodeHash, &c.ClientID, &c.SubjectKey, &c.RedirectURI, &c.CodeChallenge, &c.CodeChallengeMethod, &scope, &c.ExpiresAt, &consumedAt)
	if err != nil {
		return nil, fmt.Errorf("ConsumeAuthCode select: %w", err)
	}
	if scope.Valid {
		c.Scope = &scope.String
	}
	if consumedAt.Valid {
		c.ConsumedAt = &consumedAt.Time
	}
	return &c, nil
}

// ── Access tokens ─────────────────────────────────────────────────────────────────

func (s *Store) InsertAccessToken(ctx context.Context, t *entity.OAuthAccessToken) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mcp_access_tokens (token_hash, client_id, subject_key, scope, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, t.TokenHash, t.ClientID, t.SubjectKey, t.Scope, t.ExpiresAt)
	if err != nil {
		return fmt.Errorf("InsertAccessToken: %w", err)
	}
	return nil
}

func (s *Store) GetAccessTokenByHash(ctx context.Context, tokenHash string) (*entity.OAuthAccessToken, error) {
	var t entity.OAuthAccessToken
	var scope sql.NullString
	var revokedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT token_hash, client_id, subject_key, scope, expires_at, revoked_at
		FROM mcp_access_tokens WHERE token_hash = ?
	`, tokenHash).Scan(&t.TokenHash, &t.ClientID, &t.SubjectKey, &scope, &t.ExpiresAt, &revokedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetAccessTokenByHash: %w", err)
	}
	if scope.Valid {
		t.Scope = &scope.String
	}
	if revokedAt.Valid {
		t.RevokedAt = &revokedAt.Time
	}
	return &t, nil
}

// ── Refresh tokens ─────────────────────────────────────────────────────────────────

func (s *Store) InsertRefreshToken(ctx context.Context, t *entity.OAuthRefreshToken) (string, error) {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mcp_refresh_tokens (id, token_hash, client_id, subject_key, scope, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, t.ID, t.TokenHash, t.ClientID, t.SubjectKey, t.Scope, t.ExpiresAt)
	if err != nil {
		return "", fmt.Errorf("InsertRefreshToken: %w", err)
	}
	return t.ID, nil
}

func (s *Store) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*entity.OAuthRefreshToken, error) {
	var t entity.OAuthRefreshToken
	var scope sql.NullString
	var consumedAt sql.NullTime
	var rotatedTo sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, token_hash, client_id, subject_key, scope, expires_at, consumed_at, rotated_to_id
		FROM mcp_refresh_tokens WHERE token_hash = ?
	`, tokenHash).Scan(&t.ID, &t.TokenHash, &t.ClientID, &t.SubjectKey, &scope, &t.ExpiresAt, &consumedAt, &rotatedTo)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetRefreshTokenByHash: %w", err)
	}
	if scope.Valid {
		t.Scope = &scope.String
	}
	if consumedAt.Valid {
		t.ConsumedAt = &consumedAt.Time
	}
	if rotatedTo.Valid {
		t.RotatedToID = &rotatedTo.String
	}
	return &t, nil
}

func (s *Store) ConsumeRefreshToken(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE mcp_refresh_tokens SET consumed_at = ? WHERE id = ? AND consumed_at IS NULL
	`, time.Now(), id)
	if err != nil {
		return false, fmt.Errorf("ConsumeRefreshToken: %w", err)
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (s *Store) SetRefreshRotatedTo(ctx context.Context, id, rotatedToID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE mcp_refresh_tokens SET rotated_to_id = ? WHERE id = ?
	`, rotatedToID, id)
	if err != nil {
		return fmt.Errorf("SetRefreshRotatedTo: %w", err)
	}
	return nil
}

func (s *Store) RevokeChain(ctx context.Context, subjectKey, clientID string) error {
	now := time.Now()
	if _, err := s.db.ExecContext(ctx, `
		UPDATE mcp_access_tokens SET revoked_at = ? WHERE subject_key = ? AND client_id = ? AND revoked_at IS NULL
	`, now, subjectKey, clientID); err != nil {
		return fmt.Errorf("RevokeChain access: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		UPDATE mcp_refresh_tokens SET consumed_at = ? WHERE subject_key = ? AND client_id = ? AND consumed_at IS NULL
	`, now, subjectKey, clientID); err != nil {
		return fmt.Errorf("RevokeChain refresh: %w", err)
	}
	return nil
}
