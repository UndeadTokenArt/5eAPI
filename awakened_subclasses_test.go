package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

type mutantChoice struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Kind            string          `json:"kind"`
	Count           *int            `json:"count"`
	AllowedOptions  json.RawMessage `json:"allowed_options"`
	EligibilityRule string          `json:"eligibility_rule"`
	SelectionEffect string          `json:"selection_effect"`
}

type mutantFeature struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Description          string         `json:"description"`
	When                 string         `json:"when"`
	Choices              []mutantChoice `json:"choices"`
	AlwaysPreparedSpells []string       `json:"always_prepared_spells"`
	SurgeTable           []struct {
		Roll   string `json:"roll"`
		Effect string `json:"effect"`
	} `json:"surge_table"`
}

type mutantLevel struct {
	Level            int             `json:"level"`
	Status           string          `json:"status"`
	Features         []mutantFeature `json:"features"`
	SpellProgression struct {
		MaxClassSpellLevel int `json:"max_class_spell_level"`
	} `json:"spell_progression"`
}

type mutantSubclass struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	SelectedAtClassLevel int             `json:"selected_at_class_level"`
	SpellcastingOverride json.RawMessage `json:"spellcasting_override"`
	Resources            []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Recharge    string `json:"recharge"`
		MaximumRule string `json:"maximum_rule"`
	} `json:"resources"`
	Levels []mutantLevel `json:"levels"`
}

