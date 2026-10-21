package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// SpellcastingRow is what a character of one class level can do with spells.
type SpellcastingRow struct {
	Level         int   `json:"level"`
	Cantrips      int   `json:"cantrips"`
	Known         int   `json:"known,omitempty"`
	Slots         []int `json:"slots"`
	PactSlots     int   `json:"pact_slots,omitempty"`
	PactLevel     int   `json:"pact_level,omitempty"`
	MaxSpellLevel int   `json:"max_spell_level"`
}

// Spellcasting describes a class's (or subclass's) spell progression.
type Spellcasting struct {
	Ability       string            `json:"ability"`
	Kind          string            `json:"kind"`
	PreparedMode  string            `json:"prepared_mode,omitempty"`
	Summary       string            `json:"summary"`
	SpellListFrom string            `json:"spell_list_from"`
	Rows          []SpellcastingRow `json:"rows"`
}

type spellProfile struct {
	ability      string
	kind         string
	cantripBase  int
	cantripStart int
	known        []int
	preparedMode string
	summary      string
	slots        [][]int
}

var (
	fullCasterSlots = [][]int{
		{2}, {3}, {4, 2}, {4, 3}, {4, 3, 2}, {4, 3, 3}, {4, 3, 3, 1}, {4, 3, 3, 2}, {4, 3, 3, 3, 1}, {4, 3, 3, 3, 2},
		{4, 3, 3, 3, 2, 1}, {4, 3, 3, 3, 2, 1}, {4, 3, 3, 3, 2, 1, 1}, {4, 3, 3, 3, 2, 1, 1}, {4, 3, 3, 3, 2, 1, 1, 1}, {4, 3, 3, 3, 2, 1, 1, 1},
		{4, 3, 3, 3, 2, 1, 1, 1, 1}, {4, 3, 3, 3, 3, 1, 1, 1, 1}, {4, 3, 3, 3, 3, 2, 1, 1, 1}, {4, 3, 3, 3, 3, 2, 2, 1, 1},
	}
	halfCasterSlots = [][]int{
		nil, {2}, {3}, {3}, {4, 2}, {4, 2}, {4, 3}, {4, 3}, {4, 3, 2}, {4, 3, 2},
		{4, 3, 3}, {4, 3, 3}, {4, 3, 3, 1}, {4, 3, 3, 1}, {4, 3, 3, 2}, {4, 3, 3, 2}, {4, 3, 3, 3, 1}, {4, 3, 3, 3, 1}, {4, 3, 3, 3, 2}, {4, 3, 3, 3, 2},
	}
	thirdCasterSlots = [][]int{
		nil, nil, {2}, {3}, {3}, {3}, {4, 2}, {4, 2}, {4, 2}, {4, 3},
		{4, 3}, {4, 3}, {4, 3, 2}, {4, 3, 2}, {4, 3, 2}, {4, 3, 3}, {4, 3, 3}, {4, 3, 3}, {4, 3, 3, 1}, {4, 3, 3, 1},
	}
	// Warlock pact slots: count and spell level for each class level.
	warlockPactCount = []int{1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4}
	warlockPactLevel = []int{1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}
	// Warlock Mystic Arcanum: class level at which each 6th-9th level spell unlocks.
	warlockArcanum = map[int]int{11: 6, 13: 7, 15: 8, 17: 9}

	bardKnown     = []int{4, 5, 6, 7, 8, 9, 10, 11, 12, 14, 15, 15, 16, 18, 19, 19, 20, 22, 22, 22}
	sorcererKnown = []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 12, 13, 13, 14, 14, 15, 15, 15, 15}
	warlockKnown  = []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 10, 11, 11, 12, 12, 13, 13, 14, 14, 15, 15}
	rangerKnown   = []int{0, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10, 11, 11}
	thirdKnown    = []int{0, 0, 3, 4, 4, 4, 5, 6, 6, 7, 8, 8, 9, 10, 10, 11, 11, 11, 12, 13}

	spellProfiles = map[string]spellProfile{
		"Bard":     {ability: "charisma", kind: "full", cantripBase: 2, cantripStart: 1, known: bardKnown, slots: fullCasterSlots, summary: "Spells known are fixed per level; swap one when you gain a level."},
		"Cleric":   {ability: "wisdom", kind: "full", cantripBase: 3, cantripStart: 1, preparedMode: "full_level", slots: fullCasterSlots, summary: "Prepare Wisdom modifier + class level spells (minimum 1) after each long rest."},
		"Druid":    {ability: "wisdom", kind: "full", cantripBase: 2, cantripStart: 1, preparedMode: "full_level", slots: fullCasterSlots, summary: "Prepare Wisdom modifier + class level spells (minimum 1) after each long rest."},
		"Sorcerer": {ability: "charisma", kind: "full", cantripBase: 4, cantripStart: 1, known: sorcererKnown, slots: fullCasterSlots, summary: "Spells known are fixed per level; swap one when you gain a level."},
		"Wizard":   {ability: "intelligence", kind: "full", cantripBase: 3, cantripStart: 1, preparedMode: "full_level", slots: fullCasterSlots, summary: "Prepare Intelligence modifier + class level spells (minimum 1) from your spellbook, which holds 6 spells at level 1 and gains 2 per level."},
		"Paladin":  {ability: "charisma", kind: "half", preparedMode: "half_level", slots: halfCasterSlots, summary: "Prepare Charisma modifier + half your class level (rounded down, minimum 1) spells after each long rest. Spellcasting starts at level 2."},
		"Ranger":   {ability: "wisdom", kind: "half", known: rangerKnown, slots: halfCasterSlots, summary: "Spells known are fixed per level. Spellcasting starts at level 2."},
		"Warlock":  {ability: "charisma", kind: "pact", cantripBase: 2, cantripStart: 1, known: warlockKnown, summary: "Pact Magic: slots are all the same level and refresh on a short rest. From level 11 you also gain one Mystic Arcanum spell each of 6th-9th level, cast once per long rest."},
	}

	// Fighter and Rogue subclasses that cast with the wizard list (Eldritch Knight / Arcane Trickster).
	thirdCasterSubclasses = map[string]string{
		"Cyber-Mage Ronin":                        "Fighter",
		"The Decker (Matrix Specialist / Hacker)": "Rogue",
	}
	thirdCasterProfile = spellProfile{ability: "intelligence", kind: "third", cantripBase: 2, cantripStart: 3, known: thirdKnown, slots: thirdCasterSlots,
		summary: "Spellcasting starts at class level 3 and uses the Wizard spell list. Spells known are fixed per level; cantrips rise from 2 to 3 at level 10."}
)

