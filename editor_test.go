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
	recorder := httptest.NewRecorder()
	editorPageHandler(recorder, editorRequest(http.MethodGet, "/editor", "", "editor"))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `value="spells"`) {
		t.Fatalf("status %d", recorder.Code)
	}
}
