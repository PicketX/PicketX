package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	h := NewHandler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("health response: %d %v", w.Code, w.Header())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body["status"] != "ok" {
		t.Fatalf("invalid health response: %s, %v", w.Body.String(), err)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("liveness must not be cached")
	}
}

func TestHandlerRejectsUnknownPathsAndMethods(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/", http.StatusNotFound},
		{http.MethodGet, "/api/grants", http.StatusNotFound},
		{http.MethodGet, "/healthz/extra", http.StatusNotFound},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			NewHandler().ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("got %d; want %d", w.Code, tc.status)
			}
		})
	}
}

func TestHealthHead(t *testing.T) {
	w := httptest.NewRecorder()
	NewHandler().ServeHTTP(w, httptest.NewRequest(http.MethodHead, "/healthz", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("HEAD status: got %d; want %d", w.Code, http.StatusOK)
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("HEAD must preserve health response headers: %v", w.Header())
	}
	if w.Body.Len() != 0 {
		t.Fatalf("HEAD must not return a body: %q", w.Body.String())
	}
}
