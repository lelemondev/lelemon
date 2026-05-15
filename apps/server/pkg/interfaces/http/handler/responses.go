package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Shared JSON response helpers for the newer handlers (datasets, evals…).
// The older handlers in this package use bare `http.Error` + inline JSON —
// see them for context, but new code should funnel through these so we always
// honour the "check json.Encode errors" rule from .claude/rules/anti-patterns.md.

// writeJSON serialises `body` to JSON with the given status. On encoder
// failure the response is already half-sent (headers written), so we log and
// move on — there's no way to roll back a chunked write.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("failed to encode response", "err", err)
	}
}

// writeJSONError writes a simple `{"error": "..."}` body. Same caveat as
// writeJSON on encoder errors.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		slog.Error("failed to encode error response", "err", err)
	}
}
