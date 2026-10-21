package main

import "testing"

func TestAegisEnforcerHasChannelDivinity(t *testing.T) {
	guide, err := loadGuideFromDir(builtinGuideDir)
	if err != nil {
		t.Fatal(err)
	}
	var found *CharacterOption
	opts := buildCharacterOptions(guide)
	for i := range opts.Classes {
		if opts.Classes[i].Name == "Aegis Corporate Enforcer / Vow-Bound Operative" {
			found = &opts.Classes[i]
		}
	}
	if found == nil || len(found.Resources) != 1 {
		t.Fatalf("expected one resource on the Aegis Enforcer, got %+v", found)
	}
	res := found.Resources[0]
	if res.ID != "channel-divinity" || res.MinLevel != 3 || len(res.Uses) != 20 || res.Uses[2] != 1 || res.Uses[1] != 0 {
		t.Fatalf("unexpected resource: %+v", res)
	}
	for _, c := range opts.Classes {
		if c.Name != found.Name && len(c.Resources) > 0 {
			t.Fatalf("unexpected resources on %s", c.Name)
		}
	}
}

func TestClassResourceValidation(t *testing.T) {
	c := CharacterSheet{Name: "A", Level: 3, AbilityScores: AbilityScores{10, 10, 10, 10, 10, 10}, ClassResources: []ResourceUse{{ID: "channel-divinity", Current: 1}}}
	if err := validateCharacter(c); err != nil {
		t.Fatal(err)
	}
	c.ClassResources[0].Current = 500
	if err := validateCharacter(c); err == nil {
		t.Fatal("expected out-of-range resource to be rejected")
	}
}
