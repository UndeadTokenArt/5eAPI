package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

type shrineChoice struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Kind            string          `json:"kind"`
	Count           *int            `json:"count"`
	AllowedOptions  json.RawMessage `json:"allowed_options"`
	EligibilityRule string          `json:"eligibility_rule"`
	SelectionEffect string          `json:"selection_effect"`
}

type shrineFeature struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Description          string         `json:"description"`
	When                 string         `json:"when"`
	Choices              []shrineChoice `json:"choices"`
	AlwaysPreparedSpells []string       `json:"always_prepared_spells"`
}

type shrineLevel struct {
	Level            int             `json:"level"`
	Status           string          `json:"status"`
	Features         []shrineFeature `json:"features"`
	Notes            *string         `json:"notes"`
	SpellProgression struct {
		MaxClassSpellLevel int `json:"max_class_spell_level"`
	} `json:"spell_progression"`
}

type shrineSubclass struct {
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
	Levels []shrineLevel `json:"levels"`
}

func readShrineWorksheet(t *testing.T) (string, []shrineLevel, []shrineSubclass) {
	t.Helper()
	raw, err := os.ReadFile("static/data/neotokyo/classesv2/Shrine Priest.json")
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		SchemaStatus string           `json:"schema_status"`
		Levels       []shrineLevel    `json:"levels"`
		Subclasses   []shrineSubclass `json:"subclasses"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	return data.SchemaStatus, data.Levels, data.Subclasses
}

func TestShrinePriestDomainWorksheet(t *testing.T) {
	status, base, domains := readShrineWorksheet(t)
	if status != "draft_authoring_template_not_supported_by_app" || len(base) != 20 {
		t.Fatal("worksheet must remain a 20-level unsupported draft")
	}
	type expectation struct {
		name     string
		spells   map[int][]string
		features map[int][]string
	}
	expected := map[string]expectation{
		"arcane-circuit-domain": {
			"Arcane Circuit Domain",
			map[int][]string{3: {"Detect Magic", "Magic Missile", "Magic Weapon", "Nystul's Magic Aura"}, 5: {"Counterspell", "Dispel Magic"}, 7: {"Arcane Eye", "Leomund's Secret Chest"}, 9: {"Bigby's Hand", "Teleportation Circle"}},
			map[int][]string{3: {"arcana-spells-3", "modify-magic", "student-of-arcana"}, 6: {"dispelling-recovery"}, 17: {"magical-mastery"}},
		},
		"twin-signal-discipline-domain": {
			"Twin-Signal Discipline Domain",
			map[int][]string{3: {"Arcane Vigor", "Guiding Bolt", "Magic Weapon", "Shield of Faith"}, 5: {"Prayer of Mending", "Protection from Energy"}, 7: {"Guardian of Faith", "Stoneskin"}, 9: {"Circle of Power", "Destructive Wave"}},
			map[int][]string{3: {"discipline-spells-3", "holy-bolt", "atonement"}, 6: {"luminous-barrier"}, 17: {"ultimate-penitence"}},
		},
		"threshold-guardian-domain": {
			"Threshold Guardian Domain",
			map[int][]string{3: {"Detect Evil and Good", "False Life", "Gentle Repose", "Ray of Enfeeblement", "Spare the Dying"}, 5: {"Revivify", "Vampiric Touch"}, 7: {"Blight", "Death Ward"}, 9: {"Dispel Evil and Good", "Raise Dead"}},
			map[int][]string{3: {"grave-spells-3", "circle-of-mortality", "path-to-the-grave"}, 6: {"sentinel-at-deaths-door"}, 11: {"circle-of-mortality-improvement"}, 17: {"divine-reaper"}},
		},
		"data-shrine-knowledge-domain": {
			"Data Shrine Knowledge Domain",
			map[int][]string{3: {"Command", "Comprehend Languages", "Detect Magic", "Detect Thoughts", "Identify", "Mind Spike"}, 5: {"Dispel Magic", "Nondetection", "Tongues"}, 7: {"Arcane Eye", "Banishment", "Confusion"}, 9: {"Legend Lore", "Scrying", "Synaptic Static"}},
			map[int][]string{3: {"blessings-of-knowledge", "knowledge-spells-3", "mind-magic"}, 6: {"unfettered-mind"}, 17: {"divine-foreknowledge"}},
		},
		"trauma-care-life-domain": {
			"Trauma Care Life Domain",
			map[int][]string{3: {"Aid", "Bless", "Cure Wounds", "Lesser Restoration"}, 5: {"Mass Healing Word", "Revivify"}, 7: {"Aura of Life", "Death Ward"}, 9: {"Greater Restoration", "Mass Cure Wounds"}},
			map[int][]string{3: {"disciple-of-life", "life-spells-3", "preserve-life"}, 6: {"blessed-healer"}, 17: {"supreme-healing"}},
		},
		"neon-radiance-domain": {
			"Neon Radiance Domain",
			map[int][]string{3: {"Burning Hands", "Faerie Fire", "Scorching Ray", "See Invisibility"}, 5: {"Daylight", "Fireball"}, 7: {"Arcane Eye", "Wall of Fire"}, 9: {"Flame Strike", "Scrying"}},
			map[int][]string{3: {"light-spells-3", "radiance-of-the-dawn", "warding-flare"}, 6: {"improved-warding-flare"}, 17: {"corona-of-light"}},
		},
		"void-signal-domain": {
			"Void Signal Domain",
			map[int][]string{3: {"Crown of Madness", "Dissonant Whispers", "Mind Spike", "Shadow Word: Pain"}, 5: {"Summon Undead", "Vampiric Touch"}, 7: {"Evard's Black Tentacles", "Psychic Horror"}, 9: {"Dominate Person", "Void Eruption"}},
			map[int][]string{3: {"shadow-domain-spells-3", "shadowform", "devouring-plague"}, 6: {"dispersion"}, 17: {"vampiric-embrace"}},
		},
		"ghost-image-trickery-domain": {
			"Ghost-Image Trickery Domain",
			map[int][]string{3: {"Charm Person", "Disguise Self", "Invisibility", "Pass without Trace"}, 5: {"Hypnotic Pattern", "Nondetection"}, 7: {"Confusion", "Dimension Door"}, 9: {"Dominate Person", "Modify Memory"}},
			map[int][]string{3: {"blessing-of-the-trickster", "trickery-spells-3", "invoke-duplicity"}, 6: {"tricksters-transposition"}, 17: {"improved-duplicity"}},
		},
		"battle-shrine-domain": {
			"Battle Shrine Domain",
			map[int][]string{3: {"Guiding Bolt", "Magic Weapon", "Shield of Faith", "Spiritual Weapon"}, 5: {"Crusader's Mantle", "Spirit Guardians"}, 7: {"Fire Shield", "Freedom of Movement"}, 9: {"Hold Monster", "Steel Wind Strike"}},
			map[int][]string{3: {"guided-strike", "war-spells-3", "war-priest"}, 6: {"war-gods-blessing"}, 17: {"avatar-of-battle"}},
		},
	}
	if len(domains) != len(expected) {
		t.Fatalf("got %d domains, want %d", len(domains), len(expected))
	}
	names, seen := map[string]bool{}, map[string]bool{}
	for _, domain := range domains {
		want, ok := expected[domain.ID]
		if !ok || seen[domain.ID] {
			t.Fatalf("unexpected or duplicate domain %q", domain.ID)
		}
		seen[domain.ID], names[domain.Name] = true, true
		if domain.Name != want.name || domain.Description == "" || domain.SelectedAtClassLevel != 3 || string(domain.SpellcastingOverride) != "null" || domain.Resources == nil {
			t.Errorf("%s: incomplete domain metadata", domain.ID)
		}
		featureIDs, choiceIDs, resourceIDs := map[string]bool{}, map[string]bool{}, map[string]bool{}
		for _, resource := range domain.Resources {
			if resource.ID == "" || resourceIDs[resource.ID] || resource.Name == "" || resource.Description == "" || resource.Recharge == "" || resource.MaximumRule == "" {
				t.Errorf("%s: invalid resource %q", domain.ID, resource.ID)
			}
			resourceIDs[resource.ID] = true
		}
		byLevel := map[int]map[string]shrineFeature{}
		gotSpells := map[int][]string{}
		last := 0
		for _, level := range domain.Levels {
			if level.Level <= last || level.Level > 20 || level.Status != "unfilled" || len(level.Features) == 0 {
				t.Fatalf("%s: invalid level %d", domain.ID, level.Level)
			}
			last = level.Level
			byLevel[level.Level] = map[string]shrineFeature{}
			for _, feature := range level.Features {
				if feature.ID == "" || featureIDs[feature.ID] || feature.Name == "" || feature.Description == "" || feature.When != "both" || feature.Choices == nil {
					t.Errorf("%s level %d: invalid feature %q", domain.ID, level.Level, feature.ID)
				}
				featureIDs[feature.ID] = true
				byLevel[level.Level][feature.ID] = feature
				if len(feature.AlwaysPreparedSpells) > 0 {
					gotSpells[level.Level] = append(gotSpells[level.Level], feature.AlwaysPreparedSpells...)
					for _, term := range []string{"always prepared", "Wisdom", "prepared-spell limit"} {
						if !strings.Contains(feature.Description, term) {
							t.Errorf("%s: %s missing %q", domain.ID, feature.ID, term)
						}
					}
				}
				for _, choice := range feature.Choices {
					if choice.ID == "" || choiceIDs[choice.ID] || choice.Name == "" || choice.Kind == "" || len(choice.AllowedOptions) == 0 || string(choice.AllowedOptions) == "null" || choice.EligibilityRule == "" || choice.SelectionEffect == "" {
						t.Errorf("%s: invalid choice %q", domain.ID, choice.ID)
					}
					choiceIDs[choice.ID] = true
				}
			}
		}
		for level, ids := range want.features {
			for _, id := range ids {
				if _, ok := byLevel[level][id]; !ok {
					t.Errorf("%s: missing %s at level %d", domain.ID, id, level)
				}
			}
		}
		if !reflect.DeepEqual(gotSpells, want.spells) {
			t.Errorf("%s: spells %v, want %v", domain.ID, gotSpells, want.spells)
		}
		wantLevels := 6
		if domain.ID == "threshold-guardian-domain" {
			wantLevels = 7
		}
		if len(domain.Levels) != wantLevels {
			t.Errorf("%s: got %d subclass levels, want %d", domain.ID, len(domain.Levels), wantLevels)
		}
	}
	for _, level := range []int{3, 5, 7, 9} {
		if got, want := base[level-1].SpellProgression.MaxClassSpellLevel, (level+1)/2; got != want {
			t.Errorf("class level %d grants spell level %d, want %d", level, got, want)
		}
	}
	var options []string
	for _, feature := range base[2].Features {
		if feature.ID == "cleric-subclass-3" && len(feature.Choices) == 1 && feature.Choices[0].Count != nil && *feature.Choices[0].Count == 1 {
			if err := json.Unmarshal(feature.Choices[0].AllowedOptions, &options); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(options) != len(names) {
		t.Fatalf("domain options %v do not match authored names", options)
	}
	for _, option := range options {
		if !names[option] {
			t.Errorf("unexpected or duplicate option %q", option)
		}
		delete(names, option)
	}
}

func TestShrinePriestDomainMechanics(t *testing.T) {
	_, _, domains := readShrineWorksheet(t)
	features := map[string]shrineFeature{}
	resources := map[string]string{}
	for _, domain := range domains {
		for _, level := range domain.Levels {
			for _, feature := range level.Features {
				features[domain.ID+"/"+feature.ID] = feature
			}
		}
		for _, resource := range domain.Resources {
			resources[domain.ID+"/"+resource.ID] = resource.Recharge + " " + resource.MaximumRule
		}
	}
	checks := map[string][]string{
		"arcane-circuit-domain/modify-magic":                    {"Channel Divinity", "without an action", "2d8", "level", "succeeds", "1d6", "first saving throw"},
		"arcane-circuit-domain/student-of-arcana":               {"Arcana", "two Wizard cantrips", "replace one"},
		"arcane-circuit-domain/dispelling-recovery":             {"spell slot", "restores hit points", "ends a condition", "same action", "bonus action", "reaction", "without a spell slot", "short or long rest", "Channel Divinity"},
		"arcane-circuit-domain/magical-mastery":                 {"6, 7, 8, and 9", "always prepared", "same level"},
		"twin-signal-discipline-domain/holy-bolt":               {"bonus action", "Channel Divinity", "60 feet", "ranged spell attack", "2d10", "1d10", "Wisdom modifier"},
		"twin-signal-discipline-domain/atonement":               {"level 1 or higher", "end of your next turn", "spell slot", "deals damage", "slot level"},
		"twin-signal-discipline-domain/luminous-barrier":        {"bonus action", "Channel Divinity", "temporary hit points", "level plus Wisdom", "resistance to all damage"},
		"twin-signal-discipline-domain/ultimate-penitence":      {"teleport", "30 feet", "before", "spell slot", "additional bolts", "separate attack roll"},
		"threshold-guardian-domain/circle-of-mortality":         {"Once per turn", "missing any hit points", "1d4", "1d6", "level 11", "bonus action", "0 hit points", "maximum"},
		"threshold-guardian-domain/path-to-the-grave":           {"bonus action", "Channel Divinity", "see", "30 feet", "start of your next turn", "disadvantage", "ally", "without an action"},
		"threshold-guardian-domain/sentinel-at-deaths-door":     {"bloodied", "60 feet", "reaction", "halve", "rounding down", "critical hit", "Wisdom modifier", "long rest"},
		"threshold-guardian-domain/divine-reaper":               {"Necromancy", "level 5 or lower", "Channel Divinity", "second creature", "material components", "enemy dies", "twice your Shrine Priest level", "incapacitated", "short or long rest", "level 6"},
		"data-shrine-knowledge-domain/blessings-of-knowledge":   {"artisan", "proficiency", "expertise", "two"},
		"data-shrine-knowledge-domain/mind-magic":               {"Magic action", "Channel Divinity", "prepared Divination spell", "without a spell slot or material components"},
		"data-shrine-knowledge-domain/unfettered-mind":          {"telepathy", "60 feet", "Wisdom modifier", "minimum one", "Intelligence saving throw", "already proficient"},
		"data-shrine-knowledge-domain/divine-foreknowledge":     {"bonus action", "advantage", "D20 Tests", "1 hour", "long rest", "level 6"},
		"trauma-care-life-domain/disciple-of-life":              {"spell slot", "on the turn", "2 plus"},
		"trauma-care-life-domain/preserve-life":                 {"Magic action", "Channel Divinity", "five times", "bloodied", "30 feet", "half"},
		"trauma-care-life-domain/blessed-healer":                {"Immediately after", "spell slot", "other than yourself", "2 plus"},
		"trauma-care-life-domain/supreme-healing":               {"spell or Channel Divinity", "highest possible"},
		"neon-radiance-domain/radiance-of-the-dawn":             {"Magic action", "Channel Divinity", "30-foot", "magical darkness", "Constitution", "2d10", "radiant", "half"},
		"neon-radiance-domain/warding-flare":                    {"see", "30 feet", "reaction", "before", "disadvantage", "Wisdom modifier", "long rest"},
		"neon-radiance-domain/improved-warding-flare":           {"short or long rest", "target", "2d6", "Wisdom modifier"},
		"neon-radiance-domain/corona-of-light":                  {"Magic action", "sunlight", "1 minute", "60-foot", "30 feet", "disadvantage", "fire or radiant", "long rest"},
		"void-signal-domain/shadowform":                         {"bonus action", "Channel Divinity", "1 minute", "Hide", "spell slot", "psychic or necrotic", "1d6", "incapacitated", "radiant"},
		"void-signal-domain/devouring-plague":                   {"see", "30 feet", "fails a saving throw", "1d10 necrotic", "damage dealt", "long rest"},
		"void-signal-domain/dispersion":                         {"reaction", "end of your next turn", "all damage", "triggering attack", "cannot take", "40 feet", "half", "Hit Point Dice", "short or long rest"},
		"void-signal-domain/vampiric-embrace":                   {"whenever", "1d6", "spell", "30 feet"},
		"ghost-image-trickery-domain/blessing-of-the-trickster": {"Magic action", "willing", "30 feet", "Stealth", "long rest", "again"},
		"ghost-image-trickery-domain/invoke-duplicity":          {"bonus action", "Channel Divinity", "30 feet", "intangible", "1 minute", "own senses", "5 feet", "advantage", "120 feet"},
		"ghost-image-trickery-domain/tricksters-transposition":  {"bonus action", "create or move", "swapping"},
		"ghost-image-trickery-domain/improved-duplicity":        {"allies", "advantage", "5 feet", "illusion ends", "Shrine Priest level"},
		"battle-shrine-domain/guided-strike":                    {"30 feet", "misses", "Channel Divinity", "+10", "reaction"},
		"battle-shrine-domain/war-priest":                       {"bonus action", "weapon attack", "unarmed strike", "Wisdom modifier", "short or long rest"},
		"battle-shrine-domain/war-gods-blessing":                {"Channel Divinity", "Shield of Faith", "Spiritual Weapon", "no concentration", "1 minute", "again", "incapacitated"},
		"battle-shrine-domain/avatar-of-battle":                 {"resistance", "bludgeoning", "piercing", "slashing"},
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
		"arcane-circuit-domain/dispelling-recovery-use":          {"other", "level 6", "short or long rest", "Channel Divinity"},
		"threshold-guardian-domain/sentinel-at-deaths-door-uses": {"long", "Wisdom modifier", "minimum 1", "level 6"},
		"threshold-guardian-domain/keeper-of-souls-use":          {"other", "level 17", "short or long rest", "level 6"},
		"data-shrine-knowledge-domain/divine-foreknowledge-use":  {"other", "level 17", "long rest", "level 6"},
		"neon-radiance-domain/warding-flare-uses":                {"other", "minimum 1", "long rest", "short or long rest", "level 6"},
		"neon-radiance-domain/corona-of-light-uses":              {"long", "minimum 1", "level 17"},
		"void-signal-domain/devouring-plague-uses":               {"long", "minimum 1", "level 3"},
		"void-signal-domain/dispersion-use":                      {"short", "level 6"},
		"battle-shrine-domain/war-priest-uses":                   {"short", "minimum 1", "level 3"},
	}
	for id, fragments := range resourceChecks {
		for _, fragment := range fragments {
			if !strings.Contains(resources[id], fragment) {
				t.Errorf("%s: resource missing %q", id, fragment)
			}
		}
	}
	choiceChecks := map[string]map[string]int{
		"arcane-circuit-domain/student-of-arcana":             {"arcana-student-skill": 1, "arcana-student-cantrips": 2},
		"arcane-circuit-domain/magical-mastery":               {"master-compiler-spells": 4},
		"data-shrine-knowledge-domain/blessings-of-knowledge": {"archive-artisan-tools": 1, "archive-expertise": 2},
		"data-shrine-knowledge-domain/mind-magic":             {"archive-query-spell": 1},
		"data-shrine-knowledge-domain/unfettered-mind":        {"neural-network-save": 1},
	}
	for id, want := range choiceChecks {
		for _, choice := range features[id].Choices {
			if count, ok := want[choice.ID]; ok && choice.Count != nil && *choice.Count == count {
				delete(want, choice.ID)
			}
		}
		if len(want) != 0 {
			t.Errorf("%s: missing choices or incorrect counts: %v", id, want)
		}
	}
}
