package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func withOrigins(t *testing.T, value string) {
	t.Helper()
	original := allowedOrigins
	allowedOrigins = corsAllowedOriginsWith(value)
	t.Cleanup(func() { allowedOrigins = original })
}

func TestCorsAllowedOrigins_DefaultWhenUnset(t *testing.T) {
	withOrigins(t, "")

	if !allowedOrigins["https://parapet.digital"] {
		t.Errorf("expected default origin https://parapet.digital, got %v", allowedOrigins)
	}
	if len(allowedOrigins) != 1 {
		t.Errorf("expected exactly one default origin, got %v", allowedOrigins)
	}
}

func TestCorsAllowedOrigins_ParsesCommaSeparatedList(t *testing.T) {
	withOrigins(t, "https://a.example, https://b.example ,  ,https://c.example")

	for _, origin := range []string{"https://a.example", "https://b.example", "https://c.example"} {
		if !allowedOrigins[origin] {
			t.Errorf("expected %s to be allowed, got %v", origin, allowedOrigins)
		}
	}
	if len(allowedOrigins) != 3 {
		t.Errorf("expected 3 allowed origins after trimming, got %v", allowedOrigins)
	}
}

func TestCorsAllowedOrigins_BlankValueFallsBackToDefault(t *testing.T) {
	withOrigins(t, " , ,")

	if !allowedOrigins["https://parapet.digital"] {
		t.Errorf("expected default origin for blank CORS_ALLOWED_ORIGINS, got %v", allowedOrigins)
	}
}

func TestCorsMiddleware_EchoesAllowedOrigin(t *testing.T) {
	withOrigins(t, "https://app.example")
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://app.example")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Errorf("expected allowed origin to be echoed, got %q", got)
	}
	if w.Header().Values("Vary")[0] != "Origin" {
		t.Errorf("expected Vary: Origin header, got %v", w.Header().Values("Vary"))
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected request to pass through, got status %d", w.Code)
	}
}

func TestCorsMiddleware_OmitsHeaderForDisallowedOrigin(t *testing.T) {
	withOrigins(t, "https://app.example")
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no CORS header for disallowed origin, got %q", got)
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected request to still pass through, got status %d", w.Code)
	}
}

func TestCorsMiddleware_NoOriginHeaderPassesThrough(t *testing.T) {
	withOrigins(t, "https://app.example")
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no CORS header without an Origin, got %q", got)
	}
}

func TestCorsMiddleware_PreflightReturnsNoContent(t *testing.T) {
	withOrigins(t, "https://app.example")
	called := false
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://app.example")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if called {
		t.Errorf("preflight request should not reach the wrapped handler")
	}
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for preflight, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
		t.Errorf("expected allowed methods header, got %q", w.Header().Get("Access-Control-Allow-Methods"))
	}
	if w.Header().Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Errorf("expected allowed headers, got %q", w.Header().Get("Access-Control-Allow-Headers"))
	}
}
