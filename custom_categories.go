package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const customCategoriesDir = "_categories"

var customFieldKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)
var customCategorySlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func decodeCustomCategory(data []byte) (CustomCategory, error) {
	var properties map[string]json.RawMessage
	if err := json.Unmarshal(data, &properties); err != nil || properties == nil {
		return CustomCategory{}, errors.New("category definition must be a JSON object")
	}
	for key := range properties {
		if key != "slug" && key != "title" && key != "fields" {
			return CustomCategory{}, fmt.Errorf("unknown category property %q", key)
		}
	}
	var fields []json.RawMessage
	if raw, ok := properties["fields"]; ok {
		if err := json.Unmarshal(raw, &fields); err != nil {
			return CustomCategory{}, errors.New("fields must be an array")
		}
	}
	for i, raw := range fields {
		var field map[string]json.RawMessage
		if err := json.Unmarshal(raw, &field); err != nil || field == nil {
			return CustomCategory{}, fmt.Errorf("field %d must be an object", i+1)
		}
		for key := range field {
			switch key {
			case "key", "label", "type", "display", "size", "bold", "italic":
			default:
				return CustomCategory{}, fmt.Errorf("unknown property %q in field %d", key, i+1)
			}
		}
	}
	var category CustomCategory
	if err := json.Unmarshal(data, &category); err != nil {
		return CustomCategory{}, err
	}
	return category, nil
}

func customCategorySlug(title string) string {
	var slug strings.Builder
	lastWasSeparator := false
	for _, character := range strings.ToLower(strings.TrimSpace(title)) {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			slug.WriteRune(character)
			lastWasSeparator = false
		} else if !lastWasSeparator && slug.Len() > 0 {
			slug.WriteByte('-')
			lastWasSeparator = true
		}
	}
	return strings.Trim(slug.String(), "-")
}

func normalizeCustomCategory(category CustomCategory) (CustomCategory, error) {
	category.Title = strings.TrimSpace(category.Title)
	if category.Title == "" || len([]rune(category.Title)) > 80 || !validCategoryName(category.Title) {
		return CustomCategory{}, errors.New("category title is required (80 characters at most)")
	}
	wantSlug := customCategorySlug(category.Title)
	if wantSlug == "" {
		return CustomCategory{}, errors.New("category title must contain ASCII letters or numbers")
	}
	for _, title := range neoTokyoCategoryNames {
		if normalizeCategory(title) == normalizeCategory(category.Title) {
			return CustomCategory{}, fmt.Errorf("%q is already a built-in guide category", category.Title)
		}
	}
	if wantSlug == subclassesFolder || category.Slug == subclassesFolder {
		return CustomCategory{}, errors.New(`"Subclasses" is reserved for class subclass entries`)
	}
	if category.Slug == "" {
		category.Slug = wantSlug
	} else if !customCategorySlugPattern.MatchString(category.Slug) {
		return CustomCategory{}, errors.New("category slug is invalid")
	}
	if len(category.Fields) == 0 || len(category.Fields) > 20 {
		return CustomCategory{}, errors.New("add between 1 and 20 custom fields")
	}
	seen := map[string]bool{"name": true, "image": true}
	for i := range category.Fields {
		field := &category.Fields[i]
		field.Key = strings.TrimSpace(field.Key)
		field.Label = strings.TrimSpace(field.Label)
		if !customFieldKeyPattern.MatchString(field.Key) {
			return CustomCategory{}, fmt.Errorf("field %d key must start with a lowercase letter and contain only lowercase letters, numbers, or underscores", i+1)
		}
		if seen[field.Key] {
			return CustomCategory{}, fmt.Errorf("field key %q is reserved or duplicated", field.Key)
		}
		seen[field.Key] = true
		if field.Label == "" || len([]rune(field.Label)) > 80 {
			return CustomCategory{}, fmt.Errorf("field %q needs a label (80 characters at most)", field.Key)
		}
		switch field.Type {
		case "text", "textarea", "list":
		default:
			return CustomCategory{}, fmt.Errorf("field %q has an unsupported input type", field.Label)
		}
		switch field.Display {
		case "paragraph", "heading", "quote", "callout", "badge":
		default:
			return CustomCategory{}, fmt.Errorf("field %q has an unsupported display style", field.Label)
		}
		switch field.Size {
		case "small", "normal", "large":
		default:
			return CustomCategory{}, fmt.Errorf("field %q has an unsupported text size", field.Label)
		}
	}
	return category, nil
}

