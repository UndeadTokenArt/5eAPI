package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverMapBiomes(t *testing.T) {
	root := t.TempDir()

	mustMkdir := func(path string) {
		t.Helper()
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}
	mustWrite := func(path string) {
		t.Helper()
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	mustMkdir(filepath.Join(root, "dungeon"))
	mustWrite(filepath.Join(root, "dungeon", "d-floor.svg"))

	mustMkdir(filepath.Join(root, "outdoor"))
	mustWrite(filepath.Join(root, "outdoor", "o-grass.png"))

	mustMkdir(filepath.Join(root, "notes"))
	mustWrite(filepath.Join(root, "notes", "readme.txt"))

	biomes, err := discoverMapBiomes(root)
	if err != nil {
		t.Fatalf("discoverMapBiomes returned error: %v", err)
	}

	if len(biomes) != 2 {
		t.Fatalf("expected 2 biomes, got %d: %#v", len(biomes), biomes)
	}
	if biomes[0] != "dungeon" || biomes[1] != "outdoor" {
		t.Fatalf("unexpected biome order/content: %#v", biomes)
	}
}

func TestDiscoverMapBiomesMissingRootReturnsEmpty(t *testing.T) {
	biomes, err := discoverMapBiomes(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("discoverMapBiomes should not error for missing dir: %v", err)
	}
	if len(biomes) != 0 {
		t.Fatalf("expected empty biomes for missing root, got %#v", biomes)
	}
}

func TestMapHandlerRendersHTML(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/map", nil)
	rr := httptest.NewRecorder()

	mapHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("mapHandler status = %d, want %d", rr.Code, http.StatusOK)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Fatalf("mapHandler content-type = %q", ct)
	}
	body := rr.Body.String()
	if body == "" {
		t.Fatal("mapHandler returned empty body")
	}
	if !strings.Contains(body, "Map Generator") {
		t.Fatalf("mapHandler body missing expected text; got: %.120q", body)
	}
}

func TestMapEditorHandlerRendersHTML(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/map/editor", nil)
	rr := httptest.NewRecorder()

	mapEditorHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("mapEditorHandler status = %d, want %d", rr.Code, http.StatusOK)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Fatalf("mapEditorHandler content-type = %q", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Tile Editor") {
		t.Fatalf("mapEditorHandler body missing expected text; got: %.120q", body)
	}
}

func TestMapBiomesHandlerReturnsJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/map/biomes", nil)
	rr := httptest.NewRecorder()

	mapBiomesHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("mapBiomesHandler status = %d, want %d", rr.Code, http.StatusOK)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Fatalf("mapBiomesHandler content-type = %q", ct)
	}

	var biomes []string
	if err := json.Unmarshal(rr.Body.Bytes(), &biomes); err != nil {
		t.Fatalf("mapBiomesHandler returned invalid JSON: %v", err)
	}
	if len(biomes) == 0 {
		t.Fatal("mapBiomesHandler returned empty biome list")
	}
}
