package main

import "testing"

func TestNPCAttacksFromGuide(t *testing.T) {
	guide, err := loadGuideFromDir(builtinGuideDir)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, e := range guide.Category.NonPlayerEntities {
		for _, a := range e.attacksFor() {
			total++
			if a.Name == "" || len(a.Damage) == 0 || a.Damage[0].Dice == "" {
				t.Fatalf("incomplete attack on %s: %+v", e.Name, a)
			}
		}
	}
	if total == 0 {
		t.Fatal("expected at least one structured attack in the built-in entities")
	}
	attacks := npcAttacksByName("aegis barracuda drone")
	if len(attacks) != 2 || attacks[0].AttackBonus != 5 || attacks[0].Damage[0].Dice != "1d8+3" || attacks[0].Damage[0].Type != "Piercing" {
		t.Fatalf("unexpected attacks: %+v", attacks)
	}
}
