package encounter

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
)

const (
	dndAPI     = "https://www.dnd5eapi.co/api/2014"
	listenAddr = ":8080"
)

type APIList struct {
	Count   int    `json:"count"`
	Results []Stub `json:"results"`
}

type Stub struct {
	Index string `json:"index"`
	Name  string `json:"name"`
	URL   string `json:"url"`
	Level int    `json:"level,omitempty"`
}

type Property struct {
	Label   string
	Values  []string
	IsMulti bool
}

func fetchAPI(path string) ([]byte, int, error) {
	resp, err := http.Get(dndAPI + path) //nolint:gosec
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

// ─── Encounter Data ────────────────────────────────────────────────────────────

// xpThresholds[level-1] = {easy, medium, hard, deadly} XP per PC
var xpThresholds = [][4]int{
	{25, 50, 75, 100},
	{50, 100, 150, 200},
	{75, 150, 225, 400},
	{125, 250, 375, 500},
	{250, 500, 750, 1100},
	{300, 600, 900, 1400},
	{350, 750, 1100, 1700},
	{450, 900, 1400, 2100},
	{550, 1100, 1600, 2400},
	{600, 1200, 1900, 2800},
	{800, 1600, 2400, 3600},
	{1000, 2000, 3000, 4500},
	{1100, 2200, 3400, 5100},
	{1250, 2500, 3800, 5700},
	{1400, 2800, 4300, 6400},
	{1600, 3200, 4800, 7200},
	{2000, 3900, 5900, 8800},
	{2100, 4200, 6300, 9500},
	{2400, 4900, 7300, 10900},
	{2800, 5700, 8500, 12700},
}

var crXP = map[float64]int{
	0: 10, 0.125: 25, 0.25: 50, 0.5: 100,
	1: 200, 2: 450, 3: 700, 4: 1100, 5: 1800,
	6: 2300, 7: 2900, 8: 3900, 9: 5000, 10: 5900,
	11: 7200, 12: 8400, 13: 10000, 14: 11500, 15: 13000,
	16: 15000, 17: 18000, 18: 20000, 19: 22000, 20: 25000,
	21: 33000, 22: 41000, 23: 50000, 24: 62000, 30: 155000,
}

var lootByTier = [4][]string{
	{"Dagger", "Handaxe", "Shortbow", "Leather Armor", "Torch ×6", "Rope (50 ft)", "Healing Potion", "Rations (3 days)", "Coinpurse", "Copper signet ring"},
	{"Shortsword", "Longsword", "Scale Mail", "Light Crossbow", "Quiver of arrows", "Thieves' Tools", "Potion of Healing", "Scroll case", "Spyglass", "Silver locket"},
	{"Spell Scroll (1st level)", "Potion of Greater Healing", "Wyvern Poison (vial)", "Crystal Orb", "Adamantine ingot", "Navigator's Tools", "Sending Stone", "Chain Mail +1"},
	{"Spell Scroll (3rd level)", "Potion of Superior Healing", "Rare gemstone", "Arcane Focus (masterwork)", "Ornate Plate Armor", "Dragon Scale fragment", "Tome of Ancient Lore"},
}

// ─── Encounter Helpers ─────────────────────────────────────────────────────────

func formatCR(cr float64) string {
	switch cr {
	case 0.125:
		return "1/8"
	case 0.25:
		return "1/4"
	case 0.5:
		return "1/2"
	}
	return fmt.Sprintf("%g", cr)
}

func crXPFor(cr float64) int {
	if xp, ok := crXP[cr]; ok {
		return xp
	}
	if v, ok := crXP[float64(int(cr))]; ok {
		return v
	}
	return 100
}

func xpMultiplier(totalMonsters int) float64 {
	switch {
	case totalMonsters <= 1:
		return 1.0
	case totalMonsters == 2:
		return 1.5
	case totalMonsters <= 6:
		return 2.0
	case totalMonsters <= 10:
		return 2.5
	case totalMonsters <= 14:
		return 3.0
	default:
		return 4.0
	}
}

func crRangeForLevel(level int, difficulty string) (float64, float64) {
	base := float64(level) / 4.0
	switch difficulty {
	case "easy":
		return 0, base * 0.75
	case "medium":
		return base * 0.25, base * 1.25
	case "hard":
		return base * 0.5, base * 2.0
	default: // deadly
		return base * 0.75, base * 3.0
	}
}

func mundaneLootPool(maxCR float64) []string {
	switch {
	case maxCR < 5:
		return lootByTier[0]
	case maxCR < 11:
		return lootByTier[1]
	case maxCR < 17:
		return lootByTier[2]
	default:
		return lootByTier[3]
	}
}

func fetchRandomMagicItem() string {
	body, _, err := fetchAPI("/magic-items")
	if err != nil {
		return ""
	}
	var list APIList
	if err := json.Unmarshal(body, &list); err != nil || len(list.Results) == 0 {
		return ""
	}
	return list.Results[rand.Intn(len(list.Results))].Name
}

func generateLoot(maxCR float64, difficulty string, totalXP, partySize int) (int, []LootItem) {
	diffMult := map[string]float64{"easy": 0.5, "medium": 1.0, "hard": 1.5, "deadly": 2.5}[difficulty]

	baseGold := int(float64(totalXP) * 0.12 * diffMult)
	if baseGold < 5 {
		baseGold = 5 + rand.Intn(15)
	}
	if v := baseGold / 5; v > 0 {
		baseGold += rand.Intn(v*2+1) - v
	}
	if baseGold < 1 {
		baseGold = 1
	}

	pool := mundaneLootPool(maxCR)
	perm := rand.Perm(len(pool))
	count := 1 + rand.Intn(3)
	var loot []LootItem
	for i := 0; i < count && i < len(perm); i++ {
		loot = append(loot, LootItem{Icon: "🗡️", Name: pool[perm[i]], Type: "mundane"})
	}

	// Magic item roll — chance scales with CR tier and difficulty
	magicChance := 0.0
	switch {
	case maxCR < 1:
		magicChance = 0.05
	case maxCR < 5:
		magicChance = 0.15
	case maxCR < 11:
		magicChance = 0.35
	case maxCR < 17:
		magicChance = 0.55
	default:
		magicChance = 0.80
	}
	magicChance *= diffMult
	if magicChance > 0.95 {
		magicChance = 0.95
	}
	if rand.Float64() < magicChance {
		if name := fetchRandomMagicItem(); name != "" {
			loot = append(loot, LootItem{Icon: "🔮", Name: name, Type: "magic"})
		}
	}
	return baseGold, loot
}

func GenerateEncounter(partySize, partyLevel int, difficulty string) (*EncounterResult, error) {
	diffIdx := map[string]int{"easy": 0, "medium": 1, "hard": 2, "deadly": 3}[difficulty]
	budget := xpThresholds[partyLevel-1][diffIdx] * partySize

	body, _, err := fetchAPI("/monsters")
	if err != nil {
		return nil, fmt.Errorf("could not fetch monster list: %w", err)
	}
	var monsterList APIList
	if err := json.Unmarshal(body, &monsterList); err != nil {
		return nil, fmt.Errorf("could not parse monster list: %w", err)
	}

	all := monsterList.Results
	rand.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
	sampleSize := 32
	if len(all) < sampleSize {
		sampleSize = len(all)
	}
	sample := all[:sampleSize]

	// Fetch monster details in parallel
	type fetchResult struct {
		m   MonsterDetail
		err error
	}
	ch := make(chan fetchResult, len(sample))
	var wg sync.WaitGroup
	for _, s := range sample {
		wg.Add(1)
		go func(index string) {
			defer wg.Done()
			b, _, e := fetchAPI("/monsters/" + index)
			if e != nil {
				ch <- fetchResult{err: e}
				return
			}
			var m MonsterDetail
			if e2 := json.Unmarshal(b, &m); e2 != nil {
				ch <- fetchResult{err: e2}
				return
			}
			ch <- fetchResult{m: m}
		}(s.Index)
	}
	wg.Wait()
	close(ch)

	var candidates []MonsterDetail
	for r := range ch {
		if r.err == nil && r.m.Name != "" {
			candidates = append(candidates, r.m)
		}
	}

	// Filter by CR appropriate for the party level and difficulty
	minCR, maxCRLimit := crRangeForLevel(partyLevel, difficulty)
	var filtered []MonsterDetail
	for _, m := range candidates {
		if m.ChallengeRating >= minCR && m.ChallengeRating <= maxCRLimit {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) < 3 {
		filtered = candidates // widen to all sampled if too few pass the filter
	}

	// Sort by CR descending so the strongest monster is picked first (boss)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ChallengeRating > filtered[j].ChallengeRating
	})

	maxTypes := map[string]int{"easy": 2, "medium": 3, "hard": 4, "deadly": 5}[difficulty]
	var enc []EncounterMonster
	remaining := budget

	for _, m := range filtered {
		if len(enc) >= maxTypes || remaining <= 0 {
			break
		}
		xp := crXPFor(m.ChallengeRating)
		if xp == 0 {
			xp = 10
		}
		count := 1
		// Non-boss slots: occasionally add multiples of weaker monsters
		if len(enc) > 0 && xp < budget/4 && rand.Intn(2) == 0 {
			count = 2 + rand.Intn(2)
		}
		total := xp * count
		// Skip if this group blows the budget wildly over
		if total > remaining+(remaining/2) {
			count = 1
			total = xp
		}
		if total > budget*2 {
			continue
		}
		enc = append(enc, EncounterMonster{
			Detail:  m,
			Count:   count,
			XPEach:  xp,
			TotalXP: total,
			TotalHP: m.HitPoints * count,
			CRStr:   formatCR(m.ChallengeRating),
		})
		remaining -= total
	}

	if len(enc) == 0 && len(candidates) > 0 {
		m := candidates[0]
		xp := crXPFor(m.ChallengeRating)
		enc = append(enc, EncounterMonster{
			Detail: m, Count: 1, XPEach: xp, TotalXP: xp,
			TotalHP: m.HitPoints, CRStr: formatCR(m.ChallengeRating),
		})
	}

	// Shuffle so the boss isn't always displayed first
	rand.Shuffle(len(enc), func(i, j int) { enc[i], enc[j] = enc[j], enc[i] })

	totalRaw, totalCount, maxCR := 0, 0, 0.0
	for _, e := range enc {
		totalRaw += e.TotalXP
		totalCount += e.Count
		if e.Detail.ChallengeRating > maxCR {
			maxCR = e.Detail.ChallengeRating
		}
	}

	adjusted := int(float64(totalRaw) * xpMultiplier(totalCount))
	xpPerPC := 0
	if partySize > 0 {
		xpPerPC = totalRaw / partySize
	}
	gold, loot := generateLoot(maxCR, difficulty, totalRaw, partySize)
	goldPerPC := 0
	if partySize > 0 {
		goldPerPC = gold / partySize
	}

	diffColors := map[string]string{
		"easy": "#4ade80", "medium": "#facc15", "hard": "#fb923c", "deadly": "#f87171",
	}
	diffLabel := strings.ToUpper(difficulty[:1]) + difficulty[1:]

	return &EncounterResult{
		Monsters:   enc,
		Difficulty: diffLabel,
		DiffColor:  diffColors[difficulty],
		TotalXP:    totalRaw,
		AdjustedXP: adjusted,
		XPPerPC:    xpPerPC,
		Gold:       gold,
		GoldPerPC:  goldPerPC,
		Loot:       loot,
		PartySize:  partySize,
		PartyLevel: partyLevel,
		Budget:     budget,
	}, nil
}
