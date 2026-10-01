package main

import (
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoadJSONLoadsWrappedStarterGuide(t *testing.T) {
	guide, err := loadJSON("static/data/neo_tokyo_starter_guide.json")
	if err != nil {
		t.Fatalf("loadJSON returned error: %v", err)
	}
	if len(guide.Category.Abilities) == 0 || len(guide.Category.Classes) == 0 {
		t.Fatalf("wrapped guide categories were not decoded: abilities=%d classes=%d", len(guide.Category.Abilities), len(guide.Category.Classes))
	}
	if len(guide.Category.AdventuringGearAndContainers) == 0 {
		t.Fatal("category key containing spaces was not decoded")
	}
}

func TestNeoTokyoCategoryRoutes(t *testing.T) {
	tests := []struct {
		path  string
		title string
	}{
		{path: "/neotokyo/classes", title: "Classes"},
		{path: "/neotokyo/adventuring%20gear%20and%20containers", title: "Adventuring Gear & Containers"},
		{path: "/neotokyo/tools-kits-and-consumables", title: "Tools, Kits & Consumables"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			setupRouter().ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
			}
			if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
				t.Fatalf("content-type = %q, want HTML", got)
			}
			wantTitle := "<h1>Neo Tokyo Starter Guide - " + html.EscapeString(test.title)
			if !strings.Contains(recorder.Body.String(), wantTitle) {
				t.Fatalf("rendered page does not contain category title %q", test.title)
			}
			if strings.HasPrefix(strings.TrimSpace(recorder.Body.String()), "{") {
				t.Fatal("response is raw JSON, want rendered HTML")
			}
		})
	}
}

func TestNeoTokyoRootReturnsAllSections(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/neotokyo", nil)
	setupRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want HTML", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "<h1>Neo Tokyo Starter Guide</h1>") || !strings.Contains(body, "Abilities") || !strings.Contains(body, "Conditions") {
		t.Fatal("root page does not render the full guide sections")
	}
}

func TestNeoTokyoUnknownCategoryReturnsNotFound(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/neotokyo/not-a-category", nil)
	setupRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
