package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPrerequisites2024(t *testing.T) {
	t.Setenv("NEOTOKYO_DATA_DIR", builtinGuideDir)
	labels, err := classLabels()
	if err != nil {
		t.Fatal(err)
	}
	scores := AbilityScores{Strength: 10, Dexterity: 13, Constitution: 10, Intelligence: 10, Wisdom: 13, Charisma: 13}
	tests := []struct {
		name    string
		current []CharacterClass
		next    string
		want    []string
	}{
		{"qualifies", []CharacterClass{{Name: labels["Fighter"], Level: 1}}, labels["Sorcerer"], []string{}},
		{"both monk requirements", []CharacterClass{{Name: labels["Monk"], Level: 1}}, labels["Sorcerer"], []string{}},
		{"missing class primary", []CharacterClass{{Name: labels["Paladin"], Level: 1}}, labels["Wizard"], []string{labels["Paladin"], labels["Wizard"]}},
		{"all existing classes", []CharacterClass{{Name: labels["Wizard"], Level: 1}, {Name: labels["Monk"], Level: 1}}, labels["Warlock"], []string{labels["Wizard"]}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkMulticlassEligibility2024(multiclassEligibilityRequest{Classes: tt.current, NewClass: tt.next, AbilityScores: scores}, labels)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Unqualified, tt.want) || got.Eligible != (len(tt.want) == 0) {
				t.Fatalf("got %+v, want %v", got, tt.want)
			}
		})
	}
	for _, req := range []multiclassEligibilityRequest{
		{Classes: []CharacterClass{{Name: labels["Wizard"], Level: 20}}, NewClass: labels["Sorcerer"], AbilityScores: scores},
		{Classes: []CharacterClass{{Name: labels["Wizard"], Level: 1}}, NewClass: labels["Wizard"], AbilityScores: scores},
		{Classes: []CharacterClass{{Name: labels["Wizard"], Level: 1}}, NewClass: "Missing", AbilityScores: scores},
	} {
		if _, err := checkMulticlassEligibility2024(req, labels); err == nil {
			t.Fatalf("invalid request accepted: %+v", req)
		}
	}
}

func TestAll2024WorksheetsHavePrerequisites(t *testing.T) {
	t.Setenv("NEOTOKYO_DATA_DIR", builtinGuideDir)
	for class := range progressionFiles2024 {
		if _, err := loadProgression2024(class); err != nil {
			t.Errorf("%s: %v", class, err)
		}
	}
}

func TestEligibility2024Endpoint(t *testing.T) {
	t.Setenv("NEOTOKYO_DATA_DIR", builtinGuideDir)
	t.Setenv("CHARACTER_GUEST_ACCESS", "")
	body := `{"classes":[{"name":"Awakened Mutant","level":1}],"new_class":"Cyber-Mage","ability_scores":{"strength":10,"dexterity":10,"constitution":10,"intelligence":12,"wisdom":10,"charisma":13}}`
	req := httptest.NewRequest(http.MethodPost, "/api/multiclass-eligibility", bytes.NewBufferString(body))
	req.Header.Set("Remote-User", "alice")
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	setupRouter().ServeHTTP(rec, req)
	var got multiclassEligibilityResult
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &got) != nil || got.Eligible || !reflect.DeepEqual(got.Unqualified, []string{"Cyber-Mage"}) {
		t.Fatalf("status %d result %+v: %s", rec.Code, got, rec.Body.String())
	}
}

func TestInvalidPrerequisitesAreRejected(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NEOTOKYO_DATA_DIR", dir)
	path := filepath.Join(dir, "classesv2", "Awakened Mutant.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"original_class":"Sorcerer","multiclass_prerequisites":{"any_of":null},"levels":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadProgression2024("Sorcerer"); err == nil {
		t.Fatal("invalid prerequisite worksheet accepted")
	}
}
