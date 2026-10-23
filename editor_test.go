package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func editorRequest(method, target, body string, groups string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Host = "example.test"
	request.Header.Set("Origin", "http://example.test")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Remote-User", "zelekhut")
	if groups != "" {
		request.Header.Set("Remote-Groups", groups)
	}
	return request
}

func TestEditorRequiresGroup(t *testing.T) {
	t.Setenv("NEOTOKYO_DATA_DIR", t.TempDir())
	for _, groups := range []string{"", "5etools", "editors"} {
		recorder := httptest.NewRecorder()
		editorEntriesHandler(recorder, editorRequest(http.MethodPost, "/api/editor/entries", `{"category":"spells","entry":{"name":"X"}}`, groups))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("groups %q: status %d", groups, recorder.Code)
		}
	}
	page := httptest.NewRecorder()
	editorPageHandler(page, editorRequest(http.MethodGet, "/editor", "", "5etools"))
	if page.Code != http.StatusForbidden {
		t.Fatalf("page status %d", page.Code)
	}
}

func TestEditorAddListDeleteAndGuideReload(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NEOTOKYO_DATA_DIR", dir)
	writeFile(t, filepath.Join(dir, "spells", "spells.json"), `[{"name":"Base"}]`)
	groups := "5etools, Editor"

	add := httptest.NewRecorder()
	editorEntriesHandler(add, editorRequest(http.MethodPost, "/api/editor/entries", `{"category":"spells","entry":{"name":" New Spell ","level":"1","components":["V"]}}`, groups))
	if add.Code != http.StatusCreated {
		t.Fatalf("add status %d: %s", add.Code, add.Body.String())
	}
	guide, err := loadGuideFromDir(dir)
	if err != nil || len(guide.Category.Spells) != 2 || guide.Category.Spells[1].Name != "New Spell" {
		t.Fatalf("guide did not pick up the addition: %v %+v", err, guide)
	}

	list := httptest.NewRecorder()
	editorEntriesHandler(list, editorRequest(http.MethodGet, "/api/editor/entries?category=spells", "", groups))
	if !strings.Contains(list.Body.String(), "New Spell") {
		t.Fatalf("list: %s", list.Body.String())
	}

	del := httptest.NewRecorder()
	editorEntriesHandler(del, editorRequest(http.MethodDelete, "/api/editor/entries?category=spells&index=0", "", groups))
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete status %d", del.Code)
	}
	if _, err := os.Stat(filepath.Join(dir, "spells", "spells.json")); err != nil {
		t.Fatal("bundled file must be untouched")
	}
	guide, _ = loadGuideFromDir(dir)
	if len(guide.Category.Spells) != 1 {
		t.Fatalf("expected addition removed, got %d spells", len(guide.Category.Spells))
	}
}

func TestEditorAddsSubclassNestedUnderParentClass(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NEOTOKYO_DATA_DIR", dir)
	writeFile(t, filepath.Join(dir, "classes", "classes.json"), `[{"name":"Alpha","subclasses":[{"name":"Existing"}]},{"name":"Beta"}]`)
	groups := "Editor"
	post := func(entry string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		editorEntriesHandler(recorder, editorRequest(http.MethodPost, "/api/editor/entries", `{"category":"subclasses","entry":`+entry+`}`, groups))
		return recorder
	}

	if rec := post(`{"class":"beta","name":"Fresh","features":[{"name":"Trick (Level 3)"}]}`); rec.Code != http.StatusCreated {
		t.Fatalf("add status %d: %s", rec.Code, rec.Body.String())
	}
	for _, bad := range []string{
		`{"class":"Missing","name":"X"}`,
		`{"class":"Alpha","name":"existing"}`,
		`{"class":"Alpha","name":"X","bogus":1}`,
	} {
		if rec := post(bad); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", bad, rec.Code)
		}
	}

	guide, err := loadGuideFromDir(dir)
	if err != nil || len(guide.Category.Classes) != 2 {
		t.Fatalf("guide: %v %+v", err, guide)
	}
	beta := guide.Category.Classes[1]
	if len(beta.Subclasses) != 1 || beta.Subclasses[0].Name != "Fresh" || len(beta.Subclasses[0].Features) != 1 {
		t.Fatalf("subclass not nested under Beta: %+v", beta)
	}
	if len(guide.Category.Classes[0].Subclasses) != 1 {
		t.Fatalf("Alpha subclasses changed: %+v", guide.Category.Classes[0])
	}
}

