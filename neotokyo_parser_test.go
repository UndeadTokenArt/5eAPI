package main

import (
	"html"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadJSONLoadsWrappedStarterGuide(t *testing.T) {
	guide, err := loadGuideFromDir(builtinGuideDir)
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
	guide, err := loadGuideFromDir(builtinGuideDir)
	if err != nil {
		t.Fatalf("load guide: %v", err)
	}
	index, err := loadSpellReskinIndex(referencePath("class-and-spell-reskins.json"))
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
		`class="spell-text spell-description"`,
		`<h3 class="spell-text-title">Description</h3>`,
		`class="spell-text spell-mechanics"`,
		`<h3 class="spell-text-title">Mechanics</h3>`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("spells page missing %q", expected)
		}
	}
}

func TestNeoTokyoEntryImagesRenderForTopLevelAndNestedItems(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NEOTOKYO_DATA_DIR", dir)
	writeFile(t, filepath.Join(dir, "classes", "classes.json"), `[
		{
			"name":"Agent",
			"image":"/images/agent.png",
			"features":[{
				"name":"Protocol",
				"image":"/images/protocol.png",
				"features":[{"name":"Subprotocol","image":"/images/subprotocol.png"}]
			}],
			"subclasses":[{
				"name":"Ghost",
				"image":"/images/ghost.png",
				"features":[{"name":"Phase","image":"/images/phase.png"}]
			}]
		},
		{"name":"No Icon","image":""}
	]`)
	writeFile(t, filepath.Join(dir, "spells", "spells.json"), `[{"name":"Signal Veil","image":"/images/signal-veil.png"}]`)
	writeFile(t, filepath.Join(dir, "_reference", "class-and-spell-reskins.json"), `{"spells_reskin":[],"class_reskins":[]}`)
	guideCache.Lock()
	guideCache.loaded = false
	guideCache.Unlock()

	classes := httptest.NewRecorder()
	setupRouter().ServeHTTP(classes, httptest.NewRequest(http.MethodGet, "/neotokyo/classes", nil))
	if classes.Code != http.StatusOK {
		t.Fatalf("classes status = %d: %s", classes.Code, classes.Body.String())
	}
	for _, image := range []string{"agent", "protocol", "subprotocol", "ghost", "phase"} {
		if !strings.Contains(classes.Body.String(), `src="/images/`+image+`.png"`) {
			t.Errorf("class page missing %s icon", image)
		}
	}
	if strings.Contains(classes.Body.String(), `src=""`) {
		t.Fatal("empty image URL should not render an icon")
	}

	spells := httptest.NewRecorder()
	setupRouter().ServeHTTP(spells, httptest.NewRequest(http.MethodGet, "/neotokyo/spells", nil))
	if spells.Code != http.StatusOK || !strings.Contains(spells.Body.String(), `src="/images/signal-veil.png"`) {
		t.Fatalf("spell image did not render: status %d: %s", spells.Code, spells.Body.String())
	}
}

func TestCategoryMappersRetainImages(t *testing.T) {
	check := func(label string, node SectionNode, want string) {
		t.Helper()
		if node.Image != want {
			t.Errorf("%s image = %q, want %q", label, node.Image, want)
		}
	}

	abilities := mapAbilities([]Ability{{Name: "Ability", Image: "ability.png", Features: []AbilityFeature{{Name: "Feature", Image: "feature.png"}}}})
	check("ability", abilities.Children[0], "ability.png")
	check("ability feature", abilities.Children[0].Children[0], "feature.png")

	backgrounds := mapBackgrounds([]Background{{Name: "Background", Image: "background.png"}})
	check("background", backgrounds.Children[0], "background.png")

	species := mapSpecies([]Species{{Name: "Species", Image: "species.png", Trait: []Trait{{Name: "Trait", Image: "trait.png"}}}})
	check("species", species.Children[0], "species.png")
	check("species trait", species.Children[0].Children[0], "trait.png")

	classes := mapClasses([]Class{{
		Name: "Class", Image: "class.png",
		Features:   []ClassFeature{{Name: "Feature", Image: "feature.png", Features: []SubClassFeature{{Name: "Nested feature", Image: "nested.png"}}}},
		Subclasses: []Subclass{{Name: "Subclass", Image: "subclass.png"}},
	}})
	class := classes.Children[0]
	check("class", class, "class.png")
	check("class feature", class.Children[0], "feature.png")
	check("nested class feature", class.Children[0].Children[0], "nested.png")
	check("subclass", class.Children[1].Children[0], "subclass.png")

	feats := mapFeats([]Feat{{Name: "Feat", Image: "feat.png", Features: []FeatFeature{{Name: "Feat feature", Image: "feat-feature.png"}}}})
	check("feat", feats.Children[0], "feat.png")
	check("feat feature", feats.Children[0].Children[0], "feat-feature.png")

	items, _ := mapBasicItems("Items", []BasicItem{{Name: "Item", Image: "item.png"}})
	check("item", items.Children[0], "item.png")
	weapons := mapWeapons([]Weapon{{Name: "Weapon", Image: "weapon.png"}})
	check("weapon", weapons.Children[0], "weapon.png")
	spells := mapSpells([]Spell{{Name: "Spell", Image: "spell.png"}})
	check("spell", spells.Children[0], "spell.png")
	conditions := mapConditions([]Condition{{Name: "Condition", Image: "condition.png"}})
	check("condition", conditions.Children[0], "condition.png")
}
