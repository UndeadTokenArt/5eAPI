package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	dndAPI     = "https://www.dnd5eapi.co/api/2014"
	listenAddr = ":8080"
)

// ─── Models ────────────────────────────────────────────────────────────────────

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

// ─── Helpers ───────────────────────────────────────────────────────────────────

func prettyCat(s string) string {
	words := strings.Split(s, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

var catIcons = map[string]string{
	"ability-scores":       "💪",
	"alignments":           "⚖️",
	"backgrounds":          "📜",
	"classes":              "⚔️",
	"conditions":           "🩹",
	"damage-types":         "🔥",
	"equipment":            "🛡️",
	"equipment-categories": "📦",
	"feats":                "⭐",
	"features":             "✨",
	"languages":            "🗣️",
	"magic-items":          "🔮",
	"magic-schools":        "🧙",
	"monsters":             "👹",
	"proficiencies":        "🎯",
	"races":                "🧝",
	"rule-sections":        "📋",
	"rules":                "📖",
	"skills":               "🎲",
	"spells":               "✨",
	"subclasses":           "🗡️",
	"subraces":             "🌟",
	"traits":               "🏷️",
	"weapon-properties":    "⚔️",
}

func iconFor(cat string) string {
	if icon, ok := catIcons[cat]; ok {
		return icon
	}
	return "📌"
}

// skipInProps lists fields rendered separately or that are internal.
var skipInProps = map[string]bool{
	"index": true, "url": true, "name": true,
	"desc": true, "higher_level": true,
}

// extractProps pulls simple displayable properties from an API item map.
func extractProps(data map[string]interface{}) []Property {
	keys := make([]string, 0, len(data))
	for k := range data {
		if !skipInProps[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var props []Property
	for _, k := range keys {
		v := data[k]
		label := prettyCat(k)

		switch vt := v.(type) {
		case string:
			if vt != "" {
				props = append(props, Property{Label: label, Values: []string{vt}})
			}
		case float64:
			props = append(props, Property{Label: label, Values: []string{fmt.Sprintf("%g", vt)}})
		case bool:
			val := "No"
			if vt {
				val = "Yes"
			}
			props = append(props, Property{Label: label, Values: []string{val}})
		case map[string]interface{}:
			if n, ok := vt["name"].(string); ok {
				props = append(props, Property{Label: label, Values: []string{n}})
			}
		case []interface{}:
			var strs []string
			valid := true
			for _, item := range vt {
				switch it := item.(type) {
				case string:
					strs = append(strs, it)
				case map[string]interface{}:
					if n, ok := it["name"].(string); ok {
						strs = append(strs, n)
					} else {
						valid = false
					}
				case float64:
					strs = append(strs, fmt.Sprintf("%g", it))
				default:
					valid = false
				}
				if !valid {
					break
				}
			}
			if valid && len(strs) > 0 {
				props = append(props, Property{
					Label:   label,
					Values:  strs,
					IsMulti: len(strs) > 1,
				})
			}
		}
	}
	return props
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

func generateEncounter(partySize, partyLevel int, difficulty string) (*EncounterResult, error) {
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

// ─── Encounter Handler ─────────────────────────────────────────────────────────

func encounterHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	sizeStr := q.Get("size")
	levelStr := q.Get("level")
	diffStr := q.Get("difficulty")

	partySize, _ := strconv.Atoi(sizeStr)
	partyLevel, _ := strconv.Atoi(levelStr)
	difficulty := diffStr

	if partySize < 1 || partySize > 8 {
		partySize = 4
	}
	if partyLevel < 1 || partyLevel > 20 {
		partyLevel = 5
	}
	validDiff := map[string]bool{"easy": true, "medium": true, "hard": true, "deadly": true}
	if !validDiff[difficulty] {
		difficulty = "medium"
	}

	generate := sizeStr != "" || levelStr != "" || diffStr != ""

	data := map[string]interface{}{
		"PartySize":       partySize,
		"PartyLevel":      partyLevel,
		"Difficulty":      difficulty,
		"Error":           "",
		"ShowPlaceholder": !generate,
	}

	if generate {
		result, err := generateEncounter(partySize, partyLevel, difficulty)
		if err != nil {
			data["Error"] = err.Error()
		} else {
			data["Result"] = result
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "encounter", data); err != nil {
		log.Println("template error:", err)
	}
}

// ─── API Client ────────────────────────────────────────────────────────────────

func fetchAPI(path string) ([]byte, int, error) {
	resp, err := http.Get(dndAPI + path) //nolint:gosec
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

// ─── Template setup ────────────────────────────────────────────────────────────

var funcMap = template.FuncMap{
	"prettyCat": prettyCat,
	"iconFor":   iconFor,
}

var tmpl = template.Must(template.New("").Funcs(funcMap).Parse(pageTemplates))

// ─── Handlers ──────────────────────────────────────────────────────────────────

func homeHandler(w http.ResponseWriter, r *http.Request) {
	body, _, err := fetchAPI("")
	if err != nil {
		http.Error(w, "Could not reach D&D 5e API: "+err.Error(), http.StatusBadGateway)
		return
	}
	var cats map[string]string
	if err := json.Unmarshal(body, &cats); err != nil {
		http.Error(w, "Unexpected API response", http.StatusInternalServerError)
		return
	}
	names := make([]string, 0, len(cats))
	for k := range cats {
		names = append(names, k)
	}
	sort.Strings(names)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "home", names); err != nil {
		log.Println("template error:", err)
	}
}

func categoryHandler(w http.ResponseWriter, r *http.Request, category string) {
	body, status, err := fetchAPI("/" + category)
	if err != nil {
		http.Error(w, "Could not reach D&D 5e API: "+err.Error(), http.StatusBadGateway)
		return
	}
	if status == http.StatusNotFound {
		http.NotFound(w, r)
		return
	}
	var list APIList
	if err := json.Unmarshal(body, &list); err != nil {
		http.Error(w, "Unexpected API response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "category", map[string]interface{}{
		"Category": category,
		"Count":    list.Count,
		"Results":  list.Results,
	}); err != nil {
		log.Println("template error:", err)
	}
}

func itemHandler(w http.ResponseWriter, r *http.Request, category, slug string) {
	body, status, err := fetchAPI("/" + category + "/" + slug)
	if err != nil {
		http.Error(w, "Could not reach D&D 5e API: "+err.Error(), http.StatusBadGateway)
		return
	}
	if status == http.StatusNotFound {
		http.NotFound(w, r)
		return
	}
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(w, "Unexpected API response", http.StatusInternalServerError)
		return
	}

	name, _ := data["name"].(string)
	if name == "" {
		name = prettyCat(slug)
	}

	var descs []string
	if d, ok := data["desc"]; ok {
		switch dt := d.(type) {
		case []interface{}:
			for _, line := range dt {
				if s, ok2 := line.(string); ok2 {
					descs = append(descs, s)
				}
			}
		case string:
			descs = []string{dt}
		}
	}

	var higherLevel []string
	if hl, ok := data["higher_level"]; ok {
		if hla, ok := hl.([]interface{}); ok {
			for _, line := range hla {
				if s, ok2 := line.(string); ok2 {
					higherLevel = append(higherLevel, s)
				}
			}
		}
	}

	pretty, _ := json.MarshalIndent(data, "", "  ")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "item", map[string]interface{}{
		"Category":    category,
		"Slug":        slug,
		"Name":        name,
		"Desc":        descs,
		"HigherLevel": higherLevel,
		"Props":       extractProps(data),
		"JSON":        string(pretty),
	}); err != nil {
		log.Println("template error:", err)
	}
}

func router(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.Trim(r.URL.Path, "/")
	if path == "" {
		homeHandler(w, r)
		return
	}
	parts := strings.SplitN(path, "/", 2)
	if path == "api/map/biomes" {
		mapBiomesHandler(w, r)
		return
	}
	if parts[0] == "encounter" {
		encounterHandler(w, r)
		return
	}
	if parts[0] == "map" {
		if len(parts) == 2 && parts[1] == "editor" {
			mapEditorHandler(w, r)
			return
		}
		mapHandler(w, r)
		return
	}
	switch len(parts) {
	case 1:
		categoryHandler(w, r, parts[0])
	case 2:
		itemHandler(w, r, parts[0], parts[1])
	default:
		http.NotFound(w, r)
	}
}

func main() {
	http.Handle("/tiles/", http.StripPrefix("/tiles/", http.FileServer(http.Dir("tiles"))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", router)
	log.Printf("⚔  D&D 5e SRD Interface → http://localhost%s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

// ─── Templates ─────────────────────────────────────────────────────────────────

const pageTemplates = `
{{define "css"}}
<style>
:root {
  --bg:    #0e0c0a;
  --sf:    #1c1917;
  --sf2:   #252220;
  --bd:    #3d2b1f;
  --gold:  #c9a63c;
  --gold2: #8a6d2a;
  --red:   #9b1c1c;
  --text:  #e8dcc8;
  --muted: #9b8c7a;
  --link:  #e8a23a;
}
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body {
  background: var(--bg);
  color: var(--text);
  font-family: 'Segoe UI', system-ui, -apple-system, sans-serif;
  min-height: 100vh;
  line-height: 1.6;
}
a { color: var(--link); text-decoration: none; }
a:hover { text-decoration: underline; }

/* ── Nav ── */
nav {
  background: var(--sf);
  border-bottom: 2px solid var(--gold2);
  padding: .85rem 2rem;
  display: flex;
  align-items: center;
  gap: 1rem;
}
.brand {
  font-size: 1.3rem;
  font-weight: 700;
  color: var(--gold);
  letter-spacing: .05em;
  text-transform: uppercase;
}
.brand:hover { text-decoration: none; color: var(--link); }
.nav-sub { color: var(--muted); font-size: .8rem; }
.nav-enc {
  margin-left: auto;
  color: var(--gold);
  font-size: .85rem;
  font-weight: 600;
  border: 1px solid var(--gold2);
  border-radius: 4px;
  padding: .3rem .75rem;
  white-space: nowrap;
}
.nav-enc:hover { background: var(--sf2); text-decoration: none; }

/* ── Layout ── */
main { max-width: 1200px; margin: 0 auto; padding: 2rem; }

/* ── Footer ── */
footer {
  background: var(--sf);
  border-top: 1px solid var(--bd);
  padding: 1.5rem 2rem;
  text-align: center;
  color: var(--muted);
  font-size: .85rem;
  margin-top: 3rem;
}
footer a { color: var(--link); }

/* ── Home ── */
.hero { text-align: center; padding: 3rem 0 2rem; }
.hero h1 {
  font-size: 2.4rem;
  color: var(--gold);
  font-family: Georgia, 'Times New Roman', serif;
  margin-bottom: .5rem;
  text-shadow: 0 2px 8px rgba(0,0,0,.6);
}
.hero p { color: var(--muted); font-size: 1.05rem; }

.cat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
  gap: 1rem;
  margin-top: 2rem;
}
.cat-card {
  background: var(--sf);
  border: 1px solid var(--bd);
  border-radius: 8px;
  padding: 1.5rem 1rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: .5rem;
  color: var(--text);
  transition: border-color .2s, background .2s, transform .2s, box-shadow .2s;
}
.cat-card:hover {
  border-color: var(--gold);
  background: var(--sf2);
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(201,166,60,.15);
  text-decoration: none;
  color: var(--gold);
}
.cat-icon { font-size: 2.2rem; }
.cat-name { font-size: .85rem; text-align: center; font-weight: 500; }

/* ── Breadcrumb ── */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: .5rem;
  padding: .75rem 0 1.25rem;
  font-size: .88rem;
  color: var(--muted);
}
.breadcrumb a { color: var(--link); }

/* ── Category page ── */
.page-heading {
  font-size: 2rem;
  color: var(--gold);
  font-family: Georgia, serif;
  margin-bottom: .5rem;
}
.count-badge {
  background: var(--red);
  color: #fff;
  padding: .2rem .7rem;
  border-radius: 20px;
  font-size: .8rem;
  font-weight: 600;
  vertical-align: middle;
  margin-left: .5rem;
}
.search-box {
  display: block;
  width: 100%;
  padding: .7rem 1rem;
  background: var(--sf);
  border: 1px solid var(--bd);
  border-radius: 6px;
  color: var(--text);
  font-size: 1rem;
  margin: 1rem 0 1.5rem;
  outline: none;
}
.search-box:focus { border-color: var(--gold); }
.search-box::placeholder { color: var(--muted); }

.item-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: .5rem;
}
.item-link {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--sf);
  border: 1px solid var(--bd);
  border-radius: 6px;
  padding: .65rem 1rem;
  color: var(--text);
  font-size: .9rem;
  transition: border-color .15s, background .15s, color .15s;
}
.item-link:hover {
  border-color: var(--gold2);
  background: var(--sf2);
  color: var(--gold);
  text-decoration: none;
}
.lvl-badge {
  font-size: .7rem;
  background: var(--gold2);
  color: var(--bg);
  padding: .1rem .4rem;
  border-radius: 4px;
  font-weight: 700;
  flex-shrink: 0;
}
.no-results { text-align: center; color: var(--muted); padding: 3rem; grid-column: 1 / -1; }

/* ── Item page ── */
.item-title {
  font-size: 2.3rem;
  color: var(--gold);
  font-family: Georgia, serif;
  margin-bottom: .2rem;
  line-height: 1.2;
}
.item-cat { color: var(--muted); font-size: .9rem; margin-bottom: 1.25rem; }
.api-link {
  display: inline-block;
  margin-bottom: 1.5rem;
  font-size: .8rem;
  color: var(--muted);
  border: 1px solid var(--bd);
  border-radius: 4px;
  padding: .2rem .6rem;
}
.api-link:hover { color: var(--link); }

.section-title {
  font-size: .82rem;
  color: var(--gold);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: .07em;
  margin-bottom: .75rem;
}
.description {
  background: var(--sf);
  border-left: 3px solid var(--red);
  padding: 1rem 1.25rem;
  border-radius: 0 6px 6px 0;
  margin-bottom: 1.5rem;
  font-size: .95rem;
}
.description p + p { margin-top: .7rem; }

.higher-level {
  background: var(--sf);
  border-left: 3px solid var(--gold2);
  padding: 1rem 1.25rem;
  border-radius: 0 6px 6px 0;
  margin-bottom: 1.5rem;
  font-size: .9rem;
}
.higher-level-title {
  font-size: .82rem;
  color: var(--gold);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: .07em;
  margin-bottom: .5rem;
}

.props-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: .75rem;
  margin-bottom: 1.5rem;
}
.prop {
  background: var(--sf);
  border: 1px solid var(--bd);
  border-radius: 6px;
  padding: .75rem 1rem;
}
.prop-label {
  font-size: .75rem;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: .05em;
  margin-bottom: .3rem;
}
.prop-value { color: var(--text); font-size: .92rem; font-weight: 500; }
.prop-list { list-style: none; padding: 0; }
.prop-list li { color: var(--text); font-size: .88rem; line-height: 1.7; }
.prop-list li::before { content: "• "; color: var(--gold2); }

details {
  background: var(--sf);
  border: 1px solid var(--bd);
  border-radius: 6px;
  margin-top: 1.5rem;
}
details summary {
  padding: .75rem 1rem;
  cursor: pointer;
  color: var(--muted);
  font-size: .88rem;
  user-select: none;
  list-style: none;
}
details summary:hover { color: var(--text); }
details[open] summary { border-bottom: 1px solid var(--bd); }
details pre {
  padding: 1rem;
  font-size: .78rem;
  overflow-x: auto;
  color: var(--text);
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: 'Cascadia Code', 'Fira Code', Consolas, monospace;
}

@media (max-width: 640px) {
  main { padding: 1rem; }
  .hero h1 { font-size: 1.8rem; }
  .cat-grid { grid-template-columns: repeat(auto-fill, minmax(130px, 1fr)); }
  .item-grid { grid-template-columns: 1fr; }
}
</style>
{{end}}

{{define "nav"}}
<nav>
  <a href="/" class="brand">⚔ D&amp;D 5e SRD</a>
  <span class="nav-sub">System Reference Document Explorer</span>
  <a href="/encounter" class="nav-enc">🎲 Encounter Generator</a>
  <a href="/map" class="nav-enc">🗺 Map Generator</a>
</nav>
{{end}}

{{define "foot"}}
<footer>
  Data provided by <a href="https://www.dnd5eapi.co/" target="_blank" rel="noopener">dnd5eapi.co</a>
  &middot; <a href="https://github.com/5e-bits/5e-srd-api" target="_blank" rel="noopener">GitHub</a>
  &middot; 5e SRD content &copy; Wizards of the Coast
</footer>
</body>
</html>
{{end}}

{{define "home"}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>D&amp;D 5e SRD Explorer</title>
  {{template "css" .}}
</head>
<body>
{{template "nav" .}}
<main>
  <div class="hero">
    <h1>D&amp;D 5e SRD Explorer</h1>
    <p>Browse the complete 5th Edition System Reference Document &mdash; spells, monsters, classes, items &amp; more.</p>
  </div>
  <div class="cat-grid">
    {{range .}}
    <a href="/{{.}}" class="cat-card">
      <span class="cat-icon">{{iconFor .}}</span>
      <span class="cat-name">{{prettyCat .}}</span>
    </a>
    {{end}}
  </div>
</main>
{{template "foot" .}}
{{end}}

{{define "category"}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{prettyCat .Category}} &mdash; D&amp;D 5e SRD</title>
  {{template "css" .}}
</head>
<body>
{{template "nav" .}}
<main>
  <div class="breadcrumb">
    <a href="/">Home</a> &rsaquo; {{prettyCat .Category}}
  </div>
  <h1 class="page-heading">
    {{iconFor .Category}} {{prettyCat .Category}}
    <span class="count-badge">{{.Count}}</span>
  </h1>
  <input
    type="search"
    id="search"
    class="search-box"
    placeholder="Filter {{prettyCat .Category}}..."
    autocomplete="off"
  >
  <div class="item-grid" id="grid">
    {{range .Results}}
    <a href="/{{$.Category}}/{{.Index}}" class="item-link" data-name="{{.Name}}">
      <span>{{.Name}}</span>
      {{if .Level}}<span class="lvl-badge">Lv {{.Level}}</span>{{end}}
    </a>
    {{end}}
  </div>
</main>
<script>
(function () {
  var s = document.getElementById('search');
  var g = document.getElementById('grid');
  var links = Array.prototype.slice.call(g.querySelectorAll('.item-link'));
  var noResults = null;
  s.addEventListener('input', function () {
    var q = this.value.toLowerCase();
    var visible = 0;
    links.forEach(function (el) {
      var show = el.dataset.name.toLowerCase().indexOf(q) !== -1;
      el.style.display = show ? '' : 'none';
      if (show) visible++;
    });
    if (visible === 0 && !noResults) {
      noResults = document.createElement('div');
      noResults.className = 'no-results';
      noResults.textContent = 'No results for "' + s.value + '"';
      g.appendChild(noResults);
    } else if (visible > 0 && noResults) {
      g.removeChild(noResults);
      noResults = null;
    } else if (noResults) {
      noResults.textContent = 'No results for "' + s.value + '"';
    }
  });
  s.focus();
})();
</script>
{{template "foot" .}}
{{end}}

{{define "item"}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.Name}} &mdash; D&amp;D 5e SRD</title>
  {{template "css" .}}
</head>
<body>
{{template "nav" .}}
<main>
  <div class="breadcrumb">
    <a href="/">Home</a> &rsaquo;
    <a href="/{{.Category}}">{{prettyCat .Category}}</a> &rsaquo;
    {{.Name}}
  </div>

  <div class="item-title">{{iconFor .Category}} {{.Name}}</div>
  <div class="item-cat">{{prettyCat .Category}}</div>
  <a
    href="https://www.dnd5eapi.co/api/2014/{{.Category}}/{{.Slug}}"
    target="_blank"
    rel="noopener"
    class="api-link"
  >View raw API endpoint ↗</a>

  {{if .Desc}}
  <p class="section-title">Description</p>
  <div class="description">
    {{range .Desc}}<p>{{.}}</p>{{end}}
  </div>
  {{end}}

  {{if .HigherLevel}}
  <div class="higher-level">
    <p class="higher-level-title">At Higher Levels</p>
    {{range .HigherLevel}}<p>{{.}}</p>{{end}}
  </div>
  {{end}}

  {{if .Props}}
  <p class="section-title">Properties</p>
  <div class="props-grid">
    {{range .Props}}
    <div class="prop">
      <div class="prop-label">{{.Label}}</div>
      {{if .IsMulti}}
      <ul class="prop-list">
        {{range .Values}}<li>{{.}}</li>{{end}}
      </ul>
      {{else}}
      <div class="prop-value">{{index .Values 0}}</div>
      {{end}}
    </div>
    {{end}}
  </div>
  {{end}}

  <details>
    <summary>Raw JSON data</summary>
    <pre>{{.JSON}}</pre>
  </details>
</main>
{{template "foot" .}}
{{end}}

{{define "encounter"}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Encounter Generator — D&amp;D 5e SRD</title>
  {{template "css" .}}
  <style>
  .enc-form {
    background: var(--sf);
    border: 1px solid var(--bd);
    border-radius: 10px;
    padding: 1.5rem;
    display: flex;
    flex-wrap: wrap;
    gap: 1.25rem;
    align-items: flex-end;
    margin-bottom: 2rem;
  }
  .form-group { display: flex; flex-direction: column; gap: .4rem; }
  .form-label { font-size: .75rem; color: var(--muted); text-transform: uppercase; letter-spacing: .06em; }
  .form-input, .form-select {
    background: var(--bg);
    border: 1px solid var(--bd);
    border-radius: 6px;
    color: var(--text);
    padding: .55rem .85rem;
    font-size: .95rem;
    min-width: 90px;
    outline: none;
  }
  .form-input:focus, .form-select:focus { border-color: var(--gold); }
  .btn-generate {
    background: var(--red);
    color: #fff;
    border: none;
    border-radius: 6px;
    padding: .6rem 1.5rem;
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    transition: background .2s;
    white-space: nowrap;
  }
  .btn-generate:hover { background: #7f1d1d; }

  .enc-section-title {
    font-size: .8rem;
    color: var(--gold);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: .07em;
    display: flex;
    align-items: center;
    gap: .6rem;
    margin: 1.75rem 0 .85rem;
  }
  .enc-section-title::after { content: ""; flex: 1; height: 1px; background: var(--bd); }

  .monster-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(270px, 1fr));
    gap: 1rem;
  }
  .monster-card {
    background: var(--sf);
    border: 1px solid var(--bd);
    border-radius: 8px;
    padding: 1rem 1.25rem;
    position: relative;
    overflow: hidden;
  }
  .monster-card::before {
    content: "";
    position: absolute; top: 0; left: 0; right: 0;
    height: 3px; background: var(--red);
  }
  .monster-name { font-size: 1.05rem; font-weight: 600; color: var(--gold); margin-bottom: .2rem; }
  .monster-name a { color: inherit; }
  .monster-name a:hover { text-decoration: underline; }
  .monster-meta { font-size: .78rem; color: var(--muted); margin-bottom: .75rem; text-transform: capitalize; }
  .count-tag {
    display: inline-block;
    background: var(--red); color: #fff;
    font-size: .7rem; font-weight: 700;
    border-radius: 20px; padding: .1rem .5rem;
    margin-left: .4rem; vertical-align: middle;
  }
  .monster-stats {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: .5rem;
  }
  .stat-box { background: var(--sf2); border-radius: 4px; padding: .4rem .5rem; text-align: center; }
  .stat-label { font-size: .62rem; color: var(--muted); text-transform: uppercase; letter-spacing: .04em; }
  .stat-val { font-size: .95rem; font-weight: 700; color: var(--text); margin-top: .1rem; }
  .stat-val.cr { color: var(--gold); }

  .diff-badge {
    font-weight: 700; font-size: .82rem;
    border-radius: 4px; padding: .15rem .65rem;
    background: rgba(255,255,255,.07);
  }

  .enc-summary {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(155px, 1fr));
    gap: .75rem;
    background: var(--sf);
    border: 1px solid var(--bd);
    border-radius: 8px;
    padding: 1rem 1.25rem;
  }
  .sum-stat { text-align: center; }
  .sum-label { font-size: .7rem; color: var(--muted); text-transform: uppercase; letter-spacing: .05em; }
  .sum-val { font-size: 1.45rem; font-weight: 700; color: var(--gold); line-height: 1.2; }
  .sum-sub { font-size: .73rem; color: var(--muted); }

  .loot-wrap { display: flex; gap: 2rem; flex-wrap: wrap; align-items: flex-start; margin-top: .5rem; }
  .gold-box { min-width: 160px; }
  .gold-val { font-size: 2rem; font-weight: 700; color: var(--gold); }
  .gold-sub { font-size: .82rem; color: var(--muted); margin-top: .2rem; }
  .loot-list { display: flex; flex-direction: column; gap: .5rem; flex: 1; min-width: 240px; }
  .loot-item {
    display: flex; align-items: center; gap: .75rem;
    background: var(--sf); border: 1px solid var(--bd);
    border-radius: 6px; padding: .6rem 1rem;
  }
  .loot-item.magic { border-color: var(--gold2); }
  .loot-icon { font-size: 1.25rem; }
  .loot-name { font-size: .9rem; color: var(--text); }
  .loot-badge {
    margin-left: auto; font-size: .7rem; border-radius: 20px; padding: .1rem .5rem;
    background: rgba(255,255,255,.07); color: var(--muted); white-space: nowrap;
  }
  .loot-badge.magic { background: rgba(201,166,60,.15); color: var(--gold); }

  .enc-error {
    background: #3b0000; border: 1px solid var(--red);
    border-radius: 6px; padding: 1rem 1.25rem;
    color: #fca5a5; margin-bottom: 1rem;
  }
  .enc-placeholder {
    text-align: center; padding: 3.5rem 1rem;
    color: var(--muted); font-size: 1rem;
  }
  .enc-placeholder .ph-icon { font-size: 3.5rem; display: block; margin-bottom: .75rem; }
  </style>
</head>
<body>
{{template "nav" .}}
<main>
  <div class="breadcrumb">
    <a href="/">Home</a> &rsaquo; Encounter Generator
  </div>
  <h1 class="page-heading">🎲 Encounter Generator</h1>
  <p style="color:var(--muted);margin-bottom:1.5rem">
    Build a balanced combat encounter backed by the live D&amp;D 5e SRD, complete with loot rewards.
  </p>

  <form class="enc-form" method="get" action="/encounter">
    <div class="form-group">
      <label class="form-label" for="size">Party Size</label>
      <input class="form-input" id="size" name="size" type="number" min="1" max="8" value="{{.PartySize}}">
    </div>
    <div class="form-group">
      <label class="form-label" for="level">Avg Level</label>
      <input class="form-input" id="level" name="level" type="number" min="1" max="20" value="{{.PartyLevel}}">
    </div>
    <div class="form-group">
      <label class="form-label" for="difficulty">Difficulty</label>
      <select class="form-select" id="difficulty" name="difficulty">
        <option value="easy"   {{if eq .Difficulty "easy"}}selected{{end}}>Easy</option>
        <option value="medium" {{if eq .Difficulty "medium"}}selected{{end}}>Medium</option>
        <option value="hard"   {{if eq .Difficulty "hard"}}selected{{end}}>Hard</option>
        <option value="deadly" {{if eq .Difficulty "deadly"}}selected{{end}}>Deadly</option>
      </select>
    </div>
    <button class="btn-generate" type="submit">⚔ Generate Encounter</button>
  </form>

  {{if .Error}}
  <div class="enc-error">⚠ {{.Error}}</div>
  {{end}}

  {{if .Result}}
  {{with .Result}}

  <div class="enc-section-title">
    Encounter Composition
    <span class="diff-badge" style="color:{{.DiffColor}}">{{.Difficulty}}</span>
  </div>
  <div class="monster-grid">
    {{range .Monsters}}
    <div class="monster-card">
      <div class="monster-name">
        <a href="/monsters/{{.Detail.Index}}">{{.Detail.Name}}</a>
        {{if gt .Count 1}}<span class="count-tag">&times;{{.Count}}</span>{{end}}
      </div>
      <div class="monster-meta">
        {{.Detail.Size}} {{.Detail.Type}}{{if .Detail.Alignment}} &middot; {{.Detail.Alignment}}{{end}}
      </div>
      <div class="monster-stats">
        <div class="stat-box">
          <div class="stat-label">CR</div>
          <div class="stat-val cr">{{.CRStr}}</div>
        </div>
        <div class="stat-box">
          <div class="stat-label">HP each</div>
          <div class="stat-val">{{.Detail.HitPoints}}</div>
        </div>
        <div class="stat-box">
          <div class="stat-label">Total HP</div>
          <div class="stat-val">{{.TotalHP}}</div>
        </div>
        <div class="stat-box">
          <div class="stat-label">XP each</div>
          <div class="stat-val">{{.XPEach}}</div>
        </div>
        <div class="stat-box">
          <div class="stat-label">Count</div>
          <div class="stat-val">{{.Count}}</div>
        </div>
        <div class="stat-box">
          <div class="stat-label">Group XP</div>
          <div class="stat-val">{{.TotalXP}}</div>
        </div>
      </div>
    </div>
    {{end}}
  </div>

  <div class="enc-section-title">Encounter Summary</div>
  <div class="enc-summary">
    <div class="sum-stat">
      <div class="sum-label">Party</div>
      <div class="sum-val">{{.PartySize}}</div>
      <div class="sum-sub">characters (Level {{.PartyLevel}})</div>
    </div>
    <div class="sum-stat">
      <div class="sum-label">XP Budget</div>
      <div class="sum-val">{{.Budget}}</div>
      <div class="sum-sub">for this difficulty</div>
    </div>
    <div class="sum-stat">
      <div class="sum-label">Raw XP</div>
      <div class="sum-val">{{.TotalXP}}</div>
      <div class="sum-sub">from all monsters</div>
    </div>
    <div class="sum-stat">
      <div class="sum-label">Adjusted XP</div>
      <div class="sum-val">{{.AdjustedXP}}</div>
      <div class="sum-sub">with group multiplier</div>
    </div>
    <div class="sum-stat">
      <div class="sum-label">XP / Player</div>
      <div class="sum-val">{{.XPPerPC}}</div>
      <div class="sum-sub">if victorious</div>
    </div>
  </div>

  <div class="enc-section-title">💰 Loot &amp; Rewards</div>
  <div class="loot-wrap">
    <div class="gold-box">
      <div class="gold-val">🪙 {{.Gold}} gp</div>
      {{if gt .GoldPerPC 0}}
      <div class="gold-sub">{{.GoldPerPC}} gp per character</div>
      {{end}}
    </div>
    {{if .Loot}}
    <div class="loot-list">
      {{range .Loot}}
      <div class="loot-item {{if eq .Type "magic"}}magic{{end}}">
        <span class="loot-icon">{{.Icon}}</span>
        <span class="loot-name">{{.Name}}</span>
        <span class="loot-badge {{if eq .Type "magic"}}magic{{end}}">
          {{if eq .Type "magic"}}✦ Magic Item{{else}}Gear{{end}}
        </span>
      </div>
      {{end}}
    </div>
    {{end}}
  </div>

  {{end}}{{/* with .Result */}}
  {{else if .ShowPlaceholder}}
  <div class="enc-placeholder">
    <span class="ph-icon">🎲</span>
    Set your party details above and hit <strong>⚔ Generate Encounter</strong> to roll into battle.
  </div>
  {{end}}
</main>
{{template "foot" .}}
{{end}}
`