func TestEditorRejectsBadEntries(t *testing.T) {
	t.Setenv("NEOTOKYO_DATA_DIR", t.TempDir())
	cases := []string{
		`{"category":"nope","entry":{"name":"X"}}`,
		`{"category":"spells","entry":{"name":""}}`,
		`{"category":"spells","entry":{"name":"X","bogus":1}}`,
		`{"category":"../etc","entry":{"name":"X"}}`,
	}
	for _, body := range cases {
		recorder := httptest.NewRecorder()
		editorEntriesHandler(recorder, editorRequest(http.MethodPost, "/api/editor/entries", body, "editor"))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: status %d", body, recorder.Code)
		}
	}
	noOrigin := editorRequest(http.MethodPost, "/api/editor/entries", `{"category":"spells","entry":{"name":"X"}}`, "editor")
	noOrigin.Header.Del("Origin")
	recorder := httptest.NewRecorder()
	editorEntriesHandler(recorder, noOrigin)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("missing origin: status %d", recorder.Code)
	}
}

func TestEditorPageRenders(t *testing.T) {
	t.Setenv("NEOTOKYO_DATA_DIR", t.TempDir())
	recorder := httptest.NewRecorder()
	editorPageHandler(recorder, editorRequest(http.MethodGet, "/editor", "", "editor"))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `value="spells"`) ||
		!strings.Contains(recorder.Body.String(), "Create a guide category") {
		t.Fatalf("status %d", recorder.Code)
	}
}

func TestEditorCreatesCustomCategoryWithStyledFields(t *testing.T) {
	t.Setenv("NEOTOKYO_DATA_DIR", t.TempDir())
	groups := "editor"
	categoryRequest := editorRequest(http.MethodPost, "/api/editor/categories", `{
		"title":"Personas",
		"fields":[
			{"key":"history","label":"History","type":"textarea","display":"callout","size":"large","bold":true,"italic":true},
			{"key":"traits","label":"Traits","type":"list","display":"badge","size":"small"}
		]
	}`, groups)
	categoryResponse := httptest.NewRecorder()
	editorCategoriesHandler(categoryResponse, categoryRequest)
	if categoryResponse.Code != http.StatusCreated {
		t.Fatalf("create category status %d: %s", categoryResponse.Code, categoryResponse.Body.String())
	}
	if !strings.Contains(categoryResponse.Body.String(), `"slug":"personas"`) {
		t.Fatalf("category response: %s", categoryResponse.Body.String())
	}
	editorPage := httptest.NewRecorder()
	editorPageHandler(editorPage, editorRequest(http.MethodGet, "/editor", "", groups))
	if editorPage.Code != http.StatusOK ||
		!strings.Contains(editorPage.Body.String(), `"display":"callout"`) ||
		!strings.Contains(editorPage.Body.String(), `value="personas"`) ||
		!strings.Contains(editorPage.Body.String(), `id="category-to-edit"`) ||
		!strings.Contains(editorPage.Body.String(), `<option value="personas">Personas</option>`) {
		t.Fatalf("category definition was not retained for the editor: status %d", editorPage.Code)
	}

	add := httptest.NewRecorder()
	entry := `{"category":"personas","entry":{"name":"Rin Kato","history":"<script>past</script>","traits":["patient","curious"]}}`
	editorEntriesHandler(add, editorRequest(http.MethodPost, "/api/editor/entries", entry, groups))
	if add.Code != http.StatusCreated {
		t.Fatalf("add custom entry status %d: %s", add.Code, add.Body.String())
	}

	page := httptest.NewRecorder()
	setupRouter().ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/neotokyo/personas", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("custom category page status %d: %s", page.Code, page.Body.String())
	}
	body := page.Body.String()
	for _, want := range []string{
		"Rin Kato",
		`custom-field-callout custom-field-size-large custom-field-bold custom-field-italic`,
		`custom-field-badge custom-field-size-small`,
		"&lt;script&gt;past&lt;/script&gt;",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("custom category page missing %q", want)
		}
	}
	if strings.Contains(body, "<script>past</script>") {
		t.Fatal("custom field content was rendered as HTML")
	}

	root := httptest.NewRecorder()
	setupRouter().ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/neotokyo", nil))
	if root.Code != http.StatusOK || !strings.Contains(root.Body.String(), `href="/neotokyo/personas"`) {
		t.Fatalf("custom category missing from guide menu: status %d", root.Code)
	}

	search := httptest.NewRecorder()
	setupRouter().ServeHTTP(search, httptest.NewRequest(http.MethodGet, "/api/neotokyo/search?q=curious", nil))
	if search.Code != http.StatusOK || !strings.Contains(search.Body.String(), `/neotokyo/personas#personas-1-rin-kato-1`) {
		t.Fatalf("custom category entry missing from search: status %d, body %s", search.Code, search.Body.String())
	}
}

