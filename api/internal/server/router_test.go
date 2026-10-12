package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterHealthCheck(t *testing.T) {
	r := NewRouter(nil, nil, nil)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body != "API is running with live reload!" {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestRouterCORSPreflight(t *testing.T) {
	r := NewRouter(nil, nil, nil)

	req := httptest.NewRequest("OPTIONS", "/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://localhost:5173" {
		t.Errorf("expected Access-Control-Allow-Origin http://localhost:5173, got %s", origin)
	}

	creds := w.Header().Get("Access-Control-Allow-Credentials")
	if creds != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials true, got %s", creds)
	}
}
