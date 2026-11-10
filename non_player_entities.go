package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// NonPlayerEntity is a monster/NPC stat block. Nested lists keep the SRD API's loose object shape.
type NonPlayerEntity struct {
	Name                  string           `json:"name"`
	Index                 string           `json:"index,omitempty"`
	Image                 string           `json:"image,omitempty"`
	URL                   string           `json:"url,omitempty"`
	UpdatedAt             string           `json:"updated_at,omitempty"`
	Size                  string           `json:"size,omitempty"`
	Type                  string           `json:"type,omitempty"`
	Subtype               string           `json:"subtype,omitempty"`
	Alignment             string           `json:"alignment,omitempty"`
	ArmorClass            []map[string]any `json:"armor_class,omitempty"`
	HitPoints             int              `json:"hit_points,omitempty"`
	HitDice               string           `json:"hit_dice,omitempty"`
	HitPointsRoll         string           `json:"hit_points_roll,omitempty"`
	Speed                 map[string]any   `json:"speed,omitempty"`
	Strength              int              `json:"strength,omitempty"`
	Dexterity             int              `json:"dexterity,omitempty"`
	Constitution          int              `json:"constitution,omitempty"`
	Intelligence          int              `json:"intelligence,omitempty"`
	Wisdom                int              `json:"wisdom,omitempty"`
	Charisma              int              `json:"charisma,omitempty"`
	Proficiencies         []map[string]any `json:"proficiencies,omitempty"`
	DamageVulnerabilities []any            `json:"damage_vulnerabilities,omitempty"`
	DamageResistances     []any            `json:"damage_resistances,omitempty"`
	DamageImmunities      []any            `json:"damage_immunities,omitempty"`
	ConditionImmunities   []any            `json:"condition_immunities,omitempty"`
	Senses                map[string]any   `json:"senses,omitempty"`
	Languages             string           `json:"languages,omitempty"`
	ChallengeRating       float64          `json:"challenge_rating,omitempty"`
	ProficiencyBonus      int              `json:"proficiency_bonus,omitempty"`
	XP                    int              `json:"xp,omitempty"`
	SpecialAbilities      []map[string]any `json:"special_abilities,omitempty"`
	Actions               []map[string]any `json:"actions,omitempty"`
	Reactions             []map[string]any `json:"reactions,omitempty"`
	LegendaryActions      []map[string]any `json:"legendary_actions,omitempty"`
	Forms                 []any            `json:"forms,omitempty"`
}

func joinedMap(values map[string]any) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, strings.ReplaceAll(key, "_", " ")+" "+fmt.Sprint(values[key]))
	}
	return strings.Join(parts, ", ")
}

func namedList(values []any) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if object, ok := value.(map[string]any); ok {
			if name, ok := object["name"].(string); ok {
				parts = append(parts, name)
				continue
			}
		}
		parts = append(parts, fmt.Sprint(value))
	}
	return strings.Join(parts, ", ")
}

func describedEntries(entries []map[string]any) string {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		name, _ := entry["name"].(string)
		desc, _ := entry["desc"].(string)
		parts = append(parts, strings.TrimSpace(name+". "+desc))
	}
	return strings.Join(parts, "\n")
}

func mapNonPlayerEntities(entities []NonPlayerEntity) SectionNode {
	var children []SectionNode
	for _, e := range entities {
		subtitle := strings.TrimSpace(strings.Join(nonEmpty(e.Size, e.Type, e.Alignment), " "))
		attrs := map[string]any{}
		set := func(key string, value any) {
			if text := fmt.Sprint(value); text != "" && text != "0" {
				attrs[key] = value
			}
		}
		var armor []string
		for _, ac := range e.ArmorClass {
			text := fmt.Sprint(ac["value"])
			if kind, ok := ac["type"].(string); ok && kind != "" {
				text += " (" + kind + ")"
			}
			armor = append(armor, text)
		}
		set("armor_class", strings.Join(armor, ", "))
		hp := ""
		if e.HitPoints > 0 {
			hp = strconv.Itoa(e.HitPoints)
			if e.HitPointsRoll != "" {
				hp += " (" + e.HitPointsRoll + ")"
			}
		}
		set("hit_points", hp)
		set("speed", joinedMap(e.Speed))
		if e.Strength+e.Dexterity+e.Constitution+e.Intelligence+e.Wisdom+e.Charisma > 0 {
			attrs["ability_scores"] = fmt.Sprintf("STR %d, DEX %d, CON %d, INT %d, WIS %d, CHA %d",
				e.Strength, e.Dexterity, e.Constitution, e.Intelligence, e.Wisdom, e.Charisma)
		}
		set("damage_vulnerabilities", namedList(e.DamageVulnerabilities))
		set("damage_resistances", namedList(e.DamageResistances))
		set("damage_immunities", namedList(e.DamageImmunities))
		set("condition_immunities", namedList(e.ConditionImmunities))
		set("senses", joinedMap(e.Senses))
		set("languages", e.Languages)
		set("challenge_rating", strconv.FormatFloat(e.ChallengeRating, 'f', -1, 64))
		set("proficiency_bonus", e.ProficiencyBonus)
		set("xp", e.XP)

		node := SectionNode{Title: e.Name, Image: e.Image, Subtitle: subtitle, Attributes: attrs}
		for _, section := range []struct {
			label   string
			entries []map[string]any
		}{
			{"Special abilities", e.SpecialAbilities}, {"Actions", e.Actions},
			{"Reactions", e.Reactions}, {"Legendary actions", e.LegendaryActions},
		} {
			if len(section.entries) > 0 {
				node.CustomFields = append(node.CustomFields, SectionField{
					Label: section.label, Value: describedEntries(section.entries), Display: "paragraph", Size: "normal",
				})
			}
		}
		children = append(children, node)
	}
	return SectionNode{Title: "Non-Player Entities", Children: children}
}

