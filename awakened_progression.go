package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type awakenedChoice struct {
	Name            string          `json:"name"`
	Count           *int            `json:"count"`
	AllowedOptions  json.RawMessage `json:"allowed_options"`
	EligibilityRule string          `json:"eligibility_rule"`
	SelectionEffect string          `json:"selection_effect"`
}

type awakenedFeature struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Choices     []awakenedChoice `json:"choices"`
}

type awakenedLevel struct {
	Level            int               `json:"level"`
	Status           string            `json:"status"`
	Features         []awakenedFeature `json:"features"`
	Notes            *string           `json:"notes"`
	SpellProgression struct {
		CantripsKnownTotal  int   `json:"cantrips_known_total"`
		PreparedSpellsTotal int   `json:"prepared_spells_total"`
		ClassSpellSlots     []int `json:"class_spell_slots"`
	} `json:"spell_progression"`
}

type awakenedProgression struct {
	Name   string          `json:"name"`
	HitDie int             `json:"hit_die"`
	Levels []awakenedLevel `json:"levels"`
}

func loadAwakenedProgression() (awakenedProgression, error) {
	path := guideDataPath("classesv2", "Awakened Mutant.json")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) && filepath.Clean(guideDir()) != filepath.Clean(builtinGuideDir) {
		path = filepath.Join(builtinGuideDir, "classesv2", "Awakened Mutant.json")
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		return awakenedProgression{}, fmt.Errorf("read %s: %w", path, err)
	}
	var data awakenedProgression
	if err := json.Unmarshal(raw, &data); err != nil {
		return awakenedProgression{}, fmt.Errorf("decode %s: %w", path, err)
	}
	if data.Name != "Awakened Mutant" || data.HitDie != 6 || len(data.Levels) != 20 {
		return awakenedProgression{}, fmt.Errorf("%s: invalid class identity or levels", path)
	}
	sc := buildSpellcasting(spellProfiles["Sorcerer"], "Awakened Mutant")
	for i, level := range data.Levels {
		if level.Level != i+1 || level.Features == nil || len(level.SpellProgression.ClassSpellSlots) != 9 ||
			level.SpellProgression.CantripsKnownTotal != sc.Rows[i].Cantrips ||
			level.SpellProgression.PreparedSpellsTotal != sc.Rows[i].Prepared {
			return awakenedProgression{}, fmt.Errorf("%s: invalid or inconsistent progression at level %d", path, i+1)
		}
		for slot, count := range level.SpellProgression.ClassSpellSlots {
			if count != sc.Rows[i].Slots[slot] {
				return awakenedProgression{}, fmt.Errorf("%s: inconsistent spell slots at level %d", path, i+1)
			}
		}
		for _, feature := range level.Features {
			if feature.ID == "" || feature.Name == "" || feature.Description == "" {
				return awakenedProgression{}, fmt.Errorf("%s: incomplete feature at level %d", path, i+1)
			}
		}
	}
	return data, nil
}

func awakenedProgressionHandler(w http.ResponseWriter, r *http.Request) {
	if user, _ := characterUser(w, r); user == "" {
		writeCharacterError(w, http.StatusUnauthorized, "Sign in through Authelia to manage characters.")
		return
	}
	data, err := loadAwakenedProgression()
	if err != nil {
		log.Printf("load Awakened Mutant progression: %v", err)
		writeCharacterError(w, http.StatusInternalServerError, "Could not load Awakened Mutant progression.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeCharacterJSON(w, http.StatusOK, data)
}
