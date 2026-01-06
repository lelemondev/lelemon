package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/lelemon/server/internal/application/analytics"
	appauth "github.com/lelemon/server/internal/application/auth"
	"github.com/lelemon/server/internal/application/ingest"
	"github.com/lelemon/server/internal/application/project"
	"github.com/lelemon/server/internal/application/trace"
	"github.com/lelemon/server/internal/domain/service"
	"github.com/lelemon/server/internal/infrastructure/auth"
	"github.com/lelemon/server/internal/infrastructure/store/sqlite"
	apphttp "github.com/lelemon/server/internal/interfaces/http"
)

// TestServer wraps a test HTTP server with helper methods
type TestServer struct {
	*httptest.Server
	t *testing.T
}

// setupTestServer creates a new test server with a fresh database
func setupTestServer(t *testing.T) *TestServer {
	t.Helper()

	// Create temp database
	tmpDB := t.TempDir() + "/test.db"

	store, err := sqlite.New(tmpDB)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
		os.Remove(tmpDB)
	})

	// Create services
	jwtService := auth.NewJWTService("test-secret", 24*60*60*1000000000) // 24h in ns
	oauthService := auth.NewOAuthService("", "", "")
	pricing := service.NewPricingCalculator()

	ingestSvc := ingest.NewService(store, pricing)
	traceSvc := trace.NewService(store, pricing)
	analyticsSvc := analytics.NewService(store)
	projectSvc := project.NewService(store)
	authSvc := appauth.NewService(store, jwtService, oauthService)

	router := apphttp.NewRouter(apphttp.RouterConfig{
		PrimaryStore:   store,
		AnalyticsStore: store, // Same store for tests
		IngestSvc:      ingestSvc,
		TraceSvc:       traceSvc,
		AnalyticsSvc:   analyticsSvc,
		ProjectSvc:     projectSvc,
		AuthSvc:        authSvc,
		JWTService:     jwtService,
		FrontendURL:    "http://localhost:3000",
	})

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return &TestServer{Server: server, t: t}
}

// Request makes an HTTP request and returns the response
func (ts *TestServer) Request(method, path string, body any, headers map[string]string) *http.Response {
	ts.t.Helper()

	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, ts.URL+path, reqBody)
	if err != nil {
		ts.t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatalf("request failed: %v", err)
	}

	return resp
}

// ParseJSON parses the response body into the given struct
func ParseJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
}

// AuthResponse for parsing auth responses
type AuthResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"user"`
}

// ProjectResponse for parsing project responses
type ProjectResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	APIKey string `json:"apiKey"`
}

// IngestResponse for parsing ingest responses
type IngestResponse struct {
	Success   bool `json:"success"`
	Processed int  `json:"processed"`
}

// TracesResponse for parsing traces list
type TracesResponse struct {
	Data  []TraceData `json:"Data"`
	Total int         `json:"Total"`
}

type TraceData struct {
	ID             string  `json:"ID"`
	TotalSpans     int     `json:"TotalSpans"`
	TotalTokens    int     `json:"TotalTokens"`
	TotalCostUSD   float64 `json:"TotalCostUSD"`
}

// StatsResponse for parsing analytics
type StatsResponse struct {
	TotalTraces  int     `json:"TotalTraces"`
	TotalSpans   int     `json:"TotalSpans"`
	TotalTokens  int     `json:"TotalTokens"`
	TotalCostUSD float64 `json:"TotalCostUSD"`
}
