package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validCharacter(name string) CharacterSheet {
	return CharacterSheet{
		Name:  name,
		Level: 3,
		AbilityScores: AbilityScores{
			Strength: 12, Dexterity: 14, Constitution: 13,
			Intelligence: 10, Wisdom: 11, Charisma: 16,
		},
		Weapons: []CharacterEntry{{Name: "Longsword", Detail: "1d8 slashing"}},
		Spells:  []CharacterEntry{{Name: "Shield", Detail: "1st level"}},
		Feats:   []CharacterEntry{{Name: "Alert"}},
	}
}

func TestCharacterStorePersistsAndScopesByUser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "characters.json")
	store := newCharacterStore(path)
	created, err := store.create("alice", validCharacter("Aster"))
	if err != nil {
		t.Fatalf("create character: %v", err)
	}

	freshStore := newCharacterStore(path)
	alice, err := freshStore.list("alice")
	if err != nil {
		t.Fatalf("load persisted characters: %v", err)
	}
	if len(alice) != 1 || alice[0].ID != created.ID || alice[0].Weapons[0].Name != "Longsword" {
		t.Fatalf("persisted data = %#v", alice)
	}

	bob, err := freshStore.list("bob")
	if err != nil {
		t.Fatalf("list other user's characters: %v", err)
	}
	if len(bob) != 0 {
		t.Fatalf("other user received characters: %#v", bob)
	}
	if _, err := freshStore.update("bob", created.ID, validCharacter("Stolen")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cross-user update error = %v, want not found", err)
	}
	if err := freshStore.delete("bob", created.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cross-user delete error = %v, want not found", err)
	}
}

func TestCharacterAPIRequiresAutheliaIdentity(t *testing.T) {
	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/characters"},
		{method: http.MethodGet, path: "/api/characters"},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(test.method, test.path, nil)
		setupRouter().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want %d", test.method, test.path, recorder.Code, http.StatusUnauthorized)
		}
	}
}

func TestCharacterPageRendersForAutheliaUser(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/characters", nil)
	request.Header.Set("Remote-User", "alice")
	recorder := httptest.NewRecorder()
	setupRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("character page status = %d: %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("character page content-type = %q", got)
	}
	for _, expected := range []string{"Character Sheets", "Signed in as alice", "Save as PDF", "Ability scores", "Weapons", "Spells", "Feats", "Equipment"} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Errorf("character page missing %q", expected)
		}
	}
}

func TestCharacterGuestModeUsesPersistentIsolatedCookie(t *testing.T) {
	t.Setenv("CHARACTER_GUEST_ACCESS", "true")
	previousStore := savedCharacters
	savedCharacters = newCharacterStore(filepath.Join(t.TempDir(), "characters.json"))
	t.Cleanup(func() { savedCharacters = previousStore })
	mux := setupRouter()

	pageRequest := httptest.NewRequest(http.MethodGet, "/characters", nil)
	pageRecorder := httptest.NewRecorder()
	mux.ServeHTTP(pageRecorder, pageRequest)
	if pageRecorder.Code != http.StatusOK {
		t.Fatalf("guest page status = %d: %s", pageRecorder.Code, pageRecorder.Body.String())
	}
	if !strings.Contains(pageRecorder.Body.String(), "Temporary guest access is enabled") {
		t.Fatal("guest page is missing temporary-mode notice")
	}
	var guestCookie *http.Cookie
	for _, cookie := range pageRecorder.Result().Cookies() {
		if cookie.Name == guestCharacterCookie {
			guestCookie = cookie
			break
		}
	}
	if guestCookie == nil || !guestCookie.HttpOnly || !guestCookie.Secure {
		t.Fatalf("guest identity cookie missing expected security attributes: %#v", guestCookie)
	}

	characterBody, err := json.Marshal(validCharacter("Guest Runner"))
	if err != nil {
		t.Fatal(err)
	}
	createRequest := httptest.NewRequest(http.MethodPost, "/api/characters", bytes.NewReader(characterBody))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Origin", "https://game.example.test")
	createRequest.Host = "game.example.test"
	createRequest.AddCookie(guestCookie)
	createRecorder := httptest.NewRecorder()
	mux.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("guest create status = %d: %s", createRecorder.Code, createRecorder.Body.String())
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/characters", nil)
	listRequest.AddCookie(guestCookie)
	listRecorder := httptest.NewRecorder()
	mux.ServeHTTP(listRecorder, listRequest)
	var guestCharacters []CharacterSheet
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &guestCharacters); err != nil {
		t.Fatalf("decode guest characters: %v", err)
	}
	if len(guestCharacters) != 1 || guestCharacters[0].Name != "Guest Runner" {
		t.Fatalf("guest's saved character was not restored: %#v", guestCharacters)
	}

	otherBrowserRequest := httptest.NewRequest(http.MethodGet, "/api/characters", nil)
	otherBrowserRecorder := httptest.NewRecorder()
	mux.ServeHTTP(otherBrowserRecorder, otherBrowserRequest)
	if otherBrowserRecorder.Code != http.StatusOK || strings.Contains(otherBrowserRecorder.Body.String(), "Guest Runner") {
		t.Fatalf("guest character leaked to another browser: status=%d body=%s", otherBrowserRecorder.Code, otherBrowserRecorder.Body.String())
	}
}

func TestAutheliaIdentityTakesPrecedenceOverGuestCookie(t *testing.T) {
	t.Setenv("CHARACTER_GUEST_ACCESS", "true")
	request := httptest.NewRequest(http.MethodGet, "/characters", nil)
	request.Header.Set("Remote-User", "alice")
	request.AddCookie(&http.Cookie{Name: guestCharacterCookie, Value: "0123456789abcdef0123456789abcdef"})
	user, guest := characterUser(httptest.NewRecorder(), request)
	if user != "alice" || guest {
		t.Fatalf("identity = %q guest=%v, want Authelia alice", user, guest)
	}
}