func ordinal(n int) string {
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	}
	return strconv.Itoa(n) + "th"
}

func buildSpellcasting(profile spellProfile, spellListFrom string) *Spellcasting {
	sc := &Spellcasting{Ability: profile.ability, Kind: profile.kind, PreparedMode: profile.preparedMode, Summary: profile.summary, SpellListFrom: spellListFrom}
	for level := 1; level <= 20; level++ {
		row := SpellcastingRow{Level: level, Slots: make([]int, 9)}
		if profile.cantripBase > 0 && level >= profile.cantripStart {
			row.Cantrips = profile.cantripBase
			if level >= 4 && profile.kind != "third" {
				row.Cantrips++
			}
			if level >= 10 {
				row.Cantrips++
			}
		}
		if profile.known != nil {
			row.Known = profile.known[level-1]
		}
		if profile.kind == "pact" {
			row.PactSlots = warlockPactCount[level-1]
			row.PactLevel = warlockPactLevel[level-1]
			row.Slots[row.PactLevel-1] = row.PactSlots
			for unlock, spellLevel := range warlockArcanum {
				if level >= unlock {
					row.Slots[spellLevel-1] = 1
				}
			}
		} else {
			copy(row.Slots, profile.slots[level-1])
		}
		for spellLevel := 9; spellLevel >= 1; spellLevel-- {
			if row.Slots[spellLevel-1] > 0 {
				row.MaxSpellLevel = spellLevel
				break
			}
		}
		sc.Rows = append(sc.Rows, row)
	}
	return sc
}

