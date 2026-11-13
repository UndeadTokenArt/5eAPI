package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type cyberMageChoice struct {
	ID              string          `json:"id"`
	Kind            string          `json:"kind"`
	Count           *int            `json:"count"`
	AllowedOptions  json.RawMessage `json:"allowed_options"`
	EligibilityRule string          `json:"eligibility_rule"`
	SelectionEffect string          `json:"selection_effect"`
}

type cyberMageFeature struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	When        string            `json:"when"`
	Choices     []cyberMageChoice `json:"choices"`
}

type cyberMageSubclass struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	SelectedAtClassLevel int    `json:"selected_at_class_level"`
	Resources            []struct {
		ID          string `json:"id"`
		Recharge    string `json:"recharge"`
		MaximumRule string `json:"maximum_rule"`
	} `json:"resources"`
	Levels []struct {
		Level    int                `json:"level"`
		Status   string             `json:"status"`
		Features []cyberMageFeature `json:"features"`
	} `json:"levels"`
}

func TestCyberMageSubclassWorksheet(t *testing.T) {
	raw, err := os.ReadFile("static/data/neotokyo/classesv2/Cyber-Mage.json")
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		SchemaStatus string `json:"schema_status"`
		Levels       []struct {
			Level            int                `json:"level"`
			Features         []cyberMageFeature `json:"features"`
			SpellProgression struct {
				MaxClassSpellLevel int `json:"max_class_spell_level"`
			} `json:"spell_progression"`
		} `json:"levels"`
		Subclasses []cyberMageSubclass `json:"subclasses"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	if data.SchemaStatus != "draft_authoring_template_not_supported_by_app" || len(data.Levels) != 20 {
		t.Fatal("worksheet must remain a 20-level unsupported draft")
	}

	expected := map[string]map[int][]string{
		"neon-bladesinger": {
			3: {"bladesong", "training-in-war-and-song"}, 6: {"extra-attack"},
			10: {"song-of-defense"}, 14: {"song-of-victory"},
		},
		"system-forensics-diviner": {
			3: {"divination-savant", "portent"}, 6: {"expert-divination"},
			10: {"third-eye"}, 14: {"greater-portent"},
		},
		"ghost-protocol-necromancer": {
			3: {"necromancy-savant", "necromancy-spellbook"}, 6: {"grave-power", "undead-thralls"},
			10: {"harvest-undead"}, 14: {"deaths-master"},
		},
	}
	if len(data.Subclasses) != len(expected) {
		t.Fatalf("got %d subclasses, want %d", len(data.Subclasses), len(expected))
	}
	var names []string
	seenSubclasses := map[string]bool{}
	for _, sub := range data.Subclasses {
		milestones, ok := expected[sub.ID]
		if !ok || seenSubclasses[sub.ID] {
			t.Fatalf("unexpected or duplicate subclass %q", sub.ID)
		}
		seenSubclasses[sub.ID] = true
		names = append(names, sub.Name)
		if sub.Name == "" || sub.Description == "" || sub.SelectedAtClassLevel != 3 || len(sub.Resources) == 0 {
			t.Errorf("%s: incomplete subclass metadata", sub.ID)
		}
		resourceIDs := map[string]bool{}
		for _, resource := range sub.Resources {
			if resource.ID == "" || resourceIDs[resource.ID] || resource.Recharge == "" || resource.MaximumRule == "" {
				t.Errorf("%s: incomplete or duplicate resource %+v", sub.ID, resource)
			}
			resourceIDs[resource.ID] = true
		}
		features := map[int]map[string]cyberMageFeature{}
		featureIDs, choiceIDs := map[string]bool{}, map[string]bool{}
		lastLevel := 0
		for _, level := range sub.Levels {
			if level.Level <= lastLevel || level.Level > 20 || level.Status != "unfilled" || len(level.Features) == 0 {
				t.Fatalf("%s: invalid level %d", sub.ID, level.Level)
			}
			lastLevel = level.Level
			features[level.Level] = map[string]cyberMageFeature{}
			for _, feature := range level.Features {
				if feature.ID == "" || featureIDs[feature.ID] || feature.Name == "" || feature.Description == "" || feature.When != "both" || feature.Choices == nil {
					t.Errorf("%s level %d: incomplete or duplicate feature %q", sub.ID, level.Level, feature.ID)
				}
				featureIDs[feature.ID] = true
				features[level.Level][feature.ID] = feature
				for _, choice := range feature.Choices {
					if choice.ID == "" || choiceIDs[choice.ID] || choice.Kind == "" || len(choice.AllowedOptions) == 0 || string(choice.AllowedOptions) == "null" || choice.EligibilityRule == "" || choice.SelectionEffect == "" {
						t.Errorf("%s: incomplete or duplicate choice %q", sub.ID, choice.ID)
					}
					choiceIDs[choice.ID] = true
				}
			}
		}
		for level, ids := range milestones {
			for _, id := range ids {
				if _, ok := features[level][id]; !ok {
					t.Errorf("%s: missing %s at level %d", sub.ID, id, level)
				}
			}
		}
		if sub.ID == "neon-bladesinger" {
			if len(sub.Levels) != 4 {
				t.Errorf("%s: unexpected levels", sub.ID)
			}
			continue
		}
		school := "divination"
		if sub.ID == "ghost-protocol-necromancer" {
			school = "necromancy"
		}
		initial := features[3][school+"-savant"]
		if len(initial.Choices) != 1 || initial.Choices[0].Count == nil || *initial.Choices[0].Count != 2 {
			t.Errorf("%s: initial Savant must offer two spells", sub.ID)
		}
		unlocks := []int{5, 7, 9, 11, 13, 15, 17}
		if len(sub.Levels) != 4+len(unlocks) {
			t.Errorf("%s: unexpected subclass levels", sub.ID)
		}
		for i, level := range unlocks {
			slotLevel := i + 3
			if data.Levels[level-1].SpellProgression.MaxClassSpellLevel != slotLevel || data.Levels[level-2].SpellProgression.MaxClassSpellLevel != slotLevel-1 {
				t.Fatalf("unexpected Cyber-Mage spell slot unlock at level %d", level)
			}
			id := school + "-savant-slot-level-" + strconv.Itoa(slotLevel)
			feature, ok := features[level][id]
			if !ok || len(feature.Choices) != 1 || feature.Choices[0].Count == nil || *feature.Choices[0].Count != 1 || feature.Choices[0].Kind != "spell" || !strings.Contains(strings.ToLower(string(feature.Choices[0].AllowedOptions)), school) {
				t.Errorf("%s: missing recurring Savant spell choice at level %d", sub.ID, level)
			}
		}
	}
	var options []string
	for _, feature := range data.Levels[2].Features {
		if feature.ID == "wizard-subclass-3" && len(feature.Choices) == 1 {
			if err := json.Unmarshal(feature.Choices[0].AllowedOptions, &options); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !reflect.DeepEqual(options, names) {
		t.Errorf("subclass selection options %v do not match authored names %v", options, names)
	}
}

func TestCyberMageSubclassRules(t *testing.T) {
	raw, err := os.ReadFile("static/data/neotokyo/classesv2/Cyber-Mage.json")
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Subclasses []cyberMageSubclass `json:"subclasses"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	features := map[string]cyberMageFeature{}
	resources := map[string]string{}
	for _, sub := range data.Subclasses {
		for _, level := range sub.Levels {
			for _, feature := range level.Features {
				features[sub.ID+"/"+feature.ID] = feature
			}
		}
		for _, resource := range sub.Resources {
			resources[sub.ID+"/"+resource.ID] = resource.Recharge + " " + resource.MaximumRule
		}
	}
	checks := map[string][]string{
		"neon-bladesinger/bladesong":                      {"bonus action", "no armor", "no shield", "1 minute", "incapacitated", "two hands", "minimum +1", "10 feet", "concentration", "minimum 1", "long rest"},
		"neon-bladesinger/training-in-war-and-song":       {"martial melee", "Heavy", "Two-Handed", "spellcasting focus"},
		"neon-bladesinger/extra-attack":                   {"Attack action", "twice", "cantrip", "casting time of an action"},
		"neon-bladesinger/song-of-defense":                {"reaction", "spell slot", "five times"},
		"system-forensics-diviner/portent":                 {"two d20s", "Before", "see", "once per turn", "level 14", "three"},
		"system-forensics-diviner/expert-divination":      {"level 2 or higher", "lower", "no higher than level 5"},
		"system-forensics-diviner/third-eye":              {"bonus action", "120 feet", "read any language", "See Invisibility", "without a spell slot", "short or long rest"},
		"ghost-protocol-necromancer/necromancy-spellbook": {"resistance to necrotic", "Find Familiar", "Skeleton", "Zombie", "reaction", "60 feet", "slot level plus your Cyber-Mage level"},
		"ghost-protocol-necromancer/grave-power":          {"holding your spellbook", "exhaustion", "ignores resistance"},
		"ghost-protocol-necromancer/undead-thralls":       {"always prepared", "once without a spell slot", "effective level by 1", "holding your spellbook", "half your Cyber-Mage level", "60 feet", "minimum 1"},
		"ghost-protocol-necromancer/harvest-undead":       {"bloodied", "not at 0 hit points", "reaction", "control", "Cyber-Mage level"},
		"ghost-protocol-necromancer/deaths-master":        {"holding your spellbook", "bonus action", "60 feet", "temporary hit points", "long rest", "unexpended Hit Dice", "minimum 1d6", "10-foot", "Dexterity saving throw", "half damage", "do not control", "reaction", "level 5 or higher"},
	}
	for id, fragments := range checks {
		feature, ok := features[id]
		if !ok {
			t.Errorf("missing feature %s", id)
			continue
		}
		for _, fragment := range fragments {
			if !strings.Contains(feature.Description, fragment) {
				t.Errorf("%s: description missing %q", id, fragment)
			}
		}
	}
	resourceChecks := map[string][]string{
		"neon-bladesinger/bladesong-uses":                   {"long", "Intelligence modifier", "minimum 1"},
		"system-forensics-diviner/portent-rolls":            {"long", "Two", "three", "level 14"},
		"system-forensics-diviner/third-eye-use":            {"short", "One", "level 10", "short or long rest"},
		"ghost-protocol-necromancer/animate-dead-free-cast": {"long", "One", "level 6"},
		"ghost-protocol-necromancer/bolster-undead-use":     {"long", "One", "level 14"},
	}
	for id, fragments := range resourceChecks {
		for _, fragment := range fragments {
			if !strings.Contains(resources[id], fragment) {
				t.Errorf("%s: resource missing %q", id, fragment)
			}
		}
	}
}
