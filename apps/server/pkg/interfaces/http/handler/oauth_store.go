package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/lelemon/server/pkg/domain/entity"
	"github.com/lelemon/server/pkg/domain/repository"
)

// OAuthStoreHandler exposes repository.OAuthStore over a single internal RPC endpoint so the MCP
// authorization server (mcify, running out-of-process) can persist its clients/codes/tokens
// without direct DB access. Protected by ServiceAuth (shared secret). The JSON wire shapes match
// mcify's `OAuthStore` row types 1:1 (camelCase), so the TS adapter is a thin pass-through.
type OAuthStoreHandler struct {
	store repository.OAuthStore
}

func NewOAuthStoreHandler(store repository.OAuthStore) *OAuthStoreHandler {
	return &OAuthStoreHandler{store: store}
}

// ── Wire DTOs (== mcify OAuthStore rows) ─────────────────────────────────────────

type clientDTO struct {
	ClientID                string    `json:"clientId"`
	ClientName              *string   `json:"clientName"`
	RedirectURIs            []string  `json:"redirectUris"`
	GrantTypes              []string  `json:"grantTypes"`
	TokenEndpointAuthMethod string    `json:"tokenEndpointAuthMethod"`
	Scope                   *string   `json:"scope"`
	CreatedAt               time.Time `json:"createdAt"`
}

type authCodeDTO struct {
	CodeHash            string     `json:"codeHash"`
	ClientID            string     `json:"clientId"`
	SubjectKey          string     `json:"subjectKey"`
	RedirectURI         string     `json:"redirectUri"`
	CodeChallenge       string     `json:"codeChallenge"`
	CodeChallengeMethod string     `json:"codeChallengeMethod"`
	Scope               *string    `json:"scope"`
	ExpiresAt           time.Time  `json:"expiresAt"`
	ConsumedAt          *time.Time `json:"consumedAt,omitempty"`
}

type accessTokenDTO struct {
	TokenHash  string     `json:"tokenHash"`
	ClientID   string     `json:"clientId"`
	SubjectKey string     `json:"subjectKey"`
	Scope      *string    `json:"scope"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

type refreshTokenDTO struct {
	ID         string     `json:"id"`
	TokenHash  string     `json:"tokenHash"`
	ClientID   string     `json:"clientId"`
	SubjectKey string     `json:"subjectKey"`
	Scope      *string    `json:"scope"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	ConsumedAt *time.Time `json:"consumedAt,omitempty"`
}

func clientToDTO(c *entity.OAuthClient) *clientDTO {
	if c == nil {
		return nil
	}
	return &clientDTO{
		ClientID: c.ClientID, ClientName: c.ClientName, RedirectURIs: c.RedirectURIs,
		GrantTypes: c.GrantTypes, TokenEndpointAuthMethod: c.TokenEndpointAuthMethod,
		Scope: c.Scope, CreatedAt: c.CreatedAt,
	}
}

func (d *clientDTO) toEntity() *entity.OAuthClient {
	return &entity.OAuthClient{
		ClientID: d.ClientID, ClientName: d.ClientName, RedirectURIs: d.RedirectURIs,
		GrantTypes: d.GrantTypes, TokenEndpointAuthMethod: d.TokenEndpointAuthMethod,
		Scope: d.Scope, CreatedAt: d.CreatedAt,
	}
}

func accessToDTO(t *entity.OAuthAccessToken) *accessTokenDTO {
	if t == nil {
		return nil
	}
	return &accessTokenDTO{
		TokenHash: t.TokenHash, ClientID: t.ClientID, SubjectKey: t.SubjectKey,
		Scope: t.Scope, ExpiresAt: t.ExpiresAt, RevokedAt: t.RevokedAt,
	}
}

func refreshToDTO(t *entity.OAuthRefreshToken) *refreshTokenDTO {
	if t == nil {
		return nil
	}
	return &refreshTokenDTO{
		ID: t.ID, TokenHash: t.TokenHash, ClientID: t.ClientID, SubjectKey: t.SubjectKey,
		Scope: t.Scope, ExpiresAt: t.ExpiresAt, ConsumedAt: t.ConsumedAt,
	}
}

// ── RPC envelope ─────────────────────────────────────────────────────────────────

