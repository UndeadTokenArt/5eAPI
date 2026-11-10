package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMulticlassSpellcasting(t *testing.T) {
	labels, err := classLabels()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		classes []CharacterClass
		slots   []int
		pact    *SpellSlot
	}{
		{"single half caster with noncaster", []CharacterClass{{Name: labels["Paladin"], Level: 3}, {Name: labels["Fighter"], Level: 2}}, []int{3, 0, 0, 0, 0, 0, 0, 0, 0}, nil},
		{"full and half caster", []CharacterClass{{Name: labels["Wizard"], Level: 3}, {Name: labels["Paladin"], Level: 3}}, []int{4, 3, 2, 0, 0, 0, 0, 0, 0}, nil},
		{"two level one half casters", []CharacterClass{{Name: labels["Paladin"], Level: 1}, {Name: labels["Ranger"], Level: 1}}, []int{3, 0, 0, 0, 0, 0, 0, 0, 0}, nil},
		{"level one half and full", []CharacterClass{{Name: labels["Paladin"], Level: 1}, {Name: labels["Sorcerer"], Level: 1}}, []int{3, 0, 0, 0, 0, 0, 0, 0, 0}, nil},
		{"pact and normal slots separate", []CharacterClass{{Name: labels["Wizard"], Level: 3}, {Name: labels["Warlock"], Level: 5}}, []int{4, 2, 0, 0, 0, 0, 0, 0, 0}, &SpellSlot{Level: 3, Current: 2, Max: 2}},
		{"pact only arcanum is not slots", []CharacterClass{{Name: labels["Warlock"], Level: 11}}, []int{0, 0, 0, 0, 0, 0, 0, 0, 0}, &SpellSlot{Level: 5, Current: 3, Max: 3}},
	}
	sorcerer, err := calculateMulticlassSpellcasting([]CharacterClass{{Name: labels["Sorcerer"], Level: 3}}, labels)
	if err != nil || len(sorcerer.Classes) != 1 || sorcerer.Classes[0].Row.Prepared != 6 || sorcerer.Classes[0].Row.Cantrips != 4 {
		t.Fatalf("2024 Sorcerer progression = %+v, error %v", sorcerer, err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculateMulticlassSpellcasting(tt.classes, labels)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Slots, tt.slots) || !reflect.DeepEqual(got.Pact, tt.pact) {
				t.Fatalf("got slots %v and pact %+v, want %v and %+v", got.Slots, got.Pact, tt.slots, tt.pact)
			}
			if tt.name == "pact only arcanum is not slots" && (!reflect.DeepEqual(got.Classes[0].Arcanum, []int{6}) || got.Classes[0].Row.MaxSpellLevel != 5 || got.Classes[0].Row.Slots[5] != 0) {
				t.Fatalf("pact rules incorrectly represent arcanum: %+v", got.Classes[0])
			}
		})
	}
	for _, classes := range [][]CharacterClass{
		{{Name: "Unknown", Level: 1}},
		{{Name: labels["Wizard"], Level: 0}},
		{{Name: labels["Wizard"], Level: 20}, {Name: labels["Warlock"], Level: 1}},
		{{Name: labels["Wizard"], Level: 1}, {Name: labels["Wizard"], Level: 1}},
		{{Name: labels["Wizard"], Subclass: "Cyber-Mage Ronin", Level: 3}},
		{{Name: labels["Fighter"], Subclass: "Cyber-Mage Ronin", Level: 3}},
	} {
		if _, err := calculateMulticlassSpellcasting(classes, labels); err == nil {
			t.Errorf("expected invalid progression to be rejected: %+v", classes)
		}
	}
}

func TestCharacterProgressionPersistenceAndValidation(t *testing.T) {
	sheet := validCharacter("Multiclass")
	sheet.Class, sheet.Subclass = "Cyber-Mage", ""
	sheet.Classes = []CharacterClass{{Name: "Cyber-Mage", Level: 2}, {Name: "AI Host", Level: 1}}
	sheet.PactSlots = &SpellSlot{Level: 1, Current: 0, Max: 1}
	sheet.Spells[0].SpellClass = "Cyber-Mage"
	if err := validateCharacter(sheet); err != nil {
		t.Fatal(err)
	}
	store := newCharacterStore(filepath.Join(t.TempDir(), "characters.json"))
	if _, err := store.create("alice", sheet); err != nil {
		t.Fatal(err)
	}
	saved, err := store.list("alice")
	if err != nil || len(saved) != 1 || !reflect.DeepEqual(saved[0].Classes, sheet.Classes) || !reflect.DeepEqual(saved[0].PactSlots, sheet.PactSlots) || saved[0].Spells[0].SpellClass != "Cyber-Mage" {
		t.Fatalf("persisted sheet: %+v; error: %v", saved, err)
	}
	legacy := validCharacter("Legacy")
	legacy.Class = "Cyber-Mage"
	if err := validateCharacter(legacy); err != nil {
		t.Fatalf("legacy sheet rejected: %v", err)
	}
	sheet.Classes[1].Level = 2
	if err := validateCharacter(sheet); err == nil {
		t.Fatal("mismatched total accepted")
	}
}

func TestMulticlassSpellcastingEndpoint(t *testing.T) {
	body := []byte(`{"classes":[{"name":"Cyber-Mage","level":3},{"name":"AI Host","level":5}]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/character-spellcasting", bytes.NewReader(body))
	request.Header.Set("Remote-User", "alice")
	request.Header.Set("Origin", "http://example.com")
	response := httptest.NewRecorder()
	setupRouter().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	var result MulticlassSpellcasting
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.Pact == nil || result.Pact.Level != 3 || result.Slots[1] != 2 {
		t.Fatalf("response %+v, error %v", result, err)
	}
	for _, tt := range []struct {
		name   string
		user   string
		origin string
		body   string
		status int
	}{
		{"unauthenticated", "", "http://example.com", string(body), http.StatusUnauthorized},
		{"cross origin", "alice", "http://other.example", string(body), http.StatusForbidden},
		{"invalid levels", "alice", "http://example.com", `{"classes":[{"name":"Cyber-Mage","level":0}]}`, http.StatusBadRequest},
		{"trailing data", "alice", "http://example.com", string(body) + `{}`, http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/character-spellcasting", bytes.NewBufferString(tt.body))
			r.Header.Set("Remote-User", tt.user)
			r.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()
			setupRouter().ServeHTTP(w, r)
			if w.Code != tt.status {
				t.Fatalf("status %d, want %d: %s", w.Code, tt.status, w.Body.String())
			}
		})
	}
}
