package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"unicode"
)

const (
	neoTokyoSearchLimit    = 30
	neoTokyoSearchMinChars = 2
)

type NeoTokyoSearchResult struct {
	Title    string `json:"title"`
	Path     string `json:"path"`
	Category string `json:"category"`
	URL      string `json:"url"`
	Snippet  string `json:"snippet,omitempty"`
}

type NeoTokyoSearchResponse struct {
	Query   string                 `json:"query"`
	Total   int                    `json:"total"`
	Results []NeoTokyoSearchResult `json:"results"`
}

type searchEntry struct {
	result NeoTokyoSearchResult
	title  string
	path   string
	body   string
}

var (
	searchIndexCache versionedCache[[]searchEntry]
)

var macronFolder = strings.NewReplacer("ā", "a", "ē", "e", "ī", "i", "ō", "o", "ū", "u")

func foldSearchText(value string) string {
	var out strings.Builder
	lastSpace := true
	for _, r := range macronFolder.Replace(strings.ToLower(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			lastSpace = false
		} else if !lastSpace {
			out.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(out.String())
}

func snippetOf(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > 160 {
		return string(runes[:160]) + "…"
	}
	return value
}

func nodeBodyText(node SectionNode) string {
	parts := []string{node.Subtitle, node.Description, node.Content}
	parts = append(parts, node.Tags...)
	keys := make([]string, 0, len(node.Attributes))
	for key := range node.Attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts = append(parts, fmt.Sprint(node.Attributes[key]))
	}
	return strings.Join(parts, " ")
}

func indexNodes(entries *[]searchEntry, nodes []SectionNode, category, base string, parents []string) {
	for _, node := range nodes {
		if node.Generated {
			continue
		}
		labels := append(append([]string{}, parents...), node.Title)
		body := nodeBodyText(node)
		snippet := node.Description
		if snippet == "" {
			snippet = node.Content
		}
		*entries = append(*entries, searchEntry{
			result: NeoTokyoSearchResult{
				Title:    node.Title,
				Path:     strings.Join(labels, " / "),
				Category: category,
				URL:      base + node.ID,
				Snippet:  snippetOf(snippet),
			},
			title: foldSearchText(node.Title),
			path:  foldSearchText(strings.Join(labels, " ")),
			body:  foldSearchText(body),
		})
		indexNodes(entries, node.Children, category, base, labels)
	}
}

func buildSearchIndex(guide *StarterGuide) []searchEntry {
	var entries []searchEntry
	for _, category := range neoTokyoCategoryNames {
		node, base := categoryNode(guide, category)
		if base == "" {
			continue
		}
		// The category itself is a browse target; its children are the searchable entries.
		indexNodes(&entries, node.Children, node.Title, base, []string{node.Title})
	}
	return entries
}

func scoreSearchEntry(entry searchEntry, query string, terms []string) int {
	switch {
	case entry.title == query:
		return 0
	case strings.HasPrefix(entry.title, query):
		return 1
	case strings.Contains(" "+entry.title, " "+query):
		return 2
	case strings.Contains(entry.title, query):
		return 3
	case strings.Contains(entry.path, query):
		return 4
	}
	if containsAll(entry.title+" "+entry.path, terms) {
		return 5
	}
	if containsAll(entry.title+" "+entry.path+" "+entry.body, terms) {
		return 6
	}
	return -1
}

func containsAll(haystack string, terms []string) bool {
	for _, term := range terms {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func searchNeoTokyo(index []searchEntry, rawQuery string) NeoTokyoSearchResponse {
	query := foldSearchText(rawQuery)
	response := NeoTokyoSearchResponse{Query: strings.TrimSpace(rawQuery), Results: []NeoTokyoSearchResult{}}
	if len([]rune(query)) < neoTokyoSearchMinChars {
		return response
	}
	terms := strings.Fields(query)
	type scored struct {
		entry searchEntry
		score int
	}
	var matches []scored
	for _, entry := range index {
		if score := scoreSearchEntry(entry, query, terms); score >= 0 {
			matches = append(matches, scored{entry, score})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score < matches[j].score
		}
		return len(matches[i].entry.title) < len(matches[j].entry.title)
	})
	response.Total = len(matches)
	for i := 0; i < len(matches) && i < neoTokyoSearchLimit; i++ {
		response.Results = append(response.Results, matches[i].entry.result)
	}
	return response
}

func neoTokyoSearchHandler(w http.ResponseWriter, r *http.Request) {
	searchIndex, err := searchIndexCache.get(buildSearchIndex)
	if err != nil {
		http.Error(w, "Could not load the starter guide.", http.StatusInternalServerError)
		return
	}
	query := r.URL.Query().Get("q")
	if len(query) > 200 {
		query = query[:200]
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(searchNeoTokyo(searchIndex, query))
}