type oauthRPCRequest struct {
	Op           string           `json:"op"`
	Client       *clientDTO       `json:"client"`
	Code         *authCodeDTO     `json:"code"`
	AccessToken  *accessTokenDTO  `json:"accessToken"`
	RefreshToken *refreshTokenDTO `json:"refreshToken"`
	ClientID     string           `json:"clientId"`
	ClientName   string           `json:"clientName"`
	CodeHash     string           `json:"codeHash"`
	TokenHash    string           `json:"tokenHash"`
	ID           string           `json:"id"`
	RotatedToID  string           `json:"rotatedToId"`
	SubjectKey   string           `json:"subjectKey"`
}

func (h *OAuthStoreHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req oauthRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	ctx := r.Context()

	result, err := h.dispatch(ctx, &req)
	if err != nil {
		slog.Error("oauth store rpc failed", "op", req.Op, "error", err)
		http.Error(w, `{"error":"store operation failed"}`, http.StatusInternalServerError)
		return
	}
	if result == nil {
		result = map[string]any{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		slog.Error("oauth store rpc encode failed", "op", req.Op, "error", err)
	}
}

// dispatch maps an RPC op to an OAuthStore method. Returns the JSON-encodable result (or nil → {}).
func (h *OAuthStoreHandler) dispatch(ctx context.Context, req *oauthRPCRequest) (any, error) {
	s := h.store
	switch req.Op {
	case "insertClient":
		return nil, s.InsertClient(ctx, req.Client.toEntity())
	case "getClientById":
		c, err := s.GetClientByID(ctx, req.ClientID)
		return map[string]any{"client": clientToDTO(c)}, err
	case "findClientsByName":
		clients, err := s.FindClientsByName(ctx, req.ClientName)
		dtos := make([]clientDTO, 0, len(clients))
		for i := range clients {
			dtos = append(dtos, *clientToDTO(&clients[i]))
		}
		return map[string]any{"clients": dtos}, err
	case "insertAuthCode":
		c := req.Code
		return nil, s.InsertAuthCode(ctx, &entity.OAuthAuthCode{
			CodeHash: c.CodeHash, ClientID: c.ClientID, SubjectKey: c.SubjectKey,
			RedirectURI: c.RedirectURI, CodeChallenge: c.CodeChallenge,
			CodeChallengeMethod: c.CodeChallengeMethod, Scope: c.Scope, ExpiresAt: c.ExpiresAt,
		})
	case "consumeAuthCode":
		code, err := s.ConsumeAuthCode(ctx, req.CodeHash)
		var dto *authCodeDTO
		if code != nil {
			dto = &authCodeDTO{
				CodeHash: code.CodeHash, ClientID: code.ClientID, SubjectKey: code.SubjectKey,
				RedirectURI: code.RedirectURI, CodeChallenge: code.CodeChallenge,
				CodeChallengeMethod: code.CodeChallengeMethod, Scope: code.Scope,
				ExpiresAt: code.ExpiresAt, ConsumedAt: code.ConsumedAt,
			}
		}
		return map[string]any{"code": dto}, err
	case "insertAccessToken":
		t := req.AccessToken
		return nil, s.InsertAccessToken(ctx, &entity.OAuthAccessToken{
			TokenHash: t.TokenHash, ClientID: t.ClientID, SubjectKey: t.SubjectKey,
			Scope: t.Scope, ExpiresAt: t.ExpiresAt,
		})
	case "getAccessTokenByHash":
		t, err := s.GetAccessTokenByHash(ctx, req.TokenHash)
		return map[string]any{"token": accessToDTO(t)}, err
	case "insertRefreshToken":
		t := req.RefreshToken
		id, err := s.InsertRefreshToken(ctx, &entity.OAuthRefreshToken{
			ID: t.ID, TokenHash: t.TokenHash, ClientID: t.ClientID, SubjectKey: t.SubjectKey,
			Scope: t.Scope, ExpiresAt: t.ExpiresAt,
		})
		return map[string]any{"id": id}, err
	case "getRefreshTokenByHash":
		t, err := s.GetRefreshTokenByHash(ctx, req.TokenHash)
		return map[string]any{"token": refreshToDTO(t)}, err
	case "consumeRefreshToken":
		consumed, err := s.ConsumeRefreshToken(ctx, req.ID)
		return map[string]any{"consumed": consumed}, err
	case "setRefreshRotatedTo":
		return nil, s.SetRefreshRotatedTo(ctx, req.ID, req.RotatedToID)
	case "revokeChain":
		return nil, s.RevokeChain(ctx, req.SubjectKey, req.ClientID)
	default:
		return nil, errUnknownOp
	}
}

var errUnknownOp = &opError{"unknown op"}

type opError struct{ msg string }

func (e *opError) Error() string { return e.msg }
