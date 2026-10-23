package main

import (
	"encoding/json"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

const (
	editorFileName     = "zz-editor-additions.json" // sorts last so existing entry IDs never shift
	editorMaxBody      = 128 << 10
	editorMaxEntries   = 500
	editorDefaultGroup = "editor"
)

type editorCategory struct {
	Slug  string
	Title string
	// check validates one entry and returns its normalised JSON.
	check func(json.RawMessage) (json.RawMessage, string, error)
}

// rejectUnknownFields errors when a top-level key isn't a JSON field of the type (typos would be dropped silently).
func rejectUnknownFields(raw json.RawMessage, typ reflect.Type) error {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return errors.New("an entry must be a JSON object")
	}
	allowed := map[string]bool{}
	var collect func(reflect.Type)
	collect = func(t reflect.Type) {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.Anonymous && field.Tag.Get("json") == "" && field.Type.Kind() == reflect.Struct {
				collect(field.Type)
				continue
			}
			allowed[strings.Split(field.Tag.Get("json"), ",")[0]] = true
		}
	}
	collect(typ)
	for key := range keys {
		if !allowed[key] {
			return fmt.Errorf("unknown field %q", key)
		}
	}
	return nil
}

func typedCheck[T any](name func(*T) *string) func(json.RawMessage) (json.RawMessage, string, error) {
	return func(raw json.RawMessage) (json.RawMessage, string, error) {
		var value T
		if err := rejectUnknownFields(raw, reflect.TypeOf(value)); err != nil {
			return nil, "", err
		}
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, "", err
		}
		entryName := strings.TrimSpace(*name(&value))
		if entryName == "" || len(entryName) > 200 {
			return nil, "", errors.New("name is required (200 characters at most)")
		}
		*name(&value) = entryName
		out, err := json.Marshal(value)
		return out, entryName, err
	}
}

// checkSubclassAddition validates a subclass and that its parent class exists and doesn't already have it.
func checkSubclassAddition(raw json.RawMessage) (json.RawMessage, string, error) {
	normalised, name, err := typedCheck(func(v *SubclassAddition) *string { return &v.Name })(raw)
	if err != nil {
		return nil, "", err
	}
	var addition SubclassAddition
	if err := json.Unmarshal(normalised, &addition); err != nil {
		return nil, "", err
	}
	guide, _, err := currentGuide()
	if err != nil {
		return nil, "", err
	}
	i := findClass(guide.Category.Classes, addition.Class)
	if i < 0 {
		return nil, "", fmt.Errorf("class %q does not exist; pick one of the guide's classes", addition.Class)
	}
	for _, existing := range guide.Category.Classes[i].Subclasses {
		if strings.EqualFold(existing.Name, name) {
			return nil, "", fmt.Errorf("%s already has a subclass named %q", guide.Category.Classes[i].Name, name)
		}
	}
	addition.Class = guide.Category.Classes[i].Name
	out, err := json.Marshal(addition)
	return out, name, err
}

var editorCategories = []editorCategory{
	{"abilities", "Abilities", typedCheck(func(v *Ability) *string { return &v.Name })},
	{"backgrounds", "Backgrounds", typedCheck(func(v *Background) *string { return &v.Name })},
	{"species", "Species", typedCheck(func(v *Species) *string { return &v.Name })},
	{"classes", "Classes", typedCheck(func(v *Class) *string { return &v.Name })},
	{subclassesFolder, "Subclasses", checkSubclassAddition},
	{"feats", "Feats", typedCheck(func(v *Feat) *string { return &v.Name })},
	{"adventuring-gear-and-containers", "Adventuring Gear & Containers", typedCheck(func(v *BasicItem) *string { return &v.Name })},
	{"tools-kits-and-consumables", "Tools, Kits & Consumables", typedCheck(func(v *BasicItem) *string { return &v.Name })},
	{"ammunition", "Ammunition", typedCheck(func(v *BasicItem) *string { return &v.Name })},
	{"weapons", "Weapons", typedCheck(func(v *Weapon) *string { return &v.Name })},
	{"armor-and-shields", "Armor & Shields", typedCheck(func(v *BasicItem) *string { return &v.Name })},
	{"mounts-vehicles-and-harnesses", "Mounts, Vehicles & Harnesses", typedCheck(func(v *BasicItem) *string { return &v.Name })},
	{"foci-and-instruments", "Foci & Instruments", typedCheck(func(v *BasicItem) *string { return &v.Name })},
	{"spells", "Spells", typedCheck(func(v *Spell) *string { return &v.Name })},
	{"conditions", "Conditions", typedCheck(func(v *Condition) *string { return &v.Name })},
}

