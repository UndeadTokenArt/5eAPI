package main

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// CharacterNote is one read-only rules line shown for a chosen class, subclass,
// species or background.
type CharacterNote struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

type CharacterOption struct {
	Name   string          `json:"name"`
	Detail string          `json:"detail,omitempty"`
	Group  string          `json:"group,omitempty"`
	Link   string          `json:"link,omitempty"`
	Notes  []CharacterNote `json:"notes,omitempty"`

	Spellcasting *Spellcasting `json:"spellcasting,omitempty"`
	SpellLevel   *int          `json:"spell_level,omitempty"`
	SpellClasses []string      `json:"spell_classes,omitempty"`
}

type SubclassOption struct {
	Name  string          `json:"name"`
	Class string          `json:"class"`
	Link  string          `json:"link,omitempty"`
	Notes []CharacterNote `json:"notes,omitempty"`

	Spellcasting *Spellcasting `json:"spellcasting,omitempty"`

	// path locates the subclass in the Classes section tree.
	path []int
}

type CharacterOptions struct {
	Species     []CharacterOption `json:"species"`
	Classes     []CharacterOption `json:"classes"`
	Subclasses  []SubclassOption  `json:"subclasses"`
	Backgrounds []CharacterOption `json:"backgrounds"`
	Weapons     []CharacterOption `json:"weapons"`
	Spells      []CharacterOption `json:"spells"`
	Feats       []CharacterOption `json:"feats"`
	Equipment   []CharacterOption `json:"equipment"`
}

var (
	characterOptionsOnce  sync.Once
	characterOptionsCache CharacterOptions
	characterOptionsErr   error

	leadingNumberPattern = regexp.MustCompile(`^\d+\.\s*`)
)

func joinNonEmpty(sep string, parts ...string) string {
	var kept []string
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			kept = append(kept, strings.TrimSpace(part))
		}
	}
	return strings.Join(kept, sep)
}

func isSubclassGroup(name string) bool {
	return strings.Contains(strings.ToLower(name), "subclass")
}

// splitClasses separates base classes from the subclass groups that follow them
// in the guide. Numbered entries after a group are extra subclasses of that group.
func splitClasses(classes []Class) ([]string, []SubclassOption) {
	bases, _, subclasses := splitClassesIndexed(classes)
	return bases, subclasses
}

// splitClassesIndexed also returns each base class's index in the guide's class list.
func splitClassesIndexed(classes []Class) ([]string, []int, []SubclassOption) {
	var bases []string
	var baseIdx []int
	var subclasses []SubclassOption
	for ci, class := range classes {
		switch {
		case isSubclassGroup(class.Name):
			if len(bases) == 0 {
				continue
			}
			for fi, feature := range class.Features {
				subclasses = append(subclasses, SubclassOption{
					Name:  strings.TrimSpace(leadingNumberPattern.ReplaceAllString(feature.Name, "")),
					Class: bases[len(bases)-1],
					path:  []int{ci, fi},
				})
			}
		case leadingNumberPattern.MatchString(class.Name) && len(subclasses) > 0:
			subclasses = append(subclasses, SubclassOption{
				Name:  strings.TrimSpace(leadingNumberPattern.ReplaceAllString(class.Name, "")),
				Class: subclasses[len(subclasses)-1].Class,
				path:  []int{ci},
			})
		default:
			bases = append(bases, class.Name)
			baseIdx = append(baseIdx, ci)
		}
	}
	return bases, baseIdx, subclasses
}

const (
	noteTextLimit  = 700
	noteLimitPerOp = 40
)

var noteSpace = regexp.MustCompile(`\s+`)

func cleanNoteText(text string) string {
	text = strings.TrimSpace(noteSpace.ReplaceAllString(text, " "))
	if r := []rune(text); len(r) > noteTextLimit {
		text = strings.TrimSpace(string(r[:noteTextLimit])) + "…"
	}
	return text
}