func nonEmpty(values ...string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

// NPCAttack is one structured attack action of a stat block, ready for one-click rolling.
type NPCAttack struct {
	Name        string      `json:"name"`
	AttackBonus int         `json:"attack_bonus"`
	Damage      []NPCDamage `json:"damage"`
	Description string      `json:"description,omitempty"`
}

type NPCDamage struct {
	Dice string `json:"dice"`
	Type string `json:"type,omitempty"`
}

// attacksFor lists the actions that have both an attack bonus and damage dice.
func (e NonPlayerEntity) attacksFor() []NPCAttack {
	attacks := []NPCAttack{}
	for _, action := range e.Actions {
		name, _ := action["name"].(string)
		bonus, hasBonus := action["attack_bonus"].(float64)
		damageList, _ := action["damage"].([]any)
		if name == "" || !hasBonus || len(damageList) == 0 {
			continue
		}
		attack := NPCAttack{Name: name, AttackBonus: int(bonus), Damage: []NPCDamage{}}
		attack.Description, _ = action["desc"].(string)
		for _, raw := range damageList {
			entry, _ := raw.(map[string]any)
			dice, _ := entry["damage_dice"].(string)
			if strings.TrimSpace(dice) == "" {
				continue
			}
			damage := NPCDamage{Dice: strings.TrimSpace(dice)}
			if kind, ok := entry["damage_type"].(map[string]any); ok {
				damage.Type, _ = kind["name"].(string)
			}
			attack.Damage = append(attack.Damage, damage)
		}
		if len(attack.Damage) > 0 {
			attacks = append(attacks, attack)
		}
	}
	return attacks
}

// npcAttacksByName finds a stat block by name (case-insensitive) and returns its attacks.
func npcAttacksByName(name string) []NPCAttack {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	guide, _, err := currentGuide()
	if err != nil {
		return nil
	}
	for _, e := range guide.Category.NonPlayerEntities {
		if strings.EqualFold(e.Name, name) {
			return e.attacksFor()
		}
	}
	return nil
}

type npcCatalogEntry struct {
	Name      string  `json:"name"`
	HP        int     `json:"hit_points"`
	AC        int     `json:"armor_class,omitempty"`
	CR        float64 `json:"challenge_rating"`
	Summary   string  `json:"summary,omitempty"`
	Dexterity int     `json:"dexterity,omitempty"`
}

// npcCatalogHandler lists the guide's non-player entities for the campaign "add" picker.
func npcCatalogHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := campaignAuth(w, r); !ok {
		return
	}
	guide, _, err := currentGuide()
	if err != nil {
		writeCharacterError(w, http.StatusInternalServerError, "Could not load the non-player entity list.")
		return
	}
	entries := make([]npcCatalogEntry, 0, len(guide.Category.NonPlayerEntities))
	for _, e := range guide.Category.NonPlayerEntities {
		entry := npcCatalogEntry{Name: e.Name, HP: e.HitPoints, CR: e.ChallengeRating, Dexterity: e.Dexterity,
			Summary: strings.Join(nonEmpty(e.Size, e.Type), " ")}
		if len(e.ArmorClass) > 0 {
			if value, ok := e.ArmorClass[0]["value"].(float64); ok {
				entry.AC = int(value)
			}
		}
		entries = append(entries, entry)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}

func lootForCR(guide *StarterGuide, cr float64) ([]LootItem, int, error) {
	if cr < 0 || cr > 30 {
		return nil, 0, fmt.Errorf("%w: CR must be between 0 and 30", errCampaignInvalid)
	}
	names := make([]string, 0)
	addNames := func(items []BasicItem) {
		for _, item := range items {
			if strings.TrimSpace(item.Name) != "" {
				names = append(names, item.Name)
			}
		}
	}
	addNames(guide.Category.AdventuringGearAndContainers)
	addNames(guide.Category.ToolsKitsAndConsumables)
	addNames(guide.Category.Ammunition)
	addNames(guide.Category.ArmorAndShields)
	addNames(guide.Category.MountsVehiclesAndHarnesses)
	addNames(guide.Category.FociAndInstruments)
	for _, weapon := range guide.Category.Weapons {
		if strings.TrimSpace(weapon.Name) != "" {
			names = append(names, weapon.Name)
		}
	}
	if len(names) == 0 {
		return nil, 0, fmt.Errorf("%w: the guide has no equipment or weapons for loot generation", errCampaignInvalid)
	}
	itemCount, yenMultiplier := 1+rand.Intn(2), 10
	switch {
	case cr >= 17:
		itemCount, yenMultiplier = 3+rand.Intn(3), 250
	case cr >= 11:
		itemCount, yenMultiplier = 2+rand.Intn(3), 100
	case cr >= 5:
		itemCount, yenMultiplier = 1+rand.Intn(3), 50
	}
	items := make([]LootItem, 0, itemCount)
	for i := 0; i < itemCount; i++ {
		name := names[rand.Intn(len(names))]
		_ = addLootItems(&items, []LootItem{{Name: name, Quantity: 1}})
	}
	return items, (rand.Intn(20) + 1) * yenMultiplier, nil
}
