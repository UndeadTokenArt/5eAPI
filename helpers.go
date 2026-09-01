package main

import (
	"fmt"
	"sort"
	"strings"
)

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
