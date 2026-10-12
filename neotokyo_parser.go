package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"unicode"
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
	if response.HasSpells {
		spellIndex, err := loadSpellReskinIndex("static/data/cyberpunk-class-and-spell-reskins.json")
		if err != nil {
			http.Error(w, fmt.Sprintf("error loading spell class mappings: %v", err), http.StatusInternalServerError)
			return
		}
		applySpellReskinMetadata(response.Sections, spellIndex)
		response.SpellClassOptions = spellClassOptions(response.Sections, spellIndex)
	}

	var rendered bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rendered, "NeoTokyo.tmpl", response); err != nil {
		http.Error(w, fmt.Sprintf("error rendering NeoTokyo template: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(rendered.Bytes())
}

type spellReskinLookup struct {
	Spells      map[string]SpellReskin
	ClassLabels map[string]string
}

func loadSpellReskinIndex(filename string) (spellReskinLookup, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return spellReskinLookup{}, fmt.Errorf("failed to open spell class mappings: %w", err)
	}
	var index SpellReskinIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return spellReskinLookup{}, fmt.Errorf("failed to decode spell class mappings: %w", err)
	}
	lookup := spellReskinLookup{
		Spells:      make(map[string]SpellReskin, len(index.Spells)),
		ClassLabels: make(map[string]string, len(index.ClassReskins)),
	}
	for _, spell := range index.Spells {
		lookup.Spells[spell.Reskinned] = spell
	}
	for _, class := range index.ClassReskins {
		lookup.ClassLabels[class.OriginalClass] = class.ReskinnedClass
	}
	return lookup, nil
}

func applySpellReskinMetadata(sections []SectionNode, index spellReskinLookup) {
	for sectionIndex := range sections {
		for spellIndex := range sections[sectionIndex].Children {
			spell := &sections[sectionIndex].Children[spellIndex]
			mapping, ok := index.Spells[spell.Title]
			if !ok {
				continue
			}
			spell.SpellLevel = mapping.Level
			spell.SpellClasses = mapping.Classes
		}
	}
}

func spellClassOptions(sections []SectionNode, index spellReskinLookup) []SpellClassOption {
	labelsByOriginal := make(map[string]string)
	for _, section := range sections {
		for _, spell := range section.Children {
			mapping, ok := index.Spells[spell.Title]
			if !ok {
				continue
			}
			for _, className := range mapping.Classes {
				label := index.ClassLabels[className]
				if label == "" {
					label = className
				}
				labelsByOriginal[className] = label
			}
		}
	}
	options := make([]SpellClassOption, 0, len(labelsByOriginal))
	for original, reskinned := range labelsByOriginal {
		options = append(options, SpellClassOption{Original: original, Reskinned: reskinned})
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].Reskinned < options[j].Reskinned
	})
	return options
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
	var categories []CategoryOption

	if cleanCategory == "" {
		for _, name := range neoTokyoCategoryNames {
			if node, ok := mapCategory(name, data); ok {
				categories = append(categories, CategoryOption{
					Title: name,
					Path:  "/neotokyo/" + categorySlug(name),
					Count: len(node.Children),
				})
			}
		}
	} else {
		node, ok := mapCategory(cleanCategory, data)
		if ok {
			sections = append(sections, node)
		}
	}
	assignSectionIDs(sections, "")
	hasSpells := cleanCategory != "" && len(sections) > 0 && sections[0].Title == "Spells"

	title := "Neo Tokyo Starter Guide"
	currentCategory := ""
	if cleanCategory != "" && len(sections) > 0 {
		currentCategory = sections[0].Title
		title = fmt.Sprintf("Neo Tokyo Starter Guide - %s", currentCategory)
	}

	return &StandardResponse{
		Category:        title,
		CurrentCategory: currentCategory,
		HasSpells:       hasSpells,
		Total:           len(sections),
		Categories:      categories,
		Sections:        sections,
	}
}

var neoTokyoCategoryNames = []string{
	"Abilities", "Backgrounds", "Species", "Classes", "Feats",
	"Adventuring Gear & Containers", "Tools, Kits & Consumables", "Ammunition",
	"Weapons", "Armor & Shields", "Mounts, Vehicles & Harnesses", "Foci & Instruments",
	"Spells", "Conditions",
}

func categorySlug(category string) string {
	normalized := normalizeCategory(category)
	return strings.ReplaceAll(normalized, " ", "-")
}

func assignSectionIDs(sections []SectionNode, parentID string) {
	for index := range sections {
		id := slugifySectionTitle(sections[index].Title)
		if parentID != "" {
			id = parentID + "-" + id
		}
		sections[index].ID = fmt.Sprintf("%s-%d", id, index+1)
		assignSectionIDs(sections[index].Children, sections[index].ID)
	}
}

func slugifySectionTitle(title string) string {
	var slug strings.Builder
	lastWasSeparator := false
	for _, character := range strings.ToLower(title) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			slug.WriteRune(character)
			lastWasSeparator = false
		} else if !lastWasSeparator && slug.Len() > 0 {
			slug.WriteByte('-')
			lastWasSeparator = true
		}
	}
	return strings.Trim(slug.String(), "-")
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
		node := mapClasses(data.Category.Classes)
		addSpellcastingNodes(&node, data)
		return node, true

	case "feats":
		if len(data.Category.Feats) == 0 {
			return SectionNode{}, false
		}
		return mapFeats(data.Category.Feats), true

	case "adventuring gear and containers":
		return mapBasicItems("Adventuring Gear & Containers", data.Category.AdventuringGearAndContainers)

	case "tools kits and consumables":
		return mapBasicItems("Tools, Kits & Consumables", data.Category.ToolsKitsAndConsumables)

	case "ammunition":
		return mapBasicItems("Ammunition", data.Category.Ammunition)

	case "armor and shields":
		return mapBasicItems("Armor & Shields", data.Category.ArmorAndShields)

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
