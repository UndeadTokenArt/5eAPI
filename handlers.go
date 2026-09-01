package main

import (
	"dnd5e/encounter"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

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

func serveStaticAsset(w http.ResponseWriter, r *http.Request, prefix, root string) {
	http.StripPrefix(prefix, http.FileServer(http.Dir(root))).ServeHTTP(w, r)
}

func setupRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/static/"):
			serveStaticAsset(w, r, "/static/", "static")
		case strings.HasPrefix(r.URL.Path, "/tiles/"):
			serveStaticAsset(w, r, "/tiles/", "tiles")
		case r.URL.Path == "/":
			homeHandler(w, r)
		case r.URL.Path == "/api/map/biomes":
			mapBiomesHandler(w, r)
		case r.URL.Path == "/encounter":
			encounterHandler(w, r)
		case r.URL.Path == "/map":
			mapHandler(w, r)
		case r.URL.Path == "/map/editor":
			mapEditorHandler(w, r)
		case r.URL.Path == "/shop":
			shopHandler(w, r)
		default:
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			switch len(parts) {
			case 1:
				categoryHandler(w, r, parts[0])
			case 2:
				itemHandler(w, r, parts[0], parts[1])
			default:
				http.NotFound(w, r)
			}
		}
	})

	return mux
}

func router(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/static/") {
		serveStaticAsset(w, r, "/static/", "static")
		return
	}
	if strings.HasPrefix(r.URL.Path, "/tiles/") {
		serveStaticAsset(w, r, "/tiles/", "tiles")
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
	if parts[0] == "shop" {
		shopHandler(w, r)
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
		result, err := encounter.GenerateEncounter(partySize, partyLevel, difficulty)
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