func TestEditorUpdatesCustomCategoryAndEntry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NEOTOKYO_DATA_DIR", dir)
	groups := "editor"
	category := `{"title":"Personas","fields":[{"key":"history","label":"History","type":"textarea","display":"paragraph","size":"normal"}]}`
	created := httptest.NewRecorder()
	editorCategoriesHandler(created, editorRequest(http.MethodPost, "/api/editor/categories", category, groups))
	if created.Code != http.StatusCreated {
		t.Fatalf("create category: %d %s", created.Code, created.Body.String())
	}

	added := httptest.NewRecorder()
	editorEntriesHandler(added, editorRequest(http.MethodPost, "/api/editor/entries",
		`{"category":"personas","entry":{"name":"Rin","history":"Old"}}`, groups))
	if added.Code != http.StatusCreated {
		t.Fatalf("add entry: %d %s", added.Code, added.Body.String())
	}

	update := editorRequest(http.MethodPut, "/api/editor/categories/personas",
		`{"title":"People","fields":[{"key":"history","label":"Backstory","type":"textarea","display":"quote","size":"large"}]}`, groups)
	update.SetPathValue("slug", "personas")
	updated := httptest.NewRecorder()
	editorCategoriesHandler(updated, update)
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), `"slug":"personas"`) {
		t.Fatalf("update category: %d %s", updated.Code, updated.Body.String())
	}

	incompatible := editorRequest(http.MethodPut, "/api/editor/categories/personas",
		`{"title":"People","fields":[{"key":"bio","label":"Bio","type":"text","display":"paragraph","size":"normal"}]}`, groups)
	incompatible.SetPathValue("slug", "personas")
	rejected := httptest.NewRecorder()
	editorCategoriesHandler(rejected, incompatible)
	if rejected.Code != http.StatusConflict {
		t.Fatalf("incompatible schema update: %d %s", rejected.Code, rejected.Body.String())
	}

	entryUpdate := editorRequest(http.MethodPut, "/api/editor/entries?category=personas&index=0",
		`{"category":"personas","entry":{"name":"Rin Kato","history":"New"}}`, groups)
	entryResponse := httptest.NewRecorder()
	editorEntriesHandler(entryResponse, entryUpdate)
	if entryResponse.Code != http.StatusOK || !strings.Contains(entryResponse.Body.String(), `"name":"Rin Kato"`) {
		t.Fatalf("update entry: %d %s", entryResponse.Code, entryResponse.Body.String())
	}
	persisted, err := readEditorEntries("personas")
	if err != nil || len(persisted) != 1 || !strings.Contains(string(persisted[0]), "Rin Kato") {
		t.Fatalf("entry update was not persisted: %v %s", err, persisted)
	}

	unknown := editorRequest(http.MethodPut, "/api/editor/categories/missing", category, groups)
	unknown.SetPathValue("slug", "missing")
	notFound := httptest.NewRecorder()
	editorCategoriesHandler(notFound, unknown)
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("unknown category update: %d %s", notFound.Code, notFound.Body.String())
	}
}

func TestEditorEntryUpdateCannotEditBundledEntries(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NEOTOKYO_DATA_DIR", dir)
	bundled := filepath.Join(dir, "spells", "spells.json")
	writeFile(t, bundled, `[{"name":"Bundled"}]`)
	request := editorRequest(http.MethodPut, "/api/editor/entries?category=spells&index=0",
		`{"category":"spells","entry":{"name":"Changed"}}`, "editor")
	recorder := httptest.NewRecorder()
	editorEntriesHandler(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected bundled index to be uneditable, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if contents, err := os.ReadFile(bundled); err != nil || string(contents) != `[{"name":"Bundled"}]` {
		t.Fatalf("bundled entries changed: %v %s", err, contents)
	}
}

func TestCustomCategoryEntryValidation(t *testing.T) {
	category, err := normalizeCustomCategory(CustomCategory{
		Title: "Corporations",
		Fields: []CustomField{{
			Key: "headquarters", Label: "Headquarters", Type: "text",
			Display: "paragraph", Size: "normal",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	normalized, _, err := validateCustomEntry([]byte(`{"name":"Valid","image":"https://example.test/icon.png"}`), category)
	if err != nil || !strings.Contains(string(normalized), `"image":"https://example.test/icon.png"`) {
		t.Fatalf("custom entry image was not retained: %v %s", err, normalized)
	}
	for _, entry := range []string{
		`{"name":"Valid","unknown":"rejected"}`,
		`{"name":"Valid","headquarters":["wrong type"]}`,
		`{"name":"   "}`,
		`{"name":"Valid","image":42}`,
	} {
		if _, _, err := validateCustomEntry([]byte(entry), category); err == nil {
			t.Errorf("expected %s to be rejected", entry)
		}
	}
}