type classSpellEntry struct {
	Name  string `json:"cyberpunk_reskin"`
	Level int    `json:"level"`
}

type classSpellData struct {
	Lists map[string][]classSpellEntry `json:"class_spell_lists"`
}

var classSpellCache struct {
	sync.Mutex
	version string
	data    classSpellData
	ok      bool
}

func loadClassSpellLists() (classSpellData, error) {
	path := referencePath("class-spell-lists.json")
	info, err := os.Stat(path)
	if err != nil {
		return classSpellData{}, err
	}
	version := fmt.Sprintf("%d|%d", info.Size(), info.ModTime().UnixNano())
	classSpellCache.Lock()
	defer classSpellCache.Unlock()
	if classSpellCache.ok && classSpellCache.version == version {
		return classSpellCache.data, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return classSpellData{}, err
	}
	var data classSpellData
	if err := json.Unmarshal(raw, &data); err != nil {
		return classSpellData{}, err
	}
	classSpellCache.version, classSpellCache.data, classSpellCache.ok = version, data, true
	return data, nil
}

// classLabels maps an original class name (Wizard) to its Neo Tokyo name.
func classLabels() (map[string]string, error) {
	index, err := loadSpellReskinIndex(referencePath("class-and-spell-reskins.json"))
	if err != nil {
		return nil, err
	}
	return index.ClassLabels, nil
}

func originalClassFor(label string, labels map[string]string) string {
	for original, reskinned := range labels {
		if reskinned == label {
			return original
		}
	}
	return ""
}

// spellcastingForClass returns the progression for a base class, or nil for non-casters.
func spellcastingForClass(label string, labels map[string]string) *Spellcasting {
	original := originalClassFor(label, labels)
	profile, ok := spellProfiles[original]
	if !ok {
		return nil
	}
	return buildSpellcasting(profile, labels[original])
}

func spellcastingForSubclass(name string, labels map[string]string) *Spellcasting {
	if _, ok := thirdCasterSubclasses[name]; !ok {
		return nil
	}
	return buildSpellcasting(thirdCasterProfile, labels["Wizard"])
}

func spellColumn(row SpellcastingRow, level int) string {
	if row.Slots[level-1] == 0 {
		return "—"
	}
	return strconv.Itoa(row.Slots[level-1])
}

func progressionTable(sc *Spellcasting) SectionTable {
	table := SectionTable{Caption: fmt.Sprintf("Spellcasting ability: %s. %s", strings.Title(sc.Ability), sc.Summary)} //nolint:staticcheck
	known := sc.Rows[19].Known > 0
	if sc.Kind == "pact" {
		table.Headers = []string{"Level", "Cantrips", "Spells Known", "Pact Slots", "Slot Level", "Mystic Arcanum"}
		for _, row := range sc.Rows {
			arcanum := []string{}
			for spellLevel := 6; spellLevel <= 9; spellLevel++ {
				if row.Slots[spellLevel-1] > 0 {
					arcanum = append(arcanum, ordinal(spellLevel))
				}
			}
			table.Rows = append(table.Rows, []string{
				strconv.Itoa(row.Level), strconv.Itoa(row.Cantrips), strconv.Itoa(row.Known),
				strconv.Itoa(row.PactSlots), ordinal(row.PactLevel), dashIfEmpty(strings.Join(arcanum, ", ")),
			})
		}
		return table
	}
	maxLevel := sc.Rows[19].MaxSpellLevel
	table.Headers = []string{"Level", "Cantrips"}
	if known {
		table.Headers = append(table.Headers, "Spells Known")
	}
	for spellLevel := 1; spellLevel <= maxLevel; spellLevel++ {
		table.Headers = append(table.Headers, ordinal(spellLevel))
	}
	for _, row := range sc.Rows {
		if row.MaxSpellLevel == 0 && row.Cantrips == 0 {
			continue
		}
		cells := []string{strconv.Itoa(row.Level), strconv.Itoa(row.Cantrips)}
		if known {
			cells = append(cells, strconv.Itoa(row.Known))
		}
		for spellLevel := 1; spellLevel <= maxLevel; spellLevel++ {
			cells = append(cells, spellColumn(row, spellLevel))
		}
		table.Rows = append(table.Rows, cells)
	}
	return table
}

