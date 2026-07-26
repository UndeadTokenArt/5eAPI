package encounter

// ─── Encounter Types ───────────────────────────────────────────────────────────

type MonsterDetail struct {
	Index           string  `json:"index"`
	Name            string  `json:"name"`
	ChallengeRating float64 `json:"challenge_rating"`
	XP              int     `json:"xp"`
	HitPoints       int     `json:"hit_points"`
	Size            string  `json:"size"`
	Type            string  `json:"type"`
	Alignment       string  `json:"alignment"`
}

type EncounterMonster struct {
	Detail  MonsterDetail
	Count   int
	XPEach  int
	TotalXP int
	TotalHP int
	CRStr   string
}

type LootItem struct {
	Icon string
	Name string
	Type string // "mundane" or "magic"
}

type EncounterResult struct {
	Monsters   []EncounterMonster
	Difficulty string
	DiffColor  string
	TotalXP    int
	AdjustedXP int
	XPPerPC    int
	Gold       int
	GoldPerPC  int
	Loot       []LootItem
	PartySize  int
	PartyLevel int
	Budget     int
}
