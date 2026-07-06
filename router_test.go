package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterRejectsNonGET(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/map", nil)
	rr := httptest.NewRecorder()

	router(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("router status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestRouterMapRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/map", nil)
	rr := httptest.NewRecorder()

	router(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("router /map status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("router /map content-type = %q", got)
	}
}

func TestRouterMapEditorRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/map/editor", nil)
	rr := httptest.NewRecorder()

	router(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("router /map/editor status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("router /map/editor content-type = %q", got)
	}
}

func TestRouterMapBiomesRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/map/biomes", nil)
	rr := httptest.NewRecorder()

	router(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("router /api/map/biomes status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("router /api/map/biomes content-type = %q", got)
	}
}