func editorCategoryFor(slug string) (editorCategory, bool, error) {
	for _, category := range editorCategories {
		if category.Slug == slug {
			return category, true, nil
		}
	}
	guide, _, err := currentGuide()
	if err != nil {
		return editorCategory{}, false, err
	}
	for _, definition := range guide.CustomCategories {
		if definition.Slug == slug {
			category := editorCategory{
				Slug: definition.Slug, Title: definition.Title,
				check: func(raw json.RawMessage) (json.RawMessage, string, error) {
					return validateCustomEntry(raw, definition)
				},
			}
			return category, true, nil
		}
	}
	return editorCategory{}, false, nil
}

func editorGroup() string {
	if group := strings.TrimSpace(os.Getenv("EDITOR_GROUP")); group != "" {
		return group
	}
	return editorDefaultGroup
}

// editorUser returns the signed-in user when Authelia reports them in the editor group.
func editorUser(r *http.Request) (string, bool) {
	user := strings.TrimSpace(r.Header.Get("Remote-User"))
	if user == "" || len(user) > 256 || strings.IndexFunc(user, unicode.IsControl) >= 0 {
		return "", false
	}
	want := editorGroup()
	for _, group := range strings.Split(r.Header.Get("Remote-Groups"), ",") {
		if strings.EqualFold(strings.TrimSpace(group), want) {
			return user, true
		}
	}
	return "", false
}

func requireEditor(w http.ResponseWriter, r *http.Request) (string, bool) {
	user, ok := editorUser(r)
	if !ok {
		writeCharacterError(w, http.StatusForbidden, "The editor portal is limited to members of the "+editorGroup()+" group.")
	}
	return user, ok
}

var editorMu sync.Mutex

func editorFilePath(slug string) string {
	return filepath.Join(guideDir(), slug, editorFileName)
}

func readEditorEntries(slug string) ([]json.RawMessage, error) {
	entries, err := readJSONEntries(editorFilePath(slug))
	if errors.Is(err, os.ErrNotExist) {
		return []json.RawMessage{}, nil
	}
	return entries, err
}

func writeEditorEntries(slug string, entries []json.RawMessage) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	path := editorFilePath(slug)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".editor-*.tmp")
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

type editorEntryView struct {
	Index int             `json:"index"`
	Name  string          `json:"name"`
	Entry json.RawMessage `json:"entry"`
}

func editorPageHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := editorUser(r)
	if !ok {
		http.Error(w, "The editor portal is limited to members of the "+editorGroup()+" group. Sign in through Authelia with an editor account.", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	var classNames []string
	guide, _, err := currentGuide()
	if err != nil {
		log.Println("editor guide load:", err)
		http.Error(w, "Could not load guide data for editor", http.StatusInternalServerError)
		return
	}
	categories := append([]editorCategory{}, editorCategories...)
	for _, class := range guide.Category.Classes {
		classNames = append(classNames, class.Name)
	}
	for _, definition := range guide.CustomCategories {
		categories = append(categories, editorCategory{Slug: definition.Slug, Title: definition.Title})
	}
	categoryJSON, err := json.Marshal(guide.CustomCategories)
	if err != nil {
		log.Println("editor category JSON:", err)
		http.Error(w, "Could not load custom categories", http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"User": user, "Categories": categories, "ClassNames": classNames,
		"CategoryDefinitions": guide.CustomCategories,
		"CustomCategories":    htmltemplate.JS(categoryJSON),
	}
	if err := tmpl.ExecuteTemplate(w, "editor", data); err != nil {
		log.Println("editor page template error:", err)
		http.Error(w, "Could not render editor", http.StatusInternalServerError)
	}
}

func editorCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := requireEditor(w, r)
	if !ok {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodGet {
		guide, _, err := currentGuide()
		if err != nil {
			writeCharacterError(w, http.StatusInternalServerError, "Could not load guide categories.")
			return
		}
		writeCharacterJSON(w, http.StatusOK, guide.CustomCategories)
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		w.Header().Set("Allow", "GET, POST, PUT")
		writeCharacterError(w, http.StatusMethodNotAllowed, fmt.Sprintf("Method %s not allowed.", r.Method))
		return
	}
	if !sameOriginMutation(w, r) {
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeCharacterError(w, http.StatusUnsupportedMediaType, "Expected application/json.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, editorMaxBody)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeCharacterError(w, http.StatusBadRequest, "Invalid category definition.")
		return
	}
	category, err := decodeCustomCategory(raw)
	if err != nil {
		writeCharacterError(w, http.StatusBadRequest, "Invalid category definition.")
		return
	}
	category, err = normalizeCustomCategory(category)
	if err != nil {
		writeCharacterError(w, http.StatusBadRequest, "Invalid category: "+err.Error())
		return
	}
	if r.Method == http.MethodPut {
		slug := r.PathValue("slug")
		category.Slug = slug
		category, err = normalizeCustomCategory(category)
		if err != nil {
			writeCharacterError(w, http.StatusBadRequest, "Invalid category: "+err.Error())
			return
		}
	}
	editorMu.Lock()
	defer editorMu.Unlock()
	existing, err := loadCustomCategoryDefinitions(guideDir())
	if err != nil {
		writeCharacterError(w, http.StatusInternalServerError, "Could not load existing category definitions.")
		return
	}
	for _, definition := range existing {
		if definition.Slug != category.Slug && normalizeCategory(definition.Title) == normalizeCategory(category.Title) {
			writeCharacterError(w, http.StatusConflict, "A category with that title already exists.")
			return
		}
	}
	if r.Method == http.MethodPut {
		found := false
		for _, definition := range existing {
			found = found || definition.Slug == category.Slug
		}
		if !found {
			writeCharacterError(w, http.StatusNotFound, "Custom category not found.")
			return
		}
		entries, err := readEditorEntries(category.Slug)
		if err != nil {
			writeCharacterError(w, http.StatusInternalServerError, "Could not read category entries.")
			return
		}
		for index, entry := range entries {
			if _, _, err := validateCustomEntry(entry, category); err != nil {
				writeCharacterError(w, http.StatusConflict, fmt.Sprintf("Category changes would make entry %d invalid: %v. Update that entry first.", index+1, err))
				return
			}
		}
	}
	if err := writeCustomCategoryDefinition(guideDir(), category); err != nil {
		log.Printf("editor %s create category: %v", user, err)
		writeCharacterError(w, http.StatusInternalServerError, "Could not save the category definition.")
		return
	}
	invalidateGuideCache()
	if r.Method == http.MethodPut {
		log.Printf("editor %s updated category %q", user, category.Title)
		writeCharacterJSON(w, http.StatusOK, category)
		return
	}
	log.Printf("editor %s created category %q", user, category.Title)
	writeCharacterJSON(w, http.StatusCreated, category)
}

func editorEntriesHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := requireEditor(w, r)
	if !ok {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	category, found, categoryErr := editorCategoryFor(r.URL.Query().Get("category"))
	if categoryErr != nil {
		writeCharacterError(w, http.StatusInternalServerError, "Could not load guide categories.")
		return
	}
	if r.Method != http.MethodGet {
		if !sameOriginMutation(w, r) {
			return
		}
		if r.Method == http.MethodPost {
			var body struct {
				Category string          `json:"category"`
				Entry    json.RawMessage `json:"entry"`
			}
			r.Body = http.MaxBytesReader(w, r.Body, editorMaxBody)
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				writeCharacterError(w, http.StatusUnsupportedMediaType, "Expected application/json.")
				return
			}
			raw, err := io.ReadAll(r.Body)
			if err == nil {
				err = json.Unmarshal(raw, &body)
			}
			if err != nil || len(body.Entry) == 0 {
				writeCharacterError(w, http.StatusBadRequest, "Invalid request body.")
				return
			}
			category, found, categoryErr = editorCategoryFor(body.Category)
			if categoryErr != nil {
				writeCharacterError(w, http.StatusInternalServerError, "Could not load guide categories.")
				return
			}
			if !found {
				writeCharacterError(w, http.StatusBadRequest, "Unknown category.")
				return
			}
			normalised, name, err := category.check(body.Entry)
			if err != nil {
				writeCharacterError(w, http.StatusBadRequest, "Invalid entry: "+err.Error())
				return
			}
			editorMu.Lock()
			defer editorMu.Unlock()
			entries, err := readEditorEntries(category.Slug)
			if err != nil {
				writeCharacterError(w, http.StatusInternalServerError, "The existing additions file is unreadable: "+err.Error())
				return
			}
			if len(entries) >= editorMaxEntries {
				writeCharacterError(w, http.StatusBadRequest, "Too many editor additions in this category.")
				return
			}
			entries = append(entries, normalised)
			if err := writeEditorEntries(category.Slug, entries); err != nil {
				log.Println("editor write:", err)
				writeCharacterError(w, http.StatusInternalServerError, "Could not save the entry. Is the data folder writable?")
				return
			}
			invalidateGuideCache()
			log.Printf("editor %s added %q to %s", user, name, category.Slug)
			writeCharacterJSON(w, http.StatusCreated, map[string]any{"index": len(entries) - 1, "name": name})
			return
		}
		if r.Method == http.MethodPut {
			index, err := strconv.Atoi(r.URL.Query().Get("index"))
			if err != nil || index < 0 {
				writeCharacterError(w, http.StatusNotFound, "Entry not found.")
				return
			}
			var body struct {
				Category string          `json:"category"`
				Entry    json.RawMessage `json:"entry"`
			}
			r.Body = http.MaxBytesReader(w, r.Body, editorMaxBody)
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				writeCharacterError(w, http.StatusUnsupportedMediaType, "Expected application/json.")
				return
			}
			raw, err := io.ReadAll(r.Body)
			if err == nil {
				err = json.Unmarshal(raw, &body)
			}
			if err != nil || len(body.Entry) == 0 || body.Category != category.Slug {
				writeCharacterError(w, http.StatusBadRequest, "Invalid request body.")
				return
			}
			normalised, name, err := category.check(body.Entry)
			if err != nil {
				writeCharacterError(w, http.StatusBadRequest, "Invalid entry: "+err.Error())
				return
			}
			editorMu.Lock()
			defer editorMu.Unlock()
			entries, err := readEditorEntries(category.Slug)
			if err != nil {
				writeCharacterError(w, http.StatusInternalServerError, "The existing additions file is unreadable: "+err.Error())
				return
			}
			if index >= len(entries) {
				writeCharacterError(w, http.StatusNotFound, "Entry not found.")
				return
			}
			entries[index] = normalised
			if err := writeEditorEntries(category.Slug, entries); err != nil {
				log.Println("editor update:", err)
				writeCharacterError(w, http.StatusInternalServerError, "Could not save the entry. Is the data folder writable?")
				return
			}
			invalidateGuideCache()
			log.Printf("editor %s updated %q in %s", user, name, category.Slug)
			writeCharacterJSON(w, http.StatusOK, map[string]any{"index": index, "name": name})
			return
		}
	}
	if !found {
		writeCharacterError(w, http.StatusBadRequest, "Unknown category.")
		return
	}
	switch r.Method {
	case http.MethodGet:
		editorMu.Lock()
		entries, err := readEditorEntries(category.Slug)
		editorMu.Unlock()
		if err != nil {
			writeCharacterError(w, http.StatusInternalServerError, "The additions file is unreadable: "+err.Error())
			return
		}
		views := make([]editorEntryView, 0, len(entries))
		for i, entry := range entries {
			var named struct {
				Name string `json:"name"`
			}
			_ = json.Unmarshal(entry, &named)
			views = append(views, editorEntryView{Index: i, Name: named.Name, Entry: entry})
		}
		writeCharacterJSON(w, http.StatusOK, views)
	case http.MethodDelete:
		index, err := strconv.Atoi(r.URL.Query().Get("index"))
		editorMu.Lock()
		defer editorMu.Unlock()
		entries, readErr := readEditorEntries(category.Slug)
		if readErr != nil || err != nil || index < 0 || index >= len(entries) {
			writeCharacterError(w, http.StatusNotFound, "Entry not found.")
			return
		}
		entries = append(entries[:index], entries[index+1:]...)
		if err := writeEditorEntries(category.Slug, entries); err != nil {
			writeCharacterError(w, http.StatusInternalServerError, "Could not save the change.")
			return
		}
		invalidateGuideCache()
		log.Printf("editor %s removed addition %d from %s", user, index, category.Slug)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, POST, PUT, DELETE")
		writeCharacterError(w, http.StatusMethodNotAllowed, fmt.Sprintf("Method %s not allowed.", r.Method))
	}
}
