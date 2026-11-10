package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// MulticlassSpellcasting separates shared Spellcasting slots from Pact Magic.
type MulticlassSpellcasting struct {
	Slots   []int             `json:"slots"`
	Pact    *SpellSlot        `json:"pact,omitempty"`
	Classes []ClassSpellRules `json:"classes"`
}

type ClassSpellRules struct {
	Class         string          `json:"class"`
	Ability       string          `json:"ability"`
	Kind          string          `json:"kind"`
	PreparedMode  string          `json:"prepared_mode,omitempty"`
	SpellListFrom string          `json:"spell_list_from"`
	Row           SpellcastingRow `json:"row"`
	Arcanum       []int           `json:"arcanum,omitempty"`
}

func calculateMulticlassSpellcasting(classes []CharacterClass, labels map[string]string) (MulticlassSpellcasting, error) {
	result := MulticlassSpellcasting{Slots: make([]int, 9), Classes: []ClassSpellRules{}}
	seen := map[string]bool{}
	total, casters, casterLevel := 0, 0, 0
	var single []int
	for _, class := range classes {
		name := strings.TrimSpace(class.Name)
		original := originalClassFor(name, labels)
		if name == "" || class.Level < 1 || class.Level > 20 || original == "" || seen[original] {
			return result, fmt.Errorf("invalid or duplicate class %q", class.Name)
		}
		seen[original] = true
		total += class.Level
		if total > 20 {
			return result, errors.New("total class levels cannot exceed 20")
		}
		if original == "Artificer" {
			return result, errors.New("artificer spellcasting progression is not supported")
		}
		if parent, ok := thirdCasterSubclasses[class.Subclass]; ok {
			if original != parent {
				return result, fmt.Errorf("subclass %q does not belong to %q", class.Subclass, class.Name)
			}
			return result, fmt.Errorf("2024 spellcasting for subclass %q has not been authored", class.Subclass)
		}
		progression, err := loadProgression2024(original)
		if err != nil {
			return result, err
		}
		kind := progression.Spellcasting.Kind
		if kind == "none" {
			continue
		}
		if kind != "full" && kind != "half" && kind != "pact" {
			return result, fmt.Errorf("unsupported 2024 spellcasting kind %q for %q", kind, name)
		}
		spell := progression.Levels[class.Level-1].SpellProgression
		if spell == nil || spell.Prepared == nil || *spell.Prepared < 0 || !abilityNames[progression.Spellcasting.Ability] {
			return result, fmt.Errorf("incomplete 2024 spellcasting progression for %q level %d", name, class.Level)
		}
		row := SpellcastingRow{Level: class.Level, Slots: make([]int, 9), Prepared: *spell.Prepared}
		if spell.Cantrips != nil {
			if *spell.Cantrips < 0 {
				return result, fmt.Errorf("invalid cantrip count for %q", name)
			}
			row.Cantrips = *spell.Cantrips
		}
		if kind == "pact" {
			if spell.PactCount == nil || spell.PactLevel == nil || *spell.PactCount < 1 || *spell.PactLevel < 1 || *spell.PactLevel > 5 {
				return result, fmt.Errorf("invalid pact slots for %q", name)
			}
			row.PactSlots, row.PactLevel = *spell.PactCount, *spell.PactLevel
			row.MaxSpellLevel = row.PactLevel
		} else {
			if len(spell.Slots) != 9 {
				return result, fmt.Errorf("invalid 2024 slot table for %q", name)
			}
			for i, count := range spell.Slots {
				if count < 0 || count > 99 {
					return result, fmt.Errorf("invalid 2024 slots for %q", name)
				}
				row.Slots[i] = count
				if count > 0 {
					row.MaxSpellLevel = i + 1
				}
			}
		}
		rules := ClassSpellRules{Class: class.Name, Ability: progression.Spellcasting.Ability, Kind: kind, PreparedMode: "table", SpellListFrom: name, Row: row}
		if kind == "pact" {
			result.Pact = &SpellSlot{Level: row.PactLevel, Max: row.PactSlots, Current: row.PactSlots}
			for level := 11; level <= class.Level; level++ {
				if arcanum, ok := warlockArcanum[level]; ok {
					rules.Arcanum = append(rules.Arcanum, arcanum)
				}
			}
		} else if row.MaxSpellLevel > 0 {
			casters++
			single = row.Slots
			switch kind {
			case "full":
				casterLevel += class.Level
			case "half":
				casterLevel += (class.Level + 1) / 2
			default:
				return result, fmt.Errorf("unsupported spellcasting kind %q", kind)
			}
		}
		result.Classes = append(result.Classes, rules)
	}
	if casters == 1 {
		copy(result.Slots, single)
	} else if casters > 1 {
		copy(result.Slots, fullCasterSlots[casterLevel-1])
	}
	return result, nil
}

func multiclassSpellcastingHandler(w http.ResponseWriter, r *http.Request) {
	if user, _ := characterUser(w, r); user == "" {
		writeCharacterError(w, http.StatusUnauthorized, "Sign in through Authelia to manage characters.")
		return
	}
	if !sameOriginMutation(w, r) {
		return
	}
	var request struct {
		Classes []CharacterClass `json:"classes"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeCharacterError(w, http.StatusBadRequest, "Invalid class progression.")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) || len(request.Classes) == 0 || len(request.Classes) > 20 {
		writeCharacterError(w, http.StatusBadRequest, "Provide one class progression with 1 to 20 classes.")
		return
	}
	labels, err := classLabels()
	if err != nil {
		writeCharacterError(w, http.StatusInternalServerError, "Could not load class spellcasting rules.")
		return
	}
	result, err := calculateMulticlassSpellcasting(request.Classes, labels)
	if err != nil {
		writeCharacterError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeCharacterJSON(w, http.StatusOK, result)
}
