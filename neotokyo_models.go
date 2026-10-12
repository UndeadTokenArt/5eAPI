package main

type RootJSON struct {
	Category CategoryData `json:"catagory"`
}

type CategoryData struct {
	Abilities                    []Ability    `json:"abilities,omitempty"`
	Backgrounds                  []Background `json:"backgrounds,omitempty"`
	Species                      []Species    `json:"species,omitempty"`
	Classes                      []Class      `json:"classes,omitempty"`
	Feats                        []Feat       `json:"feats,omitempty"`
	AdventuringGearAndContainers []BasicItem  `json:"adventuring gear and containers,omitempty"`
	ToolsKitsAndConsumables      []BasicItem  `json:"tools kits and consumables,omitempty"`
	Ammunition                   []BasicItem  `json:"ammunition,omitempty"`
	ArmorAndShields              []BasicItem  `json:"armor and shields,omitempty"`
	MountsVehiclesAndHarnesses   []BasicItem  `json:"mounts vehicles and harnesses,omitempty"`
	FociAndInstruments           []BasicItem  `json:"foci and instruments,omitempty"`
	Weapons                      []Weapon     `json:"weapons,omitempty"`
	Spells                       []Spell      `json:"spells,omitempty"`
	Conditions                   []Condition  `json:"conditions,omitempty"`
}

type StandardResponse struct {
	Category          string             `json:"category"`
	CurrentCategory   string             `json:"current_category,omitempty"`
	HasSpells         bool               `json:"has_spells,omitempty"`
	Total             int                `json:"total"`
	Categories        []CategoryOption   `json:"categories,omitempty"`
	SpellClassOptions []SpellClassOption `json:"spell_class_options,omitempty"`
	Sections          []SectionNode      `json:"sections"`
}

type CategoryOption struct {
	Title string `json:"title"`
	Path  string `json:"path"`
	Count int    `json:"count"`
}

type SpellClassOption struct {
	Original  string `json:"original"`
	Reskinned string `json:"reskinned"`
}

type SectionNode struct {
	ID           string         `json:"id,omitempty"`
	Title        string         `json:"title"`
	Subtitle     string         `json:"subtitle,omitempty"`
	Description  string         `json:"description,omitempty"`
	Content      string         `json:"content,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	Attributes   map[string]any `json:"attributes,omitempty"`
	Children     []SectionNode  `json:"children,omitempty"`
	SpellLevel   int            `json:"spell_level,omitempty"`
	SpellClasses []string       `json:"spell_classes,omitempty"`
	Tables       []SectionTable `json:"tables,omitempty"`
	Links        []SectionLink  `json:"links,omitempty"`

	// Generated marks nodes computed from rules data rather than written in the guide.
	Generated bool `json:"-"`
}

type SectionTable struct {
	Caption string     `json:"caption,omitempty"`
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

type SectionLink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type SpellReskinIndex struct {
	Spells       []SpellReskin `json:"spells_reskin"`
	ClassReskins []ClassReskin `json:"class_reskins"`
}

type ClassReskin struct {
	OriginalClass  string `json:"original_class"`
	ReskinnedClass string `json:"reskinned_class"`
}

type SpellReskin struct {
	Reskinned string   `json:"reskinned"`
	Level     int      `json:"level"`
	Classes   []string `json:"classes"`
}

type Ability struct {
	Name     string           `json:"name"`
	Features []AbilityFeature `json:"features,omitempty"`
}

type AbilityFeature struct {
	Name        string `json:"name"`
	Example     string `json:"example,omitempty"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
}

type Background struct {
	Name                      string `json:"name"`
	Description               string `json:"description,omitempty"`
	Content                   string `json:"content,omitempty"`
	SkillProficiencies        string `json:"skill_proficiencies,omitempty"`
	StartingEquipment         string `json:"starting_equipment,omitempty"`
	BackgroundFeature         string `json:"background_feature,omitempty"`
	ToolLanguageProficiencies string `json:"tool_language_proficiencies,omitempty"`
	ToolProficiencies         string `json:"tool_proficiencies,omitempty"`
}

type Species struct {
	Name        string  `json:"name"`
	Origin      string  `json:"origin,omitempty"`
	Appearance  string  `json:"appearance,omitempty"`
	CulturalFit string  `json:"cultural_fit,omitempty"`
	Trait       []Trait `json:"trait,omitempty"`
}

type Trait struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Class struct {
	Name              string         `json:"name"`
	Description       string         `json:"description,omitempty"`
	Content           string         `json:"content,omitempty"`
	Details           []string       `json:"details,omitempty"`
	Focus             string         `json:"focus,omitempty"`
	DefensiveFlourish string         `json:"defensive_flourish,omitempty"`
	SlashingFlourish  string         `json:"slashing_flourish,omitempty"`
	MobileFlourish    string         `json:"mobile_flourish,omitempty"`
	Features          []ClassFeature `json:"features,omitempty"`
}

type ClassFeature struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Content     string            `json:"content,omitempty"`
	Features    []SubClassFeature `json:"features,omitempty"`
}

type SubClassFeature struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Feat struct {
	Name     string        `json:"name"`
	Features []FeatFeature `json:"features,omitempty"`
}

type FeatFeature struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Content      string   `json:"content,omitempty"`
	Details      []string `json:"details,omitempty"`
	Prerequisite string   `json:"prerequisite,omitempty"`
}

type BasicItem struct {
	Name        string `json:"name"`
	Price       string `json:"price,omitempty"`
	Description string `json:"description,omitempty"`
}

type Weapon struct {
	Name        string `json:"name"`
	Price       string `json:"price,omitempty"`
	Damage      string `json:"damage,omitempty"`
	DamageType  string `json:"damage_type,omitempty"`
	Attribute   string `json:"attribute,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Description string `json:"description,omitempty"`
}

type Spell struct {
	Name        string   `json:"name"`
	Level       string   `json:"level,omitempty"`
	School      string   `json:"school,omitempty"`
	CastingTime string   `json:"casting_time,omitempty"`
	Range       string   `json:"range,omitempty"`
	Duration    string   `json:"duration,omitempty"`
	Mechanics   string   `json:"mechanics,omitempty"`
	Description string   `json:"description,omitempty"`
	Components  []string `json:"components,omitempty"`
}

type Condition struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Content     string   `json:"content,omitempty"`
	Details     []string `json:"details,omitempty"`
	OneLevel1   string   `json:"level_1,omitempty"`
	TwoLevel2   string   `json:"level_2,omitempty"`
	ThreeLevel3 string   `json:"level_3,omitempty"`
	FourLevel4  string   `json:"level_4,omitempty"`
	FiveLevel5  string   `json:"level_5,omitempty"`
	SixLevel6   string   `json:"level_6,omitempty"`
}
