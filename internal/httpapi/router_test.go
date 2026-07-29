package httpapi

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/whiterage/subscriptions-api/internal/service"
)

func TestProtectedRoutesRequireAPIKey(t *testing.T) {
	router := NewRouter(service.New(nil), slog.Default(), "secret")

	req := httptest.NewRequest(http.MethodGet, "/subscriptions", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHealthDoesNotRequireAPIKey(t *testing.T) {
	router := NewRouter(service.New(nil), slog.Default(), "secret")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCreateRejectsOversizedBody(t *testing.T) {
	router := NewRouter(service.New(nil), slog.Default(), "secret")
	body := strings.NewReader(`{"service_name":"` + strings.Repeat("a", maxJSONBodyBytes) + `"}`)

	req := httptest.NewRequest(http.MethodPost, "/subscriptions", body)
	req.Header.Set("X-API-Key", "secret")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "JSON body must not exceed") {
		t.Fatalf("expected size limit error, got %s", rec.Body.String())
	}
}
