package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassProgressionPreviews(t *testing.T) {
	t.Setenv("NEOTOKYO_DATA_DIR", builtinGuideDir)
	labels, err := classLabels()
	if err != nil {
		t.Fatal(err)
	}
	for original := range progressionFiles2024 {
		t.Run(original, func(t *testing.T) {
			preview, err := loadClassProgressionPreview(original)
			if err != nil {
				t.Fatal(err)
			}
			if originalClassFor(labels[original], labels) != original {
				t.Fatalf("mapping %q does not resolve to %s", labels[original], original)
			}
			if original == "Barbarian" && preview.Name != "Cyber-Berserker" {
				t.Errorf("wrong Barbarian identity: %s", preview.Name)
			}
			if original == "Fighter" && preview.Name != "Corporate Enforcer" {
				t.Errorf("wrong Fighter identity: %s", preview.Name)
			}
			if original == "Cleric" && preview.Name != "Shrine Priest" {
				t.Errorf("wrong Cleric identity: %s", preview.Name)
			}
		})
	}
}

func TestClassProgressionPreviewEndpoint(t *testing.T) {
	t.Setenv("NEOTOKYO_DATA_DIR", builtinGuideDir)
	t.Setenv("CHARACTER_GUEST_ACCESS", "")
	for _, tt := range []struct {
		class string
		user  string
		want  int
	}{
		{"Awakened Mutant", "", http.StatusUnauthorized},
		{"not a class", "alice", http.StatusBadRequest},
		{"Corporate Enforcer", "alice", http.StatusOK},
		{"Cyber-Berserker", "alice", http.StatusOK},
		{"Shrine Priest", "alice", http.StatusOK},
	} {
		req := httptest.NewRequest(http.MethodGet, "/api/class-progression-preview?class="+url.QueryEscape(tt.class), nil)
		req.Header.Set("Remote-User", tt.user)
		rec := httptest.NewRecorder()
		setupRouter().ServeHTTP(rec, req)
		if rec.Code != tt.want {
			t.Fatalf("%s: status %d, want %d: %s", tt.class, rec.Code, tt.want, rec.Body.String())
		}
		if tt.want == http.StatusOK {
			var data classProgressionPreview
			if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil || len(data.Levels) != 20 {
				t.Fatalf("invalid response: %v", err)
			}
		}
	}
}

func TestClassProgressionPreviewRejectsInvalidOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NEOTOKYO_DATA_DIR", dir)
	path := filepath.Join(dir, "classesv2", "Awakened Mutant.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"name":"Awakened Mutant","original_class":"Sorcerer","hit_die":6,"levels":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadClassProgressionPreview("Sorcerer"); err == nil {
		t.Fatal("invalid override was accepted")
	}
}

func TestLevelUpIsPreviewOnly(t *testing.T) {
	raw, err := os.ReadFile("templates/characters.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `id="level-up-apply" class="primary" disabled`) {
		t.Fatal("incomplete level-up is not disabled")
	}
}