func writeCustomCategoryDefinition(dir string, category CustomCategory) error {
	category, err := normalizeCustomCategory(category)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(category, "", "  ")
	if err != nil {
		return err
	}
	definitionsDir := filepath.Join(dir, customCategoriesDir)
	if err := os.MkdirAll(definitionsDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, category.Slug), 0o755); err != nil {
		return err
	}
	path := filepath.Join(definitionsDir, category.Slug+".json")
	tmp, err := os.CreateTemp(definitionsDir, ".category-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func loadCustomCategoryDefinitions(dir string) ([]CustomCategory, error) {
	path := filepath.Join(dir, customCategoriesDir)
	files, err := os.ReadDir(path)
	if errors.Is(err, os.ErrNotExist) {
		return []CustomCategory{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read custom category definitions: %w", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	categories := make([]CustomCategory, 0, len(files))
	seenSlugs := make(map[string]bool)
	seenTitles := make(map[string]bool)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".json") || strings.HasPrefix(file.Name(), ".") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(path, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("read custom category %s: %w", file.Name(), err)
		}
		category, err := decodeCustomCategory(data)
		if err != nil {
			return nil, fmt.Errorf("decode custom category %s: %w", file.Name(), err)
		}
		category, err = normalizeCustomCategory(category)
		if err != nil {
			return nil, fmt.Errorf("invalid custom category %s: %w", file.Name(), err)
		}
		if file.Name() != category.Slug+".json" || seenSlugs[category.Slug] || seenTitles[normalizeCategory(category.Title)] {
			return nil, fmt.Errorf("custom category %s has a duplicate or mismatched title/slug", file.Name())
		}
		seenSlugs[category.Slug] = true
		seenTitles[normalizeCategory(category.Title)] = true
		categories = append(categories, category)
	}
	return categories, nil
}

func customCategoryFor(guide *StarterGuide, key string) (CustomCategory, bool) {
	for _, category := range guide.CustomCategories {
		if normalizeCategory(category.Title) == normalizeCategory(key) ||
			normalizeCategory(category.Slug) == normalizeCategory(key) {
			return category, true
		}
	}
	return CustomCategory{}, false
}

func validateCustomEntry(raw json.RawMessage, category CustomCategory) (json.RawMessage, string, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return nil, "", errors.New("an entry must be a JSON object")
	}
	allowed := map[string]CustomField{"name": {Key: "name"}, "image": {Key: "image"}}
	for _, field := range category.Fields {
		allowed[field.Key] = field
	}
	for key := range values {
		if _, ok := allowed[key]; !ok {
			return nil, "", fmt.Errorf("unknown field %q", key)
		}
	}
	var name string
	if err := json.Unmarshal(values["name"], &name); err != nil {
		return nil, "", errors.New("name must be a string")
	}
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 200 {
		return nil, "", errors.New("name is required (200 characters at most)")
	}
	normalized := map[string]any{"name": name}
	if imageRaw, exists := values["image"]; exists {
		var image string
		if err := json.Unmarshal(imageRaw, &image); err != nil {
			return nil, "", errors.New("image must be a URL string")
		}
		normalized["image"] = strings.TrimSpace(image)
	}
	for _, field := range category.Fields {
		value, exists := values[field.Key]
		if !exists {
			continue
		}
		if field.Type == "list" {
			var list []string
			if err := json.Unmarshal(value, &list); err != nil {
				return nil, "", fmt.Errorf("%s must be an array of strings", field.Label)
			}
			for i := range list {
				list[i] = strings.TrimSpace(list[i])
				if len([]rune(list[i])) > 10000 {
					return nil, "", fmt.Errorf("%s values must be 10000 characters or fewer", field.Label)
				}
			}
			normalized[field.Key] = list
			continue
		}
		var text string
		if err := json.Unmarshal(value, &text); err != nil {
			return nil, "", fmt.Errorf("%s must be a string", field.Label)
		}
		if len([]rune(text)) > 10000 {
			return nil, "", fmt.Errorf("%s must be 10000 characters or fewer", field.Label)
		}
		normalized[field.Key] = text
	}
	encoded, err := json.Marshal(normalized)
	return encoded, name, err
}

func mapCustomCategory(category CustomCategory, data *StarterGuide) SectionNode {
	entries := data.Category.Custom[normalizeCategory(category.Slug)]
	children := make([]SectionNode, 0, len(entries))
	for _, entry := range entries {
		name, _ := entry["name"].(string)
		image, _ := entry["image"].(string)
		node := SectionNode{Title: name, Image: image}
		for _, field := range category.Fields {
			value, exists := entry[field.Key]
			if !exists || value == nil {
				continue
			}
			text := ""
			switch typed := value.(type) {
			case string:
				text = typed
			case []any:
				parts := make([]string, 0, len(typed))
				for _, item := range typed {
					parts = append(parts, fmt.Sprint(item))
				}
				text = strings.Join(parts, ", ")
			default:
				text = fmt.Sprint(value)
			}
			if strings.TrimSpace(text) == "" {
				continue
			}
			node.CustomFields = append(node.CustomFields, SectionField{
				Label: field.Label, Value: text, Display: field.Display, Size: field.Size,
				Bold: field.Bold, Italic: field.Italic,
			})
		}
		children = append(children, node)
	}
	return SectionNode{Title: category.Title, Children: children}
}

func validCategoryName(title string) bool {
	for _, character := range title {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}
