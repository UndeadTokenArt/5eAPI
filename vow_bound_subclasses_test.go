package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

type vowFeature struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	When                 string          `json:"when"`
	Choices              json.RawMessage `json:"choices"`
	AlwaysPreparedSpells []string        `json:"always_prepared_spells"`
}

type vowLevel struct {
	Level            int          `json:"level"`
	Status           string       `json:"status"`
	Features         []vowFeature `json:"features"`
	SpellProgression struct {
		MaxClassSpellLevel int `json:"max_class_spell_level"`
	} `json:"spell_progression"`
}

type vowSubclass struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	SelectedAtClassLevel int       `json:"selected_at_class_level"`
	SpellcastingOverride *struct{} `json:"spellcasting_override"`
	Resources            []struct {
		ID          string `json:"id"`
		Recharge    string `json:"recharge"`
		MaximumRule string `json:"maximum_rule"`
	} `json:"resources"`
	Levels []vowLevel `json:"levels"`
}

func readVowWorksheet(t *testing.T) (string, []vowLevel, []vowSubclass) {
	t.Helper()
	raw, err := os.ReadFile("static/data/neotokyo/classesv2/Vow-Bound Operative.json")
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		SchemaStatus string        `json:"schema_status"`
		Levels       []vowLevel    `json:"levels"`
		Subclasses   []vowSubclass `json:"subclasses"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	return data.SchemaStatus, data.Levels, data.Subclasses
}

func TestVowBoundSubclassWorksheet(t *testing.T) {
	status, baseLevels, subclasses := readVowWorksheet(t)
	if status != "draft_authoring_template_not_supported_by_app" || len(baseLevels) != 20 {
		t.Fatal("worksheet must remain a 20-level unsupported draft")
	}
	type wantOath struct {
		name       string
		spellID    string
		spells     map[int][]string
		features   map[int][]string
		resourceID string
		recharge   string
	}
	expected := map[string]wantOath{
		"aegis-enforcer": {
			name: "Oath of the Aegis Enforcer", spellID: "devotion",
			spells: map[int][]string{
				3: {"Protection from Evil and Good", "Shield of Faith"}, 5: {"Aid", "Zone of Truth"},
				9: {"Beacon of Hope", "Dispel Magic"}, 13: {"Freedom of Movement", "Guardian of Faith"},
				17: {"Commune", "Flame Strike"},
			},
			features:   map[int][]string{3: {"sacred-weapon"}, 7: {"aura-of-devotion"}, 15: {"smite-of-protection"}, 20: {"holy-nimbus"}},
			resourceID: "holy-nimbus-use", recharge: "other",
		},
		"chrome-samurai": {
			name: "Oath of the Chrome Samurai", spellID: "vengeance",
			spells: map[int][]string{
				3: {"Bane", "Hunter's Mark"}, 5: {"Hold Person", "Misty Step"},
				9: {"Haste", "Protection from Energy"}, 13: {"Banishment", "Dimension Door"},
				17: {"Hold Monster", "Scrying"},
			},
			features:   map[int][]string{3: {"vow-of-enmity"}, 7: {"relentless-avenger"}, 15: {"soul-of-vengeance"}, 20: {"avenging-angel"}},
			resourceID: "avenging-angel-use", recharge: "other",
		},
		"civic-sentinel": {
			name: "Oath of the Civic Sentinel", spellID: "crown",
			spells: map[int][]string{
				3: {"Command", "Compelled Duel"}, 5: {"Warding Bond", "Zone of Truth"},
				9: {"Aura of Vitality", "Spirit Guardians"}, 13: {"Banishment", "Guardian of Faith"},
				17: {"Circle of Power", "Geas"},
			},
			features:   map[int][]string{3: {"champion-challenge", "turn-the-tide"}, 7: {"divine-allegiance"}, 15: {"unyielding-saint"}, 20: {"exalted-champion"}},
			resourceID: "exalted-champion-use", recharge: "long",
		},
	}
	if len(subclasses) != len(expected) {
		t.Fatalf("got %d subclasses, want %d", len(subclasses), len(expected))
	}
	names, seen := map[string]bool{}, map[string]bool{}
	for _, sub := range subclasses {
		want, ok := expected[sub.ID]
		if !ok || seen[sub.ID] {
			t.Fatalf("unexpected or duplicate subclass %q", sub.ID)
		}
		seen[sub.ID], names[sub.Name] = true, true
		if sub.Name != want.name || sub.Description == "" || sub.SelectedAtClassLevel != 3 || sub.SpellcastingOverride != nil {
			t.Errorf("%s: invalid subclass metadata", sub.ID)
		}
		if len(sub.Resources) != 1 || sub.Resources[0].ID != want.resourceID || sub.Resources[0].Recharge != want.recharge || !strings.Contains(sub.Resources[0].MaximumRule, "level 20") {
			t.Errorf("%s: invalid capstone resource", sub.ID)
		}
		byLevel := map[int]map[string]vowFeature{}
		featureIDs := map[string]bool{}
		last := 0
		for _, level := range sub.Levels {
			if level.Level <= last || level.Level > 20 || level.Status != "unfilled" || len(level.Features) == 0 {
				t.Fatalf("%s: invalid level %d", sub.ID, level.Level)
			}
			last = level.Level
			byLevel[level.Level] = map[string]vowFeature{}
			for _, feature := range level.Features {
				if feature.ID == "" || featureIDs[feature.ID] || feature.Name == "" || feature.Description == "" || feature.When != "both" || string(feature.Choices) != "[]" {
					t.Errorf("%s level %d: invalid feature %q", sub.ID, level.Level, feature.ID)
				}
				featureIDs[feature.ID] = true
				byLevel[level.Level][feature.ID] = feature
			}
		}
		if len(sub.Levels) != 8 {
			t.Errorf("%s: got %d subclass levels, want 8", sub.ID, len(sub.Levels))
		}
		for level, ids := range want.features {
			for _, id := range ids {
				if _, ok := byLevel[level][id]; !ok {
					t.Errorf("%s: missing %s at level %d", sub.ID, id, level)
				}
			}
		}
		for level, spells := range want.spells {
			id := fmt.Sprintf("%s-oath-spells-%d", want.spellID, level)
			feature, ok := byLevel[level][id]
			if !ok || !reflect.DeepEqual(feature.AlwaysPreparedSpells, spells) {
				t.Errorf("%s level %d: got spells %v, want %v", sub.ID, level, feature.AlwaysPreparedSpells, spells)
			}
			for _, fragment := range []string{"always prepared", "prepared-spell limit", "Charisma"} {
				if !strings.Contains(feature.Description, fragment) {
					t.Errorf("%s level %d: missing %q", sub.ID, level, fragment)
				}
			}
			if level != 3 && baseLevels[level-1].SpellProgression.MaxClassSpellLevel != (level-1)/4+1 {
				t.Errorf("%s level %d: unexpected class spell level", sub.ID, level)
			}
		}
	}
	var options []string
	for _, feature := range baseLevels[2].Features {
		if feature.ID == "paladin-subclass-3" {
			var choices []struct {
				Count          int      `json:"count"`
				AllowedOptions []string `json:"allowed_options"`
			}
			if err := json.Unmarshal(feature.Choices, &choices); err != nil || len(choices) != 1 || choices[0].Count != 1 {
				t.Fatalf("invalid subclass choice: %v", err)
			}
			options = choices[0].AllowedOptions
		}
	}
	if len(options) != len(names) {
		t.Fatalf("subclass options %v do not match authored names", options)
	}
	for _, option := range options {
		if !names[option] {
			t.Errorf("unexpected or duplicate option %q", option)
		}
		delete(names, option)
	}
}

func TestVowBoundSubclassMechanics(t *testing.T) {
	_, _, subclasses := readVowWorksheet(t)
	features := map[string]string{}
	resources := map[string]string{}
	for _, sub := range subclasses {
		for _, level := range sub.Levels {
			for _, feature := range level.Features {
				features[sub.ID+"/"+feature.ID] = feature.Description
			}
		}
		for _, resource := range sub.Resources {
			resources[sub.ID+"/"+resource.ID] = resource.MaximumRule
		}
	}
	checks := map[string][]string{
		"aegis-enforcer/sacred-weapon":       {"Attack action", "Channel Divinity", "melee weapon", "10 minutes", "Charisma modifier", "minimum +1", "radiant", "bright light", "dim light", "stop carrying"},
		"aegis-enforcer/aura-of-devotion":    {"allies", "immunity", "charmed", "Aura of Protection"},
		"aegis-enforcer/smite-of-protection": {"cast Divine Smite", "half cover", "start of your next turn"},
		"aegis-enforcer/holy-nimbus":         {"bonus action", "10 minutes", "Fiend", "Undead", "Charisma modifier plus proficiency bonus", "sunlight", "long rest", "level 5 spell slot"},
		"chrome-samurai/vow-of-enmity":       {"Attack action", "Channel Divinity", "see", "30 feet", "advantage", "1 minute", "0 hit points", "transfer", "without an action"},
		"chrome-samurai/relentless-avenger":  {"opportunity attack", "speed to 0", "end of the current turn", "half your speed", "same reaction", "without provoking"},
		"chrome-samurai/soul-of-vengeance":   {"hits or misses", "reaction", "melee attack", "within range"},
		"chrome-samurai/avenging-angel":      {"bonus action", "60 feet", "hover", "10 minutes", "Wisdom saving throw", "frightened", "1 minute", "any damage", "advantage", "level 5 spell slot"},
		"civic-sentinel/champion-challenge":  {"bonus action", "Channel Divinity", "choice", "see", "30 feet", "Wisdom saving throw", "spell save DC", "willingly move", "incapacitated", "die"},
		"civic-sentinel/turn-the-tide":       {"bonus action", "Channel Divinity", "choice", "hear", "30 feet", "no more than half", "1d6", "Charisma modifier", "minimum 1"},
		"civic-sentinel/divine-allegiance":   {"5 feet", "reaction", "instead", "cannot be reduced or prevented"},
		"civic-sentinel/unyielding-saint":    {"advantage", "paralyzed", "stunned"},
		"civic-sentinel/exalted-champion":    {"As an action", "1 hour", "incapacitated", "die", "bludgeoning", "piercing", "slashing", "nonmagical weapons", "30 feet", "death saving throws", "Wisdom saving throws", "long rest"},
	}
	for id, fragments := range checks {
		for _, fragment := range fragments {
			if !strings.Contains(features[id], fragment) {
				t.Errorf("%s: missing %q", id, fragment)
			}
		}
	}
	for _, id := range []string{"aegis-enforcer/holy-nimbus-use", "chrome-samurai/avenging-angel-use"} {
		if !strings.Contains(resources[id], "level 5 spell slot") || !strings.Contains(resources[id], "long rest") {
			t.Errorf("%s: missing 2024 capstone recovery", id)
		}
	}
	if !strings.Contains(resources["civic-sentinel/exalted-champion-use"], "No spell-slot restoration") {
		t.Error("Crown capstone must retain its long-rest-only recharge")
	}
}
