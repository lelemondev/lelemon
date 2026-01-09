package testing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// AssertStatus checks that the response has the expected status code.
func AssertStatus(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rr.Code != expected {
		t.Errorf("expected status %d, got %d. Body: %s", expected, rr.Code, rr.Body.String())
	}
}

// AssertJSON checks that the response is valid JSON and decodes it.
func AssertJSON(t *testing.T, rr *httptest.ResponseRecorder, v interface{}) {
	t.Helper()

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	if err := json.Unmarshal(rr.Body.Bytes(), v); err != nil {
		t.Errorf("failed to decode JSON response: %v. Body: %s", err, rr.Body.String())
	}
}

// AssertError checks that the response contains an error message.
func AssertError(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()
	AssertStatus(t, rr, expectedStatus)

	var resp struct {
		Error string `json:"error"`
	}
	AssertJSON(t, rr, &resp)

	if resp.Error == "" {
		t.Error("expected error message in response")
	}
}

// AssertNoError checks that the response does not contain an error.
func AssertNoError(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()

	if rr.Code >= 400 {
		t.Errorf("expected success status, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

// AssertOK is shorthand for AssertStatus with 200 OK.
func AssertOK(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	AssertStatus(t, rr, http.StatusOK)
}

// AssertCreated is shorthand for AssertStatus with 201 Created.
func AssertCreated(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	AssertStatus(t, rr, http.StatusCreated)
}

// AssertUnauthorized is shorthand for AssertStatus with 401 Unauthorized.
func AssertUnauthorized(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	AssertStatus(t, rr, http.StatusUnauthorized)
}

// AssertForbidden is shorthand for AssertStatus with 403 Forbidden.
func AssertForbidden(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	AssertStatus(t, rr, http.StatusForbidden)
}

// AssertNotFound is shorthand for AssertStatus with 404 Not Found.
func AssertNotFound(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	AssertStatus(t, rr, http.StatusNotFound)
}

// AssertBadRequest is shorthand for AssertStatus with 400 Bad Request.
func AssertBadRequest(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	AssertStatus(t, rr, http.StatusBadRequest)
}
