package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func NeoTokyoHandler(w http.ResponseWriter, r *http.Request) {
	data, err := loadJSON("static/data/neo_tokyo_starter_guide.json")
	if err != nil {
		http.Error(w, fmt.Sprintf("error loading JSON: %v", err), http.StatusInternalServerError)
		return
	}

	category := r.PathValue("category")
	if category == "" {
		category = categoryFromPath(r.URL.Path)
	}
	response := buildResponse(data, category)
	if category != "" && response.Total == 0 {
		http.NotFound(w, r)
		return
	}

	var rendered bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rendered, "NeoTokyo.tmpl", response); err != nil {
		http.Error(w, fmt.Sprintf("error rendering NeoTokyo template: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(rendered.Bytes())
}

func loadJSON(filename string) (*StarterGuide, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	var guide StarterGuide
	if err := json.Unmarshal(data, &guide); err != nil {
		return nil, fmt.Errorf("failed to decode json: %w", err)
	}

	return &guide, nil
}

func categoryFromPath(path string) string {
	cleanPath := strings.Trim(path, "/")
	cleanPath = strings.TrimPrefix(cleanPath, "neotokyo")
	return strings.Trim(cleanPath, "/")
}

func buildResponse(data *StarterGuide, category string) *StandardResponse {
	cleanCategory := strings.TrimSpace(category)
	var sections []SectionNode

	if cleanCategory == "" {
		for _, name := range []string{
			"Abilities", "Backgrounds", "Species", "Classes", "Feats",
			"Adventuring Gear & Containers", "Tools, Kits & Consumables", "Ammunition",
			"Weapons", "Armor & Shields", "Mounts, Vehicles & Harnesses", "Foci & Instruments",
			"Spells", "Conditions",
		} {
			if node, ok := mapCategory(name, data); ok {
				sections = append(sections, node)
			}
		}
	} else {
		node, ok := mapCategory(cleanCategory, data)
		if ok {
			sections = append(sections, node)
		}
	}

	title := "Neo Tokyo Starter Guide"
	if cleanCategory != "" && len(sections) > 0 {
		title = fmt.Sprintf("Neo Tokyo Starter Guide - %s", sections[0].Title)
	}

	return &StandardResponse{
		Category: title,
		Total:    len(sections),
		Sections: sections,
	}
}

// Master function routing mapping logic based on category type
func mapCategory(category string, data *StarterGuide) (SectionNode, bool) {
	switch normalizeCategory(category) {
	case "abilities":
		if len(data.Category.Abilities) == 0 {
			return SectionNode{}, false
		}
		return mapAbilities(data.Category.Abilities), true

	case "backgrounds":
		if len(data.Category.Backgrounds) == 0 {
			return SectionNode{}, false
		}
		return mapBackgrounds(data.Category.Backgrounds), true

	case "species":
		if len(data.Category.Species) == 0 {
			return SectionNode{}, false
		}
		return mapSpecies(data.Category.Species), true

	case "classes":
		if len(data.Category.Classes) == 0 {
			return SectionNode{}, false
		}
		return mapClasses(data.Category.Classes), true

	case "feats":
		if len(data.Category.Feats) == 0 {
			return SectionNode{}, false
		}
		return mapFeats(data.Category.Feats), true

	case "adventuring gear and containers":
		return mapBasicItems("Adventuring Gear & Containers", data.Category.AdventuringGearAndContainers)

	case "tools kits and consumables":
		return mapBasicItems("Tools, Kits & Consumables", data.Category.ToolsKitsAndConsumables)

	case "mounts vehicles and harnesses":
		return mapBasicItems("Mounts, Vehicles & Harnesses", data.Category.MountsVehiclesAndHarnesses)

	case "foci and instruments":
		return mapBasicItems("Foci & Instruments", data.Category.FociAndInstruments)

	case "weapons":
		if len(data.Category.Weapons) == 0 {
			return SectionNode{}, false
		}
		return mapWeapons(data.Category.Weapons), true

	case "spells":
		if len(data.Category.Spells) == 0 {
			return SectionNode{}, false
		}
		return mapSpells(data.Category.Spells), true

	case "conditions":
		if len(data.Category.Conditions) == 0 {
			return SectionNode{}, false
		}
		return mapConditions(data.Category.Conditions), true

	default:
		return SectionNode{}, false
	}
}

// --- Specific Mappers ---

func mapAbilities(abilities []Ability) SectionNode {
	var children []SectionNode
	for _, ability := range abilities {
		var featureNodes []SectionNode
		for _, feature := range ability.Features {
			featureNodes = append(featureNodes, SectionNode{
				Title:       feature.Name,
				Subtitle:    feature.Example,
				Description: feature.Description,
				Content:     feature.Content,
			})
		}
		children = append(children, SectionNode{
			Title:    ability.Name,
			Children: featureNodes,
		})
	}
	return SectionNode{Title: "Abilities", Children: children}
}

func normalizeCategory(category string) string {
	normalized := strings.ToLower(strings.TrimSpace(category))
	normalized = strings.NewReplacer("&", " and ", ",", " ", "-", " ", "_", " ").Replace(normalized)
	return strings.Join(strings.Fields(normalized), " ")
}

func mapBackgrounds(backgrounds []Background) SectionNode {
	var children []SectionNode
	for _, background := range backgrounds {
		attributes := map[string]any{
			"skill_proficiencies": background.SkillProficiencies,
			"starting_equipment":  background.StartingEquipment,
			"background_feature":  background.BackgroundFeature,
		}
		if background.ToolLanguageProficiencies != "" {
			attributes["tool_language_proficiencies"] = background.ToolLanguageProficiencies
		}
		if background.ToolProficiencies != "" {
			attributes["tool_proficiencies"] = background.ToolProficiencies
		}
		children = append(children, SectionNode{
			Title:       background.Name,
			Description: background.Description,
			Content:     background.Content,
			Attributes:  attributes,
		})
	}
	return SectionNode{Title: "Backgrounds", Children: children}
}

func mapSpecies(speciesList []Species) SectionNode {
	var children []SectionNode
	for _, species := range speciesList {
		var traitNodes []SectionNode
		for _, trait := range species.Trait {
			traitNodes = append(traitNodes, SectionNode{
				Title:       trait.Name,
				Description: trait.Description,
			})
		}
		children = append(children, SectionNode{
			Title: species.Name,
			Attributes: map[string]any{
				"origin":       species.Origin,
				"appearance":   species.Appearance,
				"cultural_fit": species.CulturalFit,
			},
			Children: traitNodes,
		})
	}
	return SectionNode{Title: "Species", Children: children}
}

func mapClasses(classes []Class) SectionNode {
	var children []SectionNode
	for _, c := range classes {
		var featNodes []SectionNode
		for _, f := range c.Features {
			var subFeatNodes []SectionNode
			for _, sf := range f.Features {
				subFeatNodes = append(subFeatNodes, SectionNode{
					Title:       sf.Name,
					Description: sf.Description,
				})
			}
			featNodes = append(featNodes, SectionNode{
				Title:       f.Name,
				Description: f.Description,
				Content:     f.Content,
				Children:    subFeatNodes,
			})
		}

		attrs := make(map[string]any)
		if c.Focus != "" {
			attrs["focus"] = c.Focus
		}
		if c.DefensiveFlourish != "" {
			attrs["defensive_flourish"] = c.DefensiveFlourish
		}
		if c.SlashingFlourish != "" {
			attrs["slashing_flourish"] = c.SlashingFlourish
		}
		if c.MobileFlourish != "" {
			attrs["mobile_flourish"] = c.MobileFlourish
		}

		children = append(children, SectionNode{
			Title:       c.Name,
			Description: c.Description,
			Content:     c.Content,
			Tags:        c.Details,
			Attributes:  attrs,
			Children:    featNodes,
		})
	}
	return SectionNode{Title: "Classes", Children: children}
}

func mapFeats(feats []Feat) SectionNode {
	var children []SectionNode
	for _, ft := range feats {
		var featNodes []SectionNode
		for _, f := range ft.Features {
			attrs := make(map[string]any)
			if f.Prerequisite != "" {
				attrs["prerequisite"] = f.Prerequisite
			}
			featNodes = append(featNodes, SectionNode{
				Title:       f.Name,
				Description: f.Description,
				Content:     f.Content,
				Tags:        f.Details,
				Attributes:  attrs,
			})
		}
		children = append(children, SectionNode{
			Title:    ft.Name,
			Children: featNodes,
		})
	}
	return SectionNode{Title: "Feats", Children: children}
}

func mapBasicItems(title string, items []BasicItem) (SectionNode, bool) {
	if len(items) == 0 {
		return SectionNode{}, false
	}
	var children []SectionNode
	for _, item := range items {
		children = append(children, SectionNode{
			Title:       item.Name,
			Description: item.Description,
			Attributes: map[string]any{
				"price": item.Price,
			},
		})
	}
	return SectionNode{Title: title, Children: children}, true
}

func mapWeapons(weapons []Weapon) SectionNode {
	var children []SectionNode
	for _, w := range weapons {
		attrs := map[string]any{
			"price":       w.Price,
			"damage":      w.Damage,
			"damage_type": w.DamageType,
			"attribute":   w.Attribute,
		}
		var tags []string
		if w.Tag != "" {
			tags = append(tags, w.Tag)
		}
		children = append(children, SectionNode{
			Title:       w.Name,
			Description: w.Description,
			Attributes:  attrs,
			Tags:        tags,
		})
	}
	return SectionNode{Title: "Weapons", Children: children}
}

func mapSpells(spells []Spell) SectionNode {
	var children []SectionNode
	for _, sp := range spells {
		children = append(children, SectionNode{
			Title:       sp.Name,
			Subtitle:    fmt.Sprintf("%s (%s)", sp.Level, sp.School),
			Description: sp.Description,
			Content:     sp.Mechanics,
			Tags:        sp.Components,
			Attributes: map[string]any{
				"casting_time": sp.CastingTime,
				"range":        sp.Range,
				"duration":     sp.Duration,
				"level":        sp.Level,
				"school":       sp.School,
			},
		})
	}
	return SectionNode{Title: "Spells", Children: children}
}

func mapConditions(conditions []Condition) SectionNode {
	var children []SectionNode
	for _, cd := range conditions {
		attrs := make(map[string]any)
		levels := []struct {
			val string
			key string
		}{
			{cd.OneLevel1, "level_1"},
			{cd.TwoLevel2, "level_2"},
			{cd.ThreeLevel3, "level_3"},
			{cd.FourLevel4, "level_4"},
			{cd.FiveLevel5, "level_5"},
			{cd.SixLevel6, "level_6"},
		}
		for _, lvl := range levels {
			if lvl.val != "" {
				attrs[lvl.key] = lvl.val
			}
		}

		children = append(children, SectionNode{
			Title:       cd.Name,
			Description: cd.Description,
			Content:     cd.Content,
			Tags:        cd.Details,
			Attributes:  attrs,
		})
	}
	return SectionNode{Title: "Conditions", Children: children}
}
