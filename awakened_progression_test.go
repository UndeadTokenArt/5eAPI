package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestAwakenedProgressionMatchesSpellcasting(t *testing.T) {
	t.Setenv("NEOTOKYO_DATA_DIR", builtinGuideDir)
	data, err := loadAwakenedProgression()
	if err != nil {
		t.Fatal(err)
	}
	if data.HitDie != 6 || len(data.Levels) != 20 {
		t.Fatalf("unexpected progression: %+v", data)
	}
	for _, level := range []int{2, 10, 17} {
		count := 0
		for _, feature := range data.Levels[level-1].Features {
			if feature.ID == "metamagic-"+strconv.Itoa(level) {
				count++
				if len(feature.Choices) != 1 || feature.Choices[0].Count == nil || *feature.Choices[0].Count != 2 {
					t.Fatalf("level %d Metamagic choices: %+v", level, feature.Choices)
				}
			}
		}
		if count != 1 {
			t.Fatalf("level %d has %d Metamagic features", level, count)
		}
	}
}

func TestAwakenedProgressionRejectsBadData(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NEOTOKYO_DATA_DIR", dir)
	path := filepath.Join(dir, "classesv2", "Awakened Mutant.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"name":"Awakened Mutant","hit_die":6,"levels":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAwakenedProgression(); err == nil {
		t.Fatal("invalid progression was accepted")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/awakened-mutant-progression", nil)
	req.Header.Set("Remote-User", "alice")
	rec := httptest.NewRecorder()
	setupRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("invalid progression endpoint status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAwakenedProgressionEndpoint(t *testing.T) {
	t.Setenv("NEOTOKYO_DATA_DIR", builtinGuideDir)
	t.Setenv("CHARACTER_GUEST_ACCESS", "")
	for _, tt := range []struct {
		user string
		want int
	}{{"", http.StatusUnauthorized}, {"alice", http.StatusOK}} {
		req := httptest.NewRequest(http.MethodGet, "/api/awakened-mutant-progression", nil)
		if tt.user != "" {
			req.Header.Set("Remote-User", tt.user)
		}
		rec := httptest.NewRecorder()
		setupRouter().ServeHTTP(rec, req)
		if rec.Code != tt.want {
			t.Fatalf("status %d, want %d: %s", rec.Code, tt.want, rec.Body.String())
		}
		if tt.want == http.StatusOK {
			var data awakenedProgression
			if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil || len(data.Levels) != 20 {
				t.Fatalf("invalid response: %v", err)
			}
		}
	}
}
