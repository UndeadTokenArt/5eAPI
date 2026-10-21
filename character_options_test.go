package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCharacterOptionsSeparateClassesFromSubclasses(t *testing.T) {
	guide, err := loadGuideFromDir(builtinGuideDir)
	if err != nil {
		t.Fatal(err)
	}
	opts := buildCharacterOptions(guide)

	classes := map[string]bool{}
	for _, class := range opts.Classes {
		if strings.Contains(strings.ToLower(class.Name), "subclass") {
			t.Errorf("subclass entry %q listed as a class", class.Name)
		}
		classes[class.Name] = true
	}
	for _, sub := range opts.Subclasses {
		if sub.Link == "" || len(sub.Notes) == 0 {
			t.Errorf("subclass %q has link %q and %d notes", sub.Name, sub.Link, len(sub.Notes))
		}
	}
	if len(opts.Classes) != 10 {
		t.Errorf("classes = %d, want 10: %v", len(opts.Classes), opts.Classes)
	}
	if len(opts.Subclasses) == 0 {
		t.Fatal("no subclasses were extracted")
	}
	wantSubclass := map[string]string{
		"Street Samurai":                    "Street Samurai / Corporate Enforcer",
		"College of the Pirate Broadcaster": "Neon Idol / Net-Hype",
		"Street Tracker":                    "Urban Hunter / Recon Drone Master",
	}
	found := map[string]string{}
	for _, sub := range opts.Subclasses {
		if !classes[sub.Class] {
			t.Errorf("subclass %q points at unknown class %q", sub.Name, sub.Class)
		}
		found[sub.Name] = sub.Class
	}
	for name, class := range wantSubclass {
		if found[name] != class {
			t.Errorf("subclass %q class = %q, want %q", name, found[name], class)
		}
	}
}

func TestWikiNestsSubclassesUnderTheirClass(t *testing.T) {
	guide, err := loadGuideFromDir(builtinGuideDir)
	if err != nil {
		t.Fatal(err)
	}
	node, base := categoryNode(guide, "Classes")
	if base == "" {
		t.Fatal("no Classes section")
	}
	if len(node.Children) != len(guide.Category.Classes) {
		t.Fatalf("class nodes = %d, want %d", len(node.Children), len(guide.Category.Classes))
	}
	for i, class := range guide.Category.Classes {
		subs := subclassNodes(node.Children[i])
		if len(subs) != len(class.Subclasses) {
			t.Fatalf("%s: subclass nodes = %d, want %d", class.Name, len(subs), len(class.Subclasses))
		}
		for j, sub := range subs {
			if sub.Title != class.Subclasses[j].Name || !strings.HasPrefix(sub.ID, node.Children[i].ID+"-") {
				t.Errorf("%s: subclass node %q (id %s) is not nested under the class", class.Name, sub.Title, sub.ID)
			}
		}
	}
}

func TestCharacterOptionsMatchGuideAndLinkToDetails(t *testing.T) {
	guide, err := loadGuideFromDir(builtinGuideDir)
	if err != nil {
		t.Fatal(err)
	}
	opts := buildCharacterOptions(guide)
	c := guide.Category

	wantEquipment := len(c.AdventuringGearAndContainers) + len(c.ToolsKitsAndConsumables) + len(c.Ammunition) +
		len(c.ArmorAndShields) + len(c.MountsVehiclesAndHarnesses) + len(c.FociAndInstruments)
	wantFeats := 0
	for _, group := range c.Feats {
		wantFeats += len(group.Features)
	}
	for name, got := range map[string][2]int{
		"equipment": {len(opts.Equipment), wantEquipment},
		"weapons":   {len(opts.Weapons), len(c.Weapons)},
		"spells":    {len(opts.Spells), len(c.Spells)},
		"feats":     {len(opts.Feats), wantFeats},
	} {
		if got[0] != got[1] {
			t.Errorf("%s options = %d, want %d", name, got[0], got[1])
		}
	}

	pages := map[string]string{}
	all := append(append(append(append([]CharacterOption{}, opts.Equipment...), opts.Weapons...), opts.Spells...), opts.Feats...)
	for _, option := range all {
		if option.Link == "" {
			t.Errorf("option %q has no link", option.Name)
			continue
		}
		path, id, _ := strings.Cut(option.Link, "#")
		body, ok := pages[path]
		if !ok {
			recorder := httptest.NewRecorder()
			setupRouter().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("%s status = %d", path, recorder.Code)
			}
			body = recorder.Body.String()
			pages[path] = body
		}
		if !strings.Contains(body, `id="`+id+`"`) {
			t.Errorf("option %q link %q does not resolve to a section", option.Name, option.Link)
		}
	}
}

func TestNeoTokyoSearchFindsAcrossCategories(t *testing.T) {
	for query, wantCategory := range map[string]string{
		"smart-arrow":    "Ammunition",
		"kevlar":         "Armor & Shields",
		"street samurai": "Classes",
		"koten":          "Backgrounds",
	} {
		recorder := httptest.NewRecorder()
		setupRouter().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/neotokyo/search?q="+url.QueryEscape(query), nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%q status = %d", query, recorder.Code)
		}
		var response NeoTokyoSearchResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, result := range response.Results {
			if result.Category == wantCategory && strings.HasPrefix(result.URL, "/neotokyo/") && strings.Contains(result.URL, "#") {
				found = true
			}
		}
		if !found {
			t.Errorf("search %q returned no %s result: %+v", query, wantCategory, response.Results)
		}
	}

	recorder := httptest.NewRecorder()
	setupRouter().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/neotokyo/search?q=a", nil))
	var short NeoTokyoSearchResponse
	_ = json.Unmarshal(recorder.Body.Bytes(), &short)
	if len(short.Results) != 0 {
		t.Errorf("one-character query returned %d results", len(short.Results))
	}
}

func TestOptionsCarryMechanicalNotes(t *testing.T) {
	guide, err := loadGuideFromDir(builtinGuideDir)
	if err != nil {
		t.Fatal(err)
	}
	opts := buildCharacterOptions(guide)
	for name, list := range map[string][]CharacterOption{"species": opts.Species, "classes": opts.Classes, "backgrounds": opts.Backgrounds} {
		for _, o := range list {
			if len(o.Notes) == 0 {
				t.Errorf("%s %q has no notes", name, o.Name)
			}
			if o.Link == "" {
				t.Errorf("%s %q has no link", name, o.Name)
			}
		}
	}
	withNotes := 0
	for _, s := range opts.Subclasses {
		if len(s.Notes) > 0 {
			withNotes++
		}
	}
	if withNotes == 0 || withNotes < len(opts.Subclasses)/2 {
		t.Errorf("only %d of %d subclasses have notes", withNotes, len(opts.Subclasses))
	}
	for _, b := range opts.Backgrounds {
		if b.Notes[0].Label != "Skill proficiencies" {
			t.Errorf("background %q notes should lead with skills, got %q", b.Name, b.Notes[0].Label)
		}
	}
}

func TestEquipmentDetailIncludesDescription(t *testing.T) {
	got := equipmentDetail(BasicItem{Name: "Vest", Price: "50", Description: "Light armor. AC: 11 + Dex modifier."})
	if got != "Price: 50 \u2014 Light armor. AC: 11 + Dex modifier." {
		t.Fatalf("unexpected detail: %q", got)
	}
	if equipmentDetail(BasicItem{Name: "Rope"}) != "" {
		t.Fatal("empty item should have no detail")
	}
}
