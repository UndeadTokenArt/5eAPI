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
		id    string
	}{
		{path: "/neotokyo/classes", title: "Classes", id: "classes-1"},
		{path: "/neotokyo/adventuring%20gear%20and%20containers", title: "Adventuring Gear & Containers", id: "adventuring-gear-containers-1"},
		{path: "/neotokyo/tools-kits-and-consumables", title: "Tools, Kits & Consumables", id: "tools-kits-consumables-1"},
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
			wantTitle := "<h1>" + html.EscapeString(test.title)
			if !strings.Contains(recorder.Body.String(), wantTitle) {
				t.Fatalf("rendered page does not contain category title %q", test.title)
			}
			if strings.HasPrefix(strings.TrimSpace(recorder.Body.String()), "{") {
				t.Fatal("response is raw JSON, want rendered HTML")
			}
			if !strings.Contains(recorder.Body.String(), `aria-label="Breadcrumb"`) ||
				!strings.Contains(recorder.Body.String(), `aria-label="Table of contents"`) ||
				!strings.Contains(recorder.Body.String(), `href="#`+test.id+`-`) ||
				!strings.Contains(recorder.Body.String(), `id="`+test.id+`-`) {
				t.Fatalf("rendered page is missing breadcrumb or section index for %q", test.title)
			}
		})
	}
}

func TestNeoTokyoRootRendersCategoryMenu(t *testing.T) {
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
	if !strings.Contains(body, "<h1>NeoTokyo</h1>") || !strings.Contains(body, `aria-label="NeoTokyo categories"`) {
		t.Fatal("root page does not render the category menu")
	}
	for _, href := range []string{
		`href="/neotokyo/abilities"`,
		`href="/neotokyo/classes"`,
		`href="/neotokyo/spells"`,
		`href="/neotokyo/adventuring-gear-and-containers"`,
	} {
		if !strings.Contains(body, href) {
			t.Errorf("root category menu missing %s", href)
		}
	}
	if strings.Contains(body, `id="spell-class-filter"`) || strings.Contains(body, `id="abilities-1"`) {
		t.Fatal("root route should show navigation rather than guide content")
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

func TestSpellReskinMetadataAddsClassAndLevel(t *testing.T) {
	guide, err := loadJSON("static/data/neo_tokyo_starter_guide.json")
	if err != nil {
		t.Fatalf("load guide: %v", err)
	}
	index, err := loadSpellReskinIndex("static/data/cyberpunk-class-and-spell-reskins.json")
	if err != nil {
		t.Fatalf("load spell index: %v", err)
	}
	response := buildResponse(guide, "spells")
	applySpellReskinMetadata(response.Sections, index)
	if len(response.Sections) != 1 {
		t.Fatalf("spells response sections = %d, want 1", len(response.Sections))
	}

	var found bool
	for _, spell := range response.Sections[0].Children {
		if spell.Title != "Corrosive Flechette Pulse" {
			continue
		}
		found = true
		if spell.SpellLevel != 2 {
			t.Errorf("spell level = %d, want 2", spell.SpellLevel)
		}
		if len(spell.SpellClasses) != 2 || spell.SpellClasses[0] != "Wizard" || spell.SpellClasses[1] != "Artificer" {
			t.Errorf("spell classes = %#v", spell.SpellClasses)
		}
	}
	if !found {
		t.Fatal("test spell was not found in the guide")
	}
}

func TestNeoTokyoSpellsPageRendersClassAndLevelFilters(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/neotokyo/spells", nil)
	setupRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, expected := range []string{
		`id="spell-class-filter"`,
		`id="spell-level-filter"`,
		`<option value="Artificer">Cyber-Doctor / Tech-Smith</option>`,
		`<option value="Wizard">Corporate Decker / Cyber-Mage</option>`,
		`value="0">Cantrip`,
		`data-spell-classes="Wizard,Artificer" data-spell-level="2"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("spells page missing %q", expected)
		}
	}
}
