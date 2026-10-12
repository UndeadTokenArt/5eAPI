package main

import "testing"

func TestSpellcastingTables(t *testing.T) {
	labels, err := classLabels()
	if err != nil {
		t.Fatal(err)
	}
	wizard := spellcastingForClass(labels["Wizard"], labels)
	if wizard == nil {
		t.Fatal("wizard label should resolve to a caster")
	}
	if got := wizard.Rows[4].Slots[:3]; got[0] != 4 || got[1] != 3 || got[2] != 2 {
		t.Errorf("wizard level 5 slots = %v", got)
	}
	if wizard.Rows[0].Cantrips != 3 || wizard.Rows[3].Cantrips != 4 || wizard.Rows[9].Cantrips != 5 {
		t.Errorf("wizard cantrips wrong: %v %v %v", wizard.Rows[0].Cantrips, wizard.Rows[3].Cantrips, wizard.Rows[9].Cantrips)
	}
	if got := wizard.Rows[19].Slots; got[8] != 1 || got[4] != 3 {
		t.Errorf("wizard level 20 slots = %v", got)
	}
	warlock := spellcastingForClass(labels["Warlock"], labels)
	if row := warlock.Rows[4]; row.PactSlots != 2 || row.PactLevel != 3 || row.Slots[2] != 2 {
		t.Errorf("warlock level 5 = %+v", row)
	}
	if warlock.Rows[10].Slots[5] != 1 {
		t.Errorf("warlock level 11 should gain a 6th-level arcanum")
	}
	paladin := spellcastingForClass(labels["Paladin"], labels)
	if paladin.Rows[0].MaxSpellLevel != 0 || paladin.Rows[1].Slots[0] != 2 {
		t.Errorf("paladin progression wrong: %+v %+v", paladin.Rows[0], paladin.Rows[1])
	}
	if spellcastingForClass(labels["Fighter"], labels) != nil {
		t.Error("fighter should not be a caster itself")
	}
	third := spellcastingForSubclass("Cyber-Mage Ronin", labels)
	if third == nil || third.Rows[1].MaxSpellLevel != 0 || third.Rows[2].Slots[0] != 2 {
		t.Errorf("third caster progression wrong: %+v", third)
	}
}

func TestGuideClassesListSpells(t *testing.T) {
	guide, err := loadJSON("static/data/neo_tokyo_starter_guide.json")
	if err != nil {
		t.Fatal(err)
	}
	classes, base := categoryNode(guide, "Classes")
	labels, _ := classLabels()
	found := false
	for _, class := range classes.Children {
		if class.Title != labels["Wizard"] {
			continue
		}
		for _, child := range class.Children {
			if child.Title != "Class Spell List" {
				continue
			}
			found = true
			if len(child.Children) == 0 || len(child.Children[0].Links) == 0 || child.Children[0].Links[0].URL == "" {
				t.Fatalf("spell list groups need links: %+v", child.Children)
			}
		}
	}
	if !found || base == "" {
		t.Fatal("wizard class should have a Class Spell List node")
	}
	opts := buildCharacterOptions(guide)
	for _, sub := range opts.Subclasses {
		if sub.Name == "Cyber-Mage Ronin" && sub.Spellcasting == nil {
			t.Error("third-caster subclass should expose spellcasting")
		}
	}
	for _, sp := range opts.Spells {
		if sp.SpellLevel == nil || len(sp.SpellClasses) == 0 {
			continue
		}
		return
	}
	t.Error("spell options should carry level and classes")
}

func TestValidateSpellSlots(t *testing.T) {
	sheet := CharacterSheet{SpellSlots: []SpellSlot{{Level: 10, Max: 1}}}
	if err := validateCharacter(sheet); err == nil {
		t.Error("slot level 10 should be rejected")
	}
}
