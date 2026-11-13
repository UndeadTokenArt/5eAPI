package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var progressionFiles2024 = map[string]string{
	"Barbarian": "Cyber-Berserker.json",
	"Bard":      "Net-Hype.json", "Cleric": "Shrine Priest.json",
	"Druid": "Bio-Tech Specialist.json", "Fighter": "Corporate Enforcer.json",
	"Monk": "Kinetic Acrobat.json", "Paladin": "Vow-Bound Operative.json",
	"Ranger": "Recon Drone Master.json", "Rogue": "Decker.json",
	"Sorcerer": "Awakened Mutant.json", "Warlock": "AI Host.json", "Wizard": "Cyber-Mage.json",
}

type abilityRequirement2024 struct {
	Ability string `json:"ability"`
	Minimum int    `json:"minimum"`
}

type abilityAlternative2024 struct {
	AllOf []abilityRequirement2024 `json:"all_of"`
}

type progression2024 struct {
	OriginalClass string `json:"original_class"`
	Prerequisites struct {
		AnyOf []abilityAlternative2024 `json:"any_of"`
	} `json:"multiclass_prerequisites"`
	Spellcasting struct {
		Kind    string `json:"kind"`
		Ability string `json:"ability"`
	} `json:"spellcasting"`
	Levels []struct {
		Level            int `json:"level"`
		SpellProgression *struct {
			Cantrips  *int  `json:"cantrips_known_total"`
			Prepared  *int  `json:"prepared_spells_total"`
			Slots     []int `json:"class_spell_slots"`
			PactCount *int  `json:"pact_slot_count"`
			PactLevel *int  `json:"pact_slot_level"`
		} `json:"spell_progression"`
	} `json:"levels"`
}

func loadProgression2024(original string) (progression2024, error) {
	filename, ok := progressionFiles2024[original]
	if !ok {
		return progression2024{}, fmt.Errorf("2024 progression for %q is not available", original)
	}
	path := guideDataPath("classesv2", filename)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && filepath.Clean(guideDir()) != filepath.Clean(builtinGuideDir) {
		path = filepath.Join(builtinGuideDir, "classesv2", filename)
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		return progression2024{}, fmt.Errorf("read %s: %w", path, err)
	}
	var data progression2024
	if err := json.Unmarshal(raw, &data); err != nil {
		return data, fmt.Errorf("decode %s: %w", path, err)
	}
	if data.OriginalClass != original || len(data.Prerequisites.AnyOf) == 0 || len(data.Levels) != 20 {
		return data, fmt.Errorf("%s: incomplete 2024 progression", path)
	}
	for _, alt := range data.Prerequisites.AnyOf {
		if len(alt.AllOf) == 0 {
			return data, fmt.Errorf("%s: empty prerequisite alternative", path)
		}
		for _, req := range alt.AllOf {
			if !abilityNames[req.Ability] || req.Minimum < 1 || req.Minimum > 30 {
				return data, fmt.Errorf("%s: invalid ability prerequisite", path)
			}
		}
	}
	for i, level := range data.Levels {
		if level.Level != i+1 {
			return data, fmt.Errorf("%s: invalid class level %d", path, i+1)
		}
	}
	return data, nil
}

func abilityScore2024(scores AbilityScores, ability string) int {
	switch ability {
	case "strength":
		return scores.Strength
	case "dexterity":
		return scores.Dexterity
	case "constitution":
		return scores.Constitution
	case "intelligence":
		return scores.Intelligence
	case "wisdom":
		return scores.Wisdom
	case "charisma":
		return scores.Charisma
	default:
		return 0
	}
}

func meetsPrerequisites2024(data progression2024, scores AbilityScores) bool {
	for _, alternative := range data.Prerequisites.AnyOf {
		valid := true
		for _, req := range alternative.AllOf {
			if abilityScore2024(scores, req.Ability) < req.Minimum {
				valid = false
				break
			}
		}
		if valid {
			return true
		}
	}
	return false
}

type multiclassEligibilityRequest struct {
	Classes       []CharacterClass `json:"classes"`
	NewClass      string           `json:"new_class"`
	AbilityScores AbilityScores    `json:"ability_scores"`
}

type multiclassEligibilityResult struct {
	Eligible     bool                        `json:"eligible"`
	Unqualified  []string                    `json:"unqualified_classes"`
	Requirements []multiclassRequirement2024 `json:"requirements"`
}

type multiclassRequirement2024 struct {
	Class string                   `json:"class"`
	AnyOf []abilityAlternative2024 `json:"any_of"`
}

func checkMulticlassEligibility2024(request multiclassEligibilityRequest, labels map[string]string) (multiclassEligibilityResult, error) {
	result := multiclassEligibilityResult{Eligible: true, Unqualified: []string{}, Requirements: []multiclassRequirement2024{}}
	if len(request.Classes) == 0 || len(request.Classes) > 20 {
		return result, errors.New("provide 1 to 20 current classes")
	}
	for _, ability := range []string{"strength", "dexterity", "constitution", "intelligence", "wisdom", "charisma"} {
		if n := abilityScore2024(request.AbilityScores, ability); n < 1 || n > 30 {
			return result, errors.New("ability scores must be between 1 and 30")
		}
	}
	seen := map[string]bool{}
	total := 0
	classes := append(append([]CharacterClass(nil), request.Classes...), CharacterClass{Name: request.NewClass, Level: 1})
	for _, class := range classes {
		original := originalClassFor(class.Name, labels)
		if original == "" || class.Level < 1 || class.Level > 20 || seen[original] {
			return result, fmt.Errorf("unknown or duplicate class %q", class.Name)
		}
		total += class.Level
		if total > 20 {
			return result, errors.New("new total character level cannot exceed 20")
		}
		seen[original] = true
		data, err := loadProgression2024(original)
		if err != nil {
			return result, err
		}
		result.Requirements = append(result.Requirements, multiclassRequirement2024{Class: class.Name, AnyOf: data.Prerequisites.AnyOf})
		if !meetsPrerequisites2024(data, request.AbilityScores) {
			result.Eligible = false
			result.Unqualified = append(result.Unqualified, class.Name)
		}
	}
	return result, nil
}

func multiclassEligibilityHandler(w http.ResponseWriter, r *http.Request) {
	if user, _ := characterUser(w, r); user == "" {
		writeCharacterError(w, http.StatusUnauthorized, "Sign in through Authelia to manage characters.")
		return
	}
	if !sameOriginMutation(w, r) {
		return
	}
	var request multiclassEligibilityRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeCharacterError(w, http.StatusBadRequest, "Invalid multiclass eligibility request.")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeCharacterError(w, http.StatusBadRequest, "Invalid multiclass eligibility request.")
		return
	}
	labels, err := classLabels()
	if err != nil {
		log.Printf("load class labels: %v", err)
		writeCharacterError(w, http.StatusInternalServerError, "Could not load multiclass rules.")
		return
	}
	result, err := checkMulticlassEligibility2024(request, labels)
	if err != nil {
		log.Printf("multiclass eligibility: %v", err)
		writeCharacterError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeCharacterJSON(w, http.StatusOK, result)
}