func TestCharacterAPICRUDIsUserScoped(t *testing.T) {
	previousStore := savedCharacters
	savedCharacters = newCharacterStore(filepath.Join(t.TempDir(), "characters.json"))
	t.Cleanup(func() { savedCharacters = previousStore })
	mux := setupRouter()

	body, err := json.Marshal(validCharacter("Aster"))
	if err != nil {
		t.Fatal(err)
	}
	create := httptest.NewRequest(http.MethodPost, "/api/characters", bytes.NewReader(body))
	create.Header.Set("Content-Type", "application/json")
	create.Header.Set("Origin", "https://game.example.test")
	create.Header.Set("Remote-User", "alice")
	create.Host = "game.example.test"
	createdRecorder := httptest.NewRecorder()
	mux.ServeHTTP(createdRecorder, create)
	if createdRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", createdRecorder.Code, createdRecorder.Body.String())
	}
	var created CharacterSheet
	if err := json.Unmarshal(createdRecorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created character: %v", err)
	}
	if created.ID == "" || created.CreatedAt.IsZero() {
		t.Fatalf("created response missing ID or creation time: %#v", created)
	}

	listForBob := httptest.NewRequest(http.MethodGet, "/api/characters", nil)
	listForBob.Header.Set("Remote-User", "bob")
	listRecorder := httptest.NewRecorder()
	mux.ServeHTTP(listRecorder, listForBob)
	var bobCharacters []CharacterSheet
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &bobCharacters); err != nil {
		t.Fatalf("decode other user's list: %v", err)
	}
	if len(bobCharacters) != 0 {
		t.Fatalf("other user listed characters: %#v", bobCharacters)
	}

	updatedBody, _ := json.Marshal(validCharacter("Aster Updated"))
	update := httptest.NewRequest(http.MethodPut, "/api/characters/"+created.ID, bytes.NewReader(updatedBody))
	update.Header.Set("Content-Type", "application/json")
	update.Header.Set("Origin", "https://game.example.test")
	update.Header.Set("Remote-User", "bob")
	update.Host = "game.example.test"
	updateRecorder := httptest.NewRecorder()
	mux.ServeHTTP(updateRecorder, update)
	if updateRecorder.Code != http.StatusNotFound {
		t.Fatalf("cross-user update status = %d, want %d", updateRecorder.Code, http.StatusNotFound)
	}

	ownerUpdate := httptest.NewRequest(http.MethodPut, "/api/characters/"+created.ID, bytes.NewReader(updatedBody))
	ownerUpdate.Header.Set("Content-Type", "application/json")
	ownerUpdate.Header.Set("Origin", "https://game.example.test")
	ownerUpdate.Header.Set("Remote-User", "alice")
	ownerUpdate.Host = "game.example.test"
	updateRecorder = httptest.NewRecorder()
	mux.ServeHTTP(updateRecorder, ownerUpdate)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("owner update status = %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/characters/"+created.ID, nil)
	deleteRequest.Header.Set("Origin", "https://game.example.test")
	deleteRequest.Header.Set("Remote-User", "alice")
	deleteRequest.Host = "game.example.test"
	deleteRecorder := httptest.NewRecorder()
	mux.ServeHTTP(deleteRecorder, deleteRequest)
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestCharacterMutationRejectsInvalidOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/characters", bytes.NewReader([]byte(`{"name":"Aster","level":1}`)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Remote-User", "alice")
	recorder := httptest.NewRecorder()
	charactersAPIHandler(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("missing Origin status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestAdoptGuestCharacters(t *testing.T) {
	store := newCharacterStore(filepath.Join(t.TempDir(), "characters.json"))
	if _, err := store.create("guest:abc", CharacterSheet{Name: "Kira", Level: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.create("neo", CharacterSheet{Name: "Existing", Level: 1}); err != nil {
		t.Fatal(err)
	}
	if err := store.adopt("guest:abc", "neo"); err != nil {
		t.Fatal(err)
	}
	owned, _ := store.list("neo")
	left, _ := store.list("guest:abc")
	if len(owned) != 2 || len(left) != 0 {
		t.Fatalf("adopt moved wrong sheets: owned=%d left=%d", len(owned), len(left))
	}
}

func TestAccountEndpoint(t *testing.T) {
	t.Setenv("AUTHELIA_URL", "https://auth.example.com/")
	t.Setenv("CHARACTER_GUEST_ACCESS", "")
	request := httptest.NewRequest(http.MethodGet, "/api/account", nil)
	request.Header.Set("Remote-User", "neo")
	request.Header.Set("Referer", "http://example.com/characters")
	recorder := httptest.NewRecorder()
	accountHandler(recorder, request)
	var info AccountInfo
	if err := json.Unmarshal(recorder.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if !info.SignedIn || info.User != "neo" || !strings.HasPrefix(info.LogoutURL, "https://auth.example.com/logout?rd=") {
		t.Fatalf("unexpected signed-in info: %+v", info)
	}

	anonymous := httptest.NewRecorder()
	accountHandler(anonymous, httptest.NewRequest(http.MethodGet, "/api/account", nil))
	info = AccountInfo{}
	_ = json.Unmarshal(anonymous.Body.Bytes(), &info)
	if info.SignedIn || info.LoginURL != "https://auth.example.com/" {
		t.Fatalf("unexpected anonymous info: %+v", info)
	}
}