func dashIfEmpty(value string) string {
	if value == "" {
		return "—"
	}
	return value
}

// firstLevelForSpellLevel is the class level at which a spell level first becomes available.
func firstLevelForSpellLevel(sc *Spellcasting, spellLevel int) int {
	for _, row := range sc.Rows {
		if spellLevel == 0 && row.Cantrips > 0 {
			return row.Level
		}
		if spellLevel > 0 && row.Slots[spellLevel-1] > 0 {
			return row.Level
		}
	}
	return 0
}

func spellLevelTitle(level int) string {
	if level == 0 {
		return "Cantrips"
	}
	return ordinal(level) + "-Level Spells"
}

// spellcastingNodes builds the "Spell Progression" and "Class Spell List" nodes for a class.
func spellcastingNodes(sc *Spellcasting, spells []classSpellEntry, spellLinks map[string]string) []SectionNode {
	progression := SectionNode{
		Title:       "Spell Progression by Class Level",
		Description: "Spell slots, cantrips and spells for each class level.",
		Tables:      []SectionTable{progressionTable(sc)},
		Generated:   true,
	}
	byLevel := map[int][]SectionLink{}
	for _, spell := range spells {
		byLevel[spell.Level] = append(byLevel[spell.Level], SectionLink{Title: spell.Name, URL: spellLinks[spell.Name]})
	}
	var groups []SectionNode
	for level := 0; level <= 9; level++ {
		links := byLevel[level]
		available := firstLevelForSpellLevel(sc, level)
		if len(links) == 0 || available == 0 {
			continue
		}
		sort.Slice(links, func(i, j int) bool { return links[i].Title < links[j].Title })
		groups = append(groups, SectionNode{
			Title:       spellLevelTitle(level),
			Description: fmt.Sprintf("Available from class level %d. %d spells.", available, len(links)),
			Links:       links,
			Generated:   true,
		})
	}
	list := SectionNode{
		Title:       "Class Spell List",
		Description: "Every spell this class can learn or prepare, grouped by spell level. Each group unlocks at the class level shown.",
		Children:    groups,
		Generated:   true,
	}
	return []SectionNode{progression, list}
}

// addSpellcastingNodes appends spell progression and spell list nodes to each casting class
// (and to the Fighter/Rogue subclasses that cast) in the Classes section.
func addSpellcastingNodes(classes *SectionNode, guide *StarterGuide) {
	labels, err := classLabels()
	if err != nil {
		return
	}
	lists, err := loadClassSpellLists()
	if err != nil {
		return
	}
	spellNode, base := categoryNode(guide, "Spells")
	spellLinks := make(map[string]string, len(spellNode.Children))
	for _, child := range spellNode.Children {
		spellLinks[child.Title] = base + child.ID
	}

	for i, class := range guide.Category.Classes {
		if i >= len(classes.Children) {
			break
		}
		child := &classes.Children[i]
		if sc := spellcastingForClass(class.Name, labels); sc != nil {
			original := originalClassFor(class.Name, labels)
			child.Children = append(child.Children, spellcastingNodes(sc, lists.Lists[original], spellLinks)...)
		}
		for _, group := range child.Children {
			if group.Title != subclassesTitle {
				continue
			}
			for j, subclass := range class.Subclasses {
				sc := spellcastingForSubclass(subclass.Name, labels)
				if sc == nil || j >= len(group.Children) {
					continue
				}
				node := &group.Children[j]
				node.Children = append(node.Children, spellcastingNodes(sc, lists.Lists["Wizard"], spellLinks)...)
			}
		}
	}
}
