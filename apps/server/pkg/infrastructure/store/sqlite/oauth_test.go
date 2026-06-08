package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/lelemon/server/pkg/domain/entity"
)

func newOAuthStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return s
}

func strptr(s string) *string { return &s }

func TestOAuthClients(t *testing.T) {
	ctx := context.Background()
	s := newOAuthStore(t)

	c := &entity.OAuthClient{
		ClientID:                "mcp-abc",
		ClientName:              strptr("Claude"),
		RedirectURIs:            []string{"https://claude.ai/cb"},
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethod: "none",
	}
	if err := s.InsertClient(ctx, c); err != nil {
		t.Fatalf("InsertClient: %v", err)
	}

	got, err := s.GetClientByID(ctx, "mcp-abc")
	if err != nil || got == nil {
		t.Fatalf("GetClientByID: %v / %v", err, got)
	}
	if got.ClientName == nil || *got.ClientName != "Claude" {
		t.Errorf("client_name = %v", got.ClientName)
	}
	if len(got.RedirectURIs) != 1 || got.RedirectURIs[0] != "https://claude.ai/cb" {
		t.Errorf("redirect_uris = %v", got.RedirectURIs)
	}
	if len(got.GrantTypes) != 2 {
		t.Errorf("grant_types = %v", got.GrantTypes)
	}

	byName, err := s.FindClientsByName(ctx, "Claude")
	if err != nil || len(byName) != 1 {
		t.Fatalf("FindClientsByName: %v / %d", err, len(byName))
	}

	missing, err := s.GetClientByID(ctx, "nope")
	if err != nil || missing != nil {
		t.Errorf("expected nil for missing client, got %v / %v", missing, err)
	}
}

func TestOAuthAuthCodeSingleUse(t *testing.T) {
	ctx := context.Background()
	s := newOAuthStore(t)

	code := &entity.OAuthAuthCode{
		CodeHash:            "codehash-1",
		ClientID:            "mcp-abc",
		SubjectKey:          `{"projectId":"p1","userId":"u1"}`,
		RedirectURI:         "https://claude.ai/cb",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		ExpiresAt:           time.Now().Add(time.Minute),
	}
	if err := s.InsertAuthCode(ctx, code); err != nil {
		t.Fatalf("InsertAuthCode: %v", err)
	}

	first, err := s.ConsumeAuthCode(ctx, "codehash-1")
	if err != nil || first == nil {
		t.Fatalf("first consume: %v / %v", err, first)
	}
	if first.SubjectKey != code.SubjectKey {
		t.Errorf("subject_key = %q", first.SubjectKey)
	}

	// Single-use: the second consume must return nil.
	second, err := s.ConsumeAuthCode(ctx, "codehash-1")
	if err != nil {
		t.Fatalf("second consume err: %v", err)
	}
	if second != nil {
		t.Error("expected nil on second consume (single-use)")
	}
}

func TestOAuthRefreshRotationAndRevoke(t *testing.T) {
	ctx := context.Background()
	s := newOAuthStore(t)
	const subject = `{"projectId":"p1","userId":"u1"}`

	// Issue an access + refresh pair.
	if err := s.InsertAccessToken(ctx, &entity.OAuthAccessToken{
		TokenHash: "access-1", ClientID: "mcp-abc", SubjectKey: subject, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("InsertAccessToken: %v", err)
	}
	id, err := s.InsertRefreshToken(ctx, &entity.OAuthRefreshToken{
		TokenHash: "refresh-1", ClientID: "mcp-abc", SubjectKey: subject, ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	if err != nil || id == "" {
		t.Fatalf("InsertRefreshToken: %v / %q", err, id)
	}

	// First rotation wins; second (reuse) loses.
	ok, err := s.ConsumeRefreshToken(ctx, id)
	if err != nil || !ok {
		t.Fatalf("first consume: %v / %v", err, ok)
	}
	ok, err = s.ConsumeRefreshToken(ctx, id)
	if err != nil {
		t.Fatalf("second consume err: %v", err)
	}
	if ok {
		t.Error("expected false on refresh reuse (already consumed)")
	}

	// Theft response: revoke the chain → access token is revoked.
	if err := s.RevokeChain(ctx, subject, "mcp-abc"); err != nil {
		t.Fatalf("RevokeChain: %v", err)
	}
	at, err := s.GetAccessTokenByHash(ctx, "access-1")
	if err != nil || at == nil {
		t.Fatalf("GetAccessTokenByHash: %v / %v", err, at)
	}
	if at.RevokedAt == nil {
		t.Error("expected access token to be revoked after RevokeChain")
	}
}

func TestOAuthAccessTokenLookup(t *testing.T) {
	ctx := context.Background()
	s := newOAuthStore(t)

	if err := s.InsertAccessToken(ctx, &entity.OAuthAccessToken{
		TokenHash: "access-x", ClientID: "mcp-abc", SubjectKey: `{"u":"1"}`, Scope: strptr("read"), ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("InsertAccessToken: %v", err)
	}
	got, err := s.GetAccessTokenByHash(ctx, "access-x")
	if err != nil || got == nil {
		t.Fatalf("GetAccessTokenByHash: %v / %v", err, got)
	}
	if got.Scope == nil || *got.Scope != "read" {
		t.Errorf("scope = %v", got.Scope)
	}
	missing, err := s.GetAccessTokenByHash(ctx, "nope")
	if err != nil || missing != nil {
		t.Errorf("expected nil for missing token, got %v / %v", missing, err)
	}
}
