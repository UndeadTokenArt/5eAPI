package main

import "testing"

func TestAegisEnforcerHasBioNaniteInjector(t *testing.T) {
	guide, err := loadGuideFromDir(builtinGuideDir)
	if err != nil {
		t.Fatal(err)
	}
	var found *CharacterOption
	opts := buildCharacterOptions(guide)
	for i := range opts.Classes {
		if opts.Classes[i].Name == "Vow-Bound Operative" {
			found = &opts.Classes[i]
		}
	}
	if found == nil || len(found.Resources) != 1 {
		t.Fatalf("expected one resource on the Aegis Enforcer, got %+v", found)
	}
	res := found.Resources[0]
	if res.ID != "bio-nanite-injector" || res.MinLevel != 0 || len(res.Uses) != 20 || res.Uses[0] != 5 || res.Uses[19] != 100 {
		t.Fatalf("unexpected resource: %+v", res)
	}
	for _, c := range opts.Classes {
		if c.Name != found.Name && c.Name != "Awakened Mutant" && len(c.Resources) > 0 {
			t.Fatalf("unexpected resources on %s", c.Name)
		}
	}
}

func TestClassResourceValidation(t *testing.T) {
	c := CharacterSheet{Name: "A", Level: 3, AbilityScores: AbilityScores{10, 10, 10, 10, 10, 10}, ClassResources: []ResourceUse{{ID: "bio-nanite-injector", Current: 1}}}
	if err := validateCharacter(c); err != nil {
		t.Fatal(err)
	}
	c.ClassResources[0].Current = 500
	if err := validateCharacter(c); err == nil {
		t.Fatal("expected out-of-range resource to be rejected")
	}
}

func TestAwakenedMutantHasMutationPoints(t *testing.T) {
	data, err := loadClassResources()
	if err != nil {
		t.Fatal(err)
	}
	list := data.Resources["Sorcerer"]
	if len(list) != 2 || list[0].ID != "innate-sorcery" || list[0].Uses[0] != 2 || list[0].Uses[19] != 2 ||
		list[1].ID != "mutation-points" || list[1].MinLevel != 2 || list[1].Uses[1] != 2 || list[1].Uses[19] != 20 {
		t.Fatalf("mutation points = %+v", list)
	}
}