func readMutantWorksheet(t *testing.T) (string, []mutantLevel, []mutantSubclass) {
	t.Helper()
	raw, err := os.ReadFile("static/data/neotokyo/classesv2/Awakened Mutant.json")
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		SchemaStatus string           `json:"schema_status"`
		Levels       []mutantLevel    `json:"levels"`
		Subclasses   []mutantSubclass `json:"subclasses"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	return data.SchemaStatus, data.Levels, data.Subclasses
}

func TestAwakenedSubclassWorksheet(t *testing.T) {
	status, base, subclasses := readMutantWorksheet(t)
	if status != "draft_authoring_template_not_supported_by_app" || len(base) != 20 {
		t.Fatal("worksheet must remain a 20-level unsupported draft")
	}
	type expectation struct {
		name     string
		spells   map[int][]string
		features map[int][]string
	}
	expected := map[string]expectation{
		"sector-17-bio-mutant": {
			name: "Sector 17 Bio-Mutant",
			spells: map[int][]string{
				3: {"Arms of Hadar", "Calm Emotions", "Detect Thoughts", "Dissonant Whispers", "Mind Sliver"},
				5: {"Hunger of Hadar", "Sending"}, 7: {"Evard's Black Tentacles", "Summon Aberration"}, 9: {"Rary's Telepathic Bond", "Telekinesis"},
			},
			features: map[int][]string{3: {"psionic-spells-3", "telepathic-speech"}, 6: {"psionic-sorcery", "psychic-defenses"}, 14: {"revelation-in-flesh"}, 18: {"warping-implosion"}},
		},
		"kami-synthetic-soul-vessel": {
			name: "KAMI Synthetic Soul Vessel",
			spells: map[int][]string{
				3: {"Aid", "Alarm", "Lesser Restoration", "Protection from Evil and Good"}, 5: {"Dispel Magic", "Protection from Energy"},
				7: {"Freedom of Movement", "Summon Construct"}, 9: {"Greater Restoration", "Wall of Force"},
			},
			features: map[int][]string{3: {"clockwork-spells-3", "restore-balance"}, 6: {"bastion-of-law"}, 14: {"trance-of-order"}, 18: {"clockwork-cavalcade"}},
		},
		"dragon-blood-replicant": {
			name: "Dragon-Blood Replicant",
			spells: map[int][]string{
				3: {"Alter Self", "Chromatic Orb", "Command", "Dragon's Breath"}, 5: {"Fear", "Fly"},
				7: {"Arcane Eye", "Charm Monster"}, 9: {"Legend Lore", "Summon Dragon"},
			},
			features: map[int][]string{3: {"draconic-resilience", "draconic-spells-3"}, 6: {"elemental-affinity"}, 14: {"dragon-wings"}, 18: {"dragon-companion"}},
		},
		"shadow-signal-mutant": {
			name: "Shadow Signal Mutant",
			spells: map[int][]string{
				3: {"Bane", "Darkness", "Inflict Wounds", "Pass Without Trace"}, 5: {"Hunger of Hadar", "Nondetection"},
				7: {"Greater Invisibility", "Phantasmal Killer"}, 9: {"Contagion", "Creation"},
			},
			features: map[int][]string{3: {"shadow-spells-3", "power-of-shadow"}, 6: {"beasts-of-ill-omen"}, 14: {"shadow-walk"}, 18: {"umbral-form"}},
		},
		"spellfire-reactor-mutant": {
			name: "Spellfire Reactor Mutant",
			spells: map[int][]string{
				3: {"Cure Wounds", "Guiding Bolt", "Lesser Restoration", "Scorching Ray"}, 5: {"Aura of Vitality", "Dispel Magic"},
				6: {"Counterspell"}, 7: {"Fire Shield", "Wall of Fire"}, 9: {"Greater Restoration", "Flame Strike"},
			},
			features: map[int][]string{3: {"spellfire-burst", "spellfire-spells-3"}, 6: {"absorb-spells"}, 14: {"honed-spellfire"}, 18: {"crown-of-spellfire"}},
		},
		"wild-signal-mutant": {
			name: "Wild Signal Mutant", spells: map[int][]string{},
			features: map[int][]string{3: {"wild-magic-surge", "tides-of-chaos"}, 6: {"bend-luck"}, 14: {"controlled-chaos"}, 18: {"tamed-surge"}},
		},
	}
	if len(subclasses) != len(expected) {
		t.Fatalf("got %d origins, want %d", len(subclasses), len(expected))
	}
	names, seen := map[string]bool{}, map[string]bool{}
	for _, sub := range subclasses {
		want, ok := expected[sub.ID]
		if !ok || seen[sub.ID] {
			t.Fatalf("unexpected or duplicate origin %q", sub.ID)
		}
		seen[sub.ID], names[sub.Name] = true, true
		if sub.Name != want.name || sub.Description == "" || sub.SelectedAtClassLevel != 3 || string(sub.SpellcastingOverride) != "null" {
			t.Errorf("%s: incomplete origin metadata", sub.ID)
		}
		resourceIDs, featureIDs, choiceIDs := map[string]bool{}, map[string]bool{}, map[string]bool{}
		for _, resource := range sub.Resources {
			if resource.ID == "" || resourceIDs[resource.ID] || resource.Name == "" || resource.Description == "" || resource.Recharge == "" || resource.MaximumRule == "" {
				t.Errorf("%s: invalid resource %q", sub.ID, resource.ID)
			}
			resourceIDs[resource.ID] = true
		}
		if len(sub.Resources) == 0 {
			t.Errorf("%s: expected limited resources", sub.ID)
		}
		byLevel := map[int]map[string]mutantFeature{}
		gotSpells := map[int][]string{}
		last := 0
		for _, level := range sub.Levels {
			if level.Level <= last || level.Level > 20 || level.Status != "unfilled" || len(level.Features) == 0 {
				t.Fatalf("%s: invalid level %d", sub.ID, level.Level)
			}
			last = level.Level
			byLevel[level.Level] = map[string]mutantFeature{}
			for _, feature := range level.Features {
				if feature.ID == "" || featureIDs[feature.ID] || feature.Name == "" || feature.Description == "" || feature.When != "both" || feature.Choices == nil {
					t.Errorf("%s level %d: invalid feature %q", sub.ID, level.Level, feature.ID)
				}
				featureIDs[feature.ID] = true
				byLevel[level.Level][feature.ID] = feature
				if len(feature.AlwaysPreparedSpells) > 0 {
					gotSpells[level.Level] = append(gotSpells[level.Level], feature.AlwaysPreparedSpells...)
					for _, term := range []string{"always prepared", "Charisma", "prepared-spell limit"} {
						if !strings.Contains(feature.Description, term) {
							t.Errorf("%s: %s missing %q", sub.ID, feature.ID, term)
						}
					}
				}
				for _, choice := range feature.Choices {
					if choice.ID == "" || choiceIDs[choice.ID] || choice.Name == "" || choice.Kind == "" || len(choice.AllowedOptions) == 0 || string(choice.AllowedOptions) == "null" || choice.EligibilityRule == "" || choice.SelectionEffect == "" {
						t.Errorf("%s: invalid choice %q", sub.ID, choice.ID)
					}
					choiceIDs[choice.ID] = true
				}
			}
		}
		for level, ids := range want.features {
			for _, id := range ids {
				if _, ok := byLevel[level][id]; !ok {
					t.Errorf("%s: missing %s at level %d", sub.ID, id, level)
				}
			}
		}
		if !reflect.DeepEqual(gotSpells, want.spells) {
			t.Errorf("%s: spells %v, want %v", sub.ID, gotSpells, want.spells)
		}
		wantLevels := 7
		if sub.ID == "wild-signal-mutant" {
			wantLevels = 4
		}
		if len(sub.Levels) != wantLevels {
			t.Errorf("%s: got %d subclass levels, want %d", sub.ID, len(sub.Levels), wantLevels)
		}
	}
	for _, level := range []int{3, 5, 7, 9} {
		if got, want := base[level-1].SpellProgression.MaxClassSpellLevel, (level+1)/2; got != want {
			t.Errorf("class level %d grants spell level %d, want %d", level, got, want)
		}
	}
	var options []string
	for _, feature := range base[2].Features {
		if feature.ID == "sorcerer-subclass-3" && len(feature.Choices) == 1 && feature.Choices[0].Count != nil && *feature.Choices[0].Count == 1 {
			if err := json.Unmarshal(feature.Choices[0].AllowedOptions, &options); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(options) != len(names) {
		t.Fatalf("origin options %v do not match authored names", options)
	}
	for _, option := range options {
		if !names[option] {
			t.Errorf("unexpected or duplicate option %q", option)
		}
		delete(names, option)
	}
}

func TestAwakenedSubclassMechanics(t *testing.T) {
	_, _, subclasses := readMutantWorksheet(t)
	features := map[string]mutantFeature{}
	resources := map[string]string{}
	for _, sub := range subclasses {
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
		"sector-17-bio-mutant/psionic-spells-3":          {"Mind Sliver", "cantrip", "separately granted"},
		"sector-17-bio-mutant/telepathic-speech":         {"bonus action", "see", "30 feet", "miles", "Charisma modifier", "minimum 1", "language", "minutes", "another creature"},
		"sector-17-bio-mutant/psionic-sorcery":           {"level 1 or higher", "equal to its spell level", "verbal", "somatic", "consumed", "specified cost"},
		"sector-17-bio-mutant/psychic-defenses":          {"resistance to psychic", "advantage", "charmed", "frightened"},
		"sector-17-bio-mutant/revelation-in-flesh":       {"bonus action", "10 minutes", "one benefit per point", "twice your speed", "hover", "60 feet", "total cover", "1 inch", "5 feet", "grappled"},
		"sector-17-bio-mutant/warping-implosion":         {"Magic action", "120 feet", "30 feet", "Strength save", "3d10 force", "half damage", "5 mutation points"},
		"kami-synthetic-soul-vessel/restore-balance":     {"see", "60 feet", "reaction", "advantage", "disadvantage", "Charisma modifier", "minimum once", "long rest"},
		"kami-synthetic-soul-vessel/bastion-of-law":      {"Magic action", "1 to 5", "see", "30 feet", "one d8 per point", "warded creature", "long rest", "again"},
		"kami-synthetic-soul-vessel/trance-of-order":     {"bonus action", "1-minute", "Attack rolls", "advantage", "9 or lower", "10", "5 mutation points"},
		"kami-synthetic-soul-vessel/clockwork-cavalcade": {"Magic action", "30-foot cube", "100 hit points", "entirely", "level 6 or lower", "7 mutation points"},
		"dragon-blood-replicant/draconic-resilience":     {"increases by 3", "by 1 for each later", "not wearing armor", "10 plus", "Dexterity modifier", "Charisma modifier"},
		"dragon-blood-replicant/elemental-affinity":      {"acid", "cold", "fire", "lightning", "poison", "resistance", "Charisma modifier", "one damage roll"},
		"dragon-blood-replicant/dragon-wings":            {"bonus action", "1 hour", "60 feet", "3 mutation points"},
		"dragon-blood-replicant/dragon-companion":        {"without a material component", "without a spell slot", "long rest", "concentration", "1 minute"},
		"shadow-signal-mutant/power-of-shadow":           {"120 feet", "10 feet", "see normally", "0 hit points", "Charisma saving throw", "DC 5", "damage taken", "long rest"},
		"shadow-signal-mutant/beasts-of-ill-omen":        {"3 mutation points", "Summon Beast", "bonus action", "without a spell slot", "preparation", "material components", "5 feet", "disadvantage", "concentration", "1 minute", "again"},
		"shadow-signal-mutant/shadow-walk":               {"dim light or darkness", "bonus action", "120 feet", "unoccupied", "see"},
		"shadow-signal-mutant/umbral-form":               {"Innate Sorcery", "difficult terrain", "1d10 force", "except force and radiant", "6 mutation points"},
		"spellfire-reactor-mutant/spellfire-burst":       {"Once per turn", "at least 1 mutation point", "Magic action", "bonus action", "30 feet", "1d4", "Charisma modifier", "fire or radiant"},
		"spellfire-reactor-mutant/absorb-spells":         {"Counterspell", "fails", "saving throw", "1d4", "point maximum"},
		"spellfire-reactor-mutant/honed-spellfire":       {"Awakened Mutant level", "temporary hit points", "1d8"},
		"spellfire-reactor-mutant/crown-of-spellfire":    {"Innate Sorcery", "once per turn", "attack roll", "Hit Point Dice", "Charisma modifier", "60 feet", "hover", "no damage", "half", "incapacitated", "5 mutation points"},
		"wild-signal-mutant/wild-magic-surge":            {"Once per turn", "spell slot", "1d20", "20", "Metamagic"},
		"wild-signal-mutant/tides-of-chaos":              {"Before rolling", "advantage", "spell slot", "long rest", "automatically"},
		"wild-signal-mutant/bend-luck":                   {"another creature", "see", "reaction", "1 mutation point", "1d4", "add or subtract"},
		"wild-signal-mutant/controlled-chaos":            {"roll twice", "choose"},
		"wild-signal-mutant/tamed-surge":                 {"spell slot", "instead of rolling", "except 97-00", "make it", "long rest"},
	}
	for id, fragments := range checks {
		feature, ok := features[id]
		if !ok {
			t.Errorf("missing feature %s", id)
			continue
		}
		for _, fragment := range fragments {
			if !strings.Contains(feature.Description, fragment) {
				t.Errorf("%s: missing %q", id, fragment)
			}
		}
	}
	resourceChecks := map[string][]string{
		"sector-17-bio-mutant/warping-implosion-use":         {"other", "level 18", "5 mutation points"},
		"kami-synthetic-soul-vessel/restore-balance-uses":    {"long", "Charisma modifier", "minimum 1"},
		"kami-synthetic-soul-vessel/trance-of-order-use":     {"other", "level 14", "5 mutation points"},
		"kami-synthetic-soul-vessel/clockwork-cavalcade-use": {"other", "level 18", "7 mutation points"},
		"dragon-blood-replicant/dragon-wings-use":            {"other", "level 14", "3 mutation points"},
		"dragon-blood-replicant/dragon-companion-free-cast":  {"long", "level 18"},
		"shadow-signal-mutant/strength-of-the-grave-use":     {"long", "successful", "level 3"},
		"shadow-signal-mutant/umbral-form-use":               {"other", "level 18", "6 mutation points"},
		"spellfire-reactor-mutant/crown-of-spellfire-use":    {"other", "level 18", "5 mutation points"},
		"wild-signal-mutant/tides-of-chaos-use":              {"other", "class level 3"},
		"wild-signal-mutant/tamed-surge-use":                 {"long", "level 18"},
	}
	for id, fragments := range resourceChecks {
		for _, fragment := range fragments {
			if !strings.Contains(resources[id], fragment) {
				t.Errorf("%s: resource missing %q", id, fragment)
			}
		}
	}
	affinity := features["dragon-blood-replicant/elemental-affinity"]
	if len(affinity.Choices) != 1 || affinity.Choices[0].Count == nil || *affinity.Choices[0].Count != 1 {
		t.Fatal("elemental affinity must have one permanent choice")
	}
	var options []string
	if err := json.Unmarshal(affinity.Choices[0].AllowedOptions, &options); err != nil || !reflect.DeepEqual(options, []string{"Acid", "Cold", "Fire", "Lightning", "Poison"}) {
		t.Fatalf("incorrect affinity options: %v (%v)", options, err)
	}
}

func TestAwakenedWildSignalSurgeTable(t *testing.T) {
	_, _, subclasses := readMutantWorksheet(t)
	for _, sub := range subclasses {
		if sub.ID != "wild-signal-mutant" {
			continue
		}
		for _, level := range sub.Levels {
			for _, feature := range level.Features {
				if feature.ID != "wild-magic-surge" {
					continue
				}
				if len(feature.SurgeTable) != 25 {
					t.Fatalf("got %d surge rows, want 25", len(feature.SurgeTable))
				}
				for i, row := range feature.SurgeTable {
					end := fmt.Sprintf("%02d", (i+1)*4)
					if i == 24 {
						end = "00"
					}
					want := fmt.Sprintf("%02d-%s", i*4+1, end)
					if row.Roll != want || row.Effect == "" {
						t.Errorf("surge row %d: got %q, want %q with effect", i, row.Roll, want)
					}
				}
				for index, terms := range map[int][]string{
					1:  {"1d4", "Duodrone", "Flumph", "Monodrone", "Unicorn"},
					4:  {"1d8", "24 hours", "Remove Curse"},
					14: {"1d10", "Confusion", "Fireball", "Fog Cloud", "Fly", "Grease", "Levitate", "Magic Missile", "Mirror Image", "Polymorph", "See Invisibility"},
					24: {"1d6", "300 feet", "lowest-level", "mutation points", "17-20"},
				} {
					for _, term := range terms {
						if !strings.Contains(feature.SurgeTable[index].Effect, term) {
							t.Errorf("surge row %s missing %q", feature.SurgeTable[index].Roll, term)
						}
					}
				}
				return
			}
		}
	}
	t.Fatal("Wild Signal Surge table not found")
}