// humanizeTrait turns trait ids such as "thermal_optic_vision_darkvision" into words.
func humanizeTrait(name string) string {
	if !strings.Contains(name, "_") {
		return name
	}
	words := strings.Fields(strings.ReplaceAll(name, "_", " "))
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// nodeNote renders a section and its nested features as one note line.
func nodeNote(node SectionNode) (CharacterNote, bool) {
	parts := []string{}
	if desc := strings.TrimSpace(node.Description); desc != "" {
		parts = append(parts, desc)
	}
	for _, child := range node.Children {
		text := strings.TrimSpace(child.Description)
		switch {
		case child.Title == "Mechanics" || child.Title == "Focus":
			parts = append(parts, text)
		case text != "":
			parts = append(parts, child.Title+": "+text)
		case child.Title != "":
			parts = append(parts, child.Title)
		}
	}
	text := cleanNoteText(strings.Join(parts, " "))
	if text == "" {
		return CharacterNote{}, false
	}
	return CharacterNote{Label: humanizeTrait(node.Title), Text: text}, true
}

func childNotes(node SectionNode) []CharacterNote {
	notes := []CharacterNote{}
	for _, child := range node.Children {
		if child.Generated {
			continue
		}
		if note, ok := nodeNote(child); ok && len(notes) < noteLimitPerOp {
			notes = append(notes, note)
		}
	}
	return notes
}

var backgroundNoteFields = []struct{ key, label string }{
	{"skill_proficiencies", "Skill proficiencies"},
	{"tool_language_proficiencies", "Tool / language proficiencies"},
	{"tool_proficiencies", "Tool proficiencies"},
	{"language_proficiencies", "Language proficiencies"},
	{"starting_equipment", "Starting equipment"},
	{"background_feature", "Background feature"},
}

func backgroundNotes(node SectionNode) []CharacterNote {
	notes := []CharacterNote{}
	for _, field := range backgroundNoteFields {
		if text, ok := node.Attributes[field.key].(string); ok && strings.TrimSpace(text) != "" {
			notes = append(notes, CharacterNote{Label: field.label, Text: cleanNoteText(text)})
		}
	}
	return notes
}

func nodeAt(node SectionNode, path []int) (SectionNode, bool) {
	for _, i := range path {
		if i < 0 || i >= len(node.Children) {
			return SectionNode{}, false
		}
		node = node.Children[i]
	}
	return node, true
}

// categoryNode maps one guide category and assigns the same section IDs the
// guide pages render, so options can deep-link to their detail entries.
func categoryNode(guide *StarterGuide, category string) (SectionNode, string) {
	node, ok := mapCategory(category, guide)
	if !ok {
		return SectionNode{}, ""
	}
	sections := []SectionNode{node}
	assignSectionIDs(sections, "")
	return sections[0], "/neotokyo/" + categorySlug(category) + "#"
}

func itemLink(base, id string) string {
	if base == "" || id == "" {
		return ""
	}
	return base + id
}

func buildCharacterOptions(guide *StarterGuide) CharacterOptions {
	c := guide.Category
	opts := CharacterOptions{
		Species: []CharacterOption{}, Classes: []CharacterOption{}, Subclasses: []SubclassOption{}, Backgrounds: []CharacterOption{},
		Weapons: []CharacterOption{}, Spells: []CharacterOption{}, Feats: []CharacterOption{}, Equipment: []CharacterOption{},
	}
	speciesNode, speciesBase := categoryNode(guide, "Species")
	for i, s := range c.Species {
		var notes []CharacterNote
		if i < len(speciesNode.Children) {
			notes = childNotes(speciesNode.Children[i])
		}
		opts.Species = append(opts.Species, CharacterOption{Name: s.Name, Link: itemLink(speciesBase, childID(speciesNode, i)), Notes: notes})
	}
	labels, _ := classLabels()
	spellLists, _ := loadClassSpellLists()
	classNode, classBase := categoryNode(guide, "Classes")
	bases, baseIdx, subclasses := splitClassesIndexed(c.Classes)
	for i, name := range bases {
		option := CharacterOption{Name: name, Link: itemLink(classBase, childID(classNode, baseIdx[i]))}
		if i < len(baseIdx) && baseIdx[i] < len(classNode.Children) {
			option.Notes = childNotes(classNode.Children[baseIdx[i]])
		}
		option.Spellcasting = spellcastingForClass(name, labels)
		opts.Classes = append(opts.Classes, option)
	}
	for _, sub := range subclasses {
		if node, ok := nodeAt(classNode, sub.path); ok {
			sub.Link = itemLink(classBase, node.ID)
			if desc := cleanNoteText(node.Description); desc != "" {
				sub.Notes = append(sub.Notes, CharacterNote{Label: "Features by level", Text: desc})
			}
			sub.Notes = append(sub.Notes, childNotes(node)...)
		}
		sub.Spellcasting = spellcastingForSubclass(sub.Name, labels)
		opts.Subclasses = append(opts.Subclasses, sub)
	}
	backgroundNode, backgroundBase := categoryNode(guide, "Backgrounds")
	for i, b := range c.Backgrounds {
		var notes []CharacterNote
		if i < len(backgroundNode.Children) {
			notes = backgroundNotes(backgroundNode.Children[i])
		}
		opts.Backgrounds = append(opts.Backgrounds, CharacterOption{Name: b.Name, Link: itemLink(backgroundBase, childID(backgroundNode, i)), Notes: notes})
	}

	node, base := categoryNode(guide, "Weapons")
	for i, w := range c.Weapons {
		opts.Weapons = append(opts.Weapons, CharacterOption{
			Name:   w.Name,
			Detail: joinNonEmpty(", ", joinNonEmpty(" ", w.Damage, w.DamageType), w.Attribute, w.Tag),
			Link:   itemLink(base, childID(node, i)),
		})
	}

	spellClasses := map[string][]string{}
	for original, entries := range spellLists.Lists {
		for _, entry := range entries {
			spellClasses[entry.Name] = append(spellClasses[entry.Name], labels[original])
		}
	}
	node, base = categoryNode(guide, "Spells")
	for i, sp := range c.Spells {
		option := CharacterOption{
			Name:   sp.Name,
			Detail: joinNonEmpty(", ", sp.Level, sp.School, sp.CastingTime, sp.Range),
			Link:   itemLink(base, childID(node, i)),
		}
		if level, err := strconv.Atoi(strings.TrimSpace(sp.Level)); err == nil {
			option.SpellLevel = &level
		}
		option.SpellClasses = spellClasses[sp.Name]
		sort.Strings(option.SpellClasses)
		opts.Spells = append(opts.Spells, option)
	}

	// Feats are nested: each guide entry is a themed group of individual feats.
	node, base = categoryNode(guide, "Feats")
	for i, group := range c.Feats {
		for j, feature := range group.Features {
			detail := ""
			if feature.Prerequisite != "" {
				detail = "Prerequisite: " + feature.Prerequisite
			}
			id := ""
			if i < len(node.Children) && j < len(node.Children[i].Children) {
				id = node.Children[i].Children[j].ID
			}
			opts.Feats = append(opts.Feats, CharacterOption{Name: feature.Name, Detail: detail, Group: group.Name, Link: itemLink(base, id)})
		}
	}

	groups := []struct {
		title string
		items []BasicItem
	}{
		{"Adventuring Gear & Containers", c.AdventuringGearAndContainers},
		{"Tools, Kits & Consumables", c.ToolsKitsAndConsumables},
		{"Ammunition", c.Ammunition},
		{"Armor & Shields", c.ArmorAndShields},
		{"Mounts, Vehicles & Harnesses", c.MountsVehiclesAndHarnesses},
		{"Foci & Instruments", c.FociAndInstruments},
	}
	for _, group := range groups {
		node, base := categoryNode(guide, group.title)
		for i, item := range group.items {
			opts.Equipment = append(opts.Equipment, CharacterOption{
				Name: item.Name, Detail: equipmentDetail(item), Group: group.title, Link: itemLink(base, childID(node, i)),
			})
		}
	}
	return opts
}

func childID(node SectionNode, index int) string {
	if index < len(node.Children) {
		return node.Children[index].ID
	}
	return ""
}

func characterOptionsHandler(w http.ResponseWriter, r *http.Request) {
	characterOptionsOnce.Do(func() {
		guide, err := loadJSON("static/data/neo_tokyo_starter_guide.json")
		if err != nil {
			characterOptionsErr = err
			return
		}
		characterOptionsCache = buildCharacterOptions(guide)
	})
	if characterOptionsErr != nil {
		writeCharacterError(w, http.StatusInternalServerError, "Could not load character options.")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeCharacterJSON(w, http.StatusOK, characterOptionsCache)
}

// equipmentDetail pairs an item's price with its description so the sheet shows both.
func equipmentDetail(item BasicItem) string {
	price := strings.TrimSpace(item.Price)
	if price != "" {
		price = "Price: " + price
	}
	detail := joinNonEmpty(" \u2014 ", price, cleanNoteText(item.Description))
	if r := []rune(detail); len(r) > 1500 {
		detail = string(r[:1500]) + "…"
	}
	return detail
}
