package main

import (
	"encoding/json"
	"testing"
)

func TestClassLevelDataReachesGuideAndCharacterOptions(t *testing.T) {
	raw := []byte(`{"uid":"C001","name":"Example","features":[{"name":"Existing"}],"levels":[{"level":2,"hitdie":"1d6","skills":[{"name":"Feature","description":"Example description","features":[{"name":"Choice","description":"Choose one."}]}]}]}`)
	var class Class
	if err := json.Unmarshal(raw, &class); err != nil {
		t.Fatal(err)
	}
	if class.UID != "C001" || len(class.Levels) != 1 || class.Levels[0].Skills[0].Features[0].Description != "Choose one." {
		t.Fatalf("class level data lost: %+v", class)
	}
	guide := &StarterGuide{}
	guide.Category.Classes = []Class{class}
	options := buildCharacterOptions(guide)
	if len(options.Classes) != 1 || len(options.Classes[0].Levels) != 1 || options.Classes[0].Levels[0].HitDie != "1d6" {
		t.Fatalf("character options missing levels: %+v", options.Classes)
	}
	nodes := mapClasses(guide.Category.Classes).Children[0].Children
	if len(nodes) != 2 || nodes[0].Title != "Existing" || nodes[1].Title != "Level 2" || nodes[1].Children[0].Children[0].Title != "Choice" {
		t.Fatalf("guide nodes missing class levels: %+v", nodes)
	}
}

func TestInvalidPactSlotsRejectedWithoutOrdinarySlots(t *testing.T) {
	sheet := validCharacter("Example")
	sheet.PactSlots = &SpellSlot{Level: 1, Max: 1, Current: 2}
	if err := validateCharacter(sheet); err == nil {
		t.Fatal("invalid pact slots accepted when ordinary slots are empty")
	}
}

func TestClassHitDiceAvailableToCharacterOptions(t *testing.T) {
	guide, err := loadGuideFromDir(builtinGuideDir)
	if err != nil {
		t.Fatal(err)
	}
	options := buildCharacterOptions(guide)
	found := map[string]int{}
	for _, class := range options.Classes {
		found[class.Name] = class.HitDie
	}
	for name, die := range map[string]int{"Cyber-Mage": 6, "AI Host": 8, "Corporate Enforcer": 10} {
		if found[name] != die {
			t.Errorf("%s hit die = %d, want %d", name, found[name], die)
		}
	}
}
