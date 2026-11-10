package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
)

type AbilityScores struct {
	Strength     int `json:"strength"`
	Dexterity    int `json:"dexterity"`
	Constitution int `json:"constitution"`
	Intelligence int `json:"intelligence"`
	Wisdom       int `json:"wisdom"`
	Charisma     int `json:"charisma"`
}

type CharacterEntry struct {
	Name       string `json:"name"`
	Detail     string `json:"detail,omitempty"`
	Proficient bool   `json:"proficient,omitempty"`
	Quantity   int    `json:"quantity,omitempty"`
	SpellClass string `json:"spell_class,omitempty"`
}

// CharacterClass records levels in an individual class, in the order acquired.
type CharacterClass struct {
	Name     string `json:"name"`
	Subclass string `json:"subclass,omitempty"`
	Level    int    `json:"level"`
}

// ResourceUse stores the remaining uses of a class resource; the maximum comes from class and level.
type ResourceUse struct {
	ID      string `json:"id"`
	Current int    `json:"current"`
}

type SpellSlot struct {
	Level   int `json:"level"`
	Current int `json:"current"`
	Max     int `json:"max"`
}

var abilityNames = map[string]bool{"strength": true, "dexterity": true, "constitution": true, "intelligence": true, "wisdom": true, "charisma": true}

type CharacterSheet struct {
	ID                 string           `json:"id"`
	Name               string           `json:"name"`
	Species            string           `json:"species,omitempty"`
	Class              string           `json:"class,omitempty"`
	Subclass           string           `json:"subclass,omitempty"`
	Classes            []CharacterClass `json:"classes,omitempty"`
	Background         string           `json:"background,omitempty"`
	Level              int              `json:"level"`
	ExperiencePoints   int              `json:"experience_points,omitempty"`
	Alignment          string           `json:"alignment,omitempty"`
	AbilityScores      AbilityScores    `json:"ability_scores"`
	SaveProficiencies  []string         `json:"save_proficiencies,omitempty"`
	SpellSlots         []SpellSlot      `json:"spell_slots,omitempty"`
	PactSlots          *SpellSlot       `json:"pact_slots,omitempty"`
	ClassResources     []ResourceUse    `json:"class_resources,omitempty"`
	SkillProficiencies []string         `json:"skill_proficiencies,omitempty"`
	ArmorClass         int              `json:"armor_class,omitempty"`
	Initiative         int              `json:"initiative,omitempty"`
	Speed              int              `json:"speed,omitempty"`
	CurrentHP          int              `json:"current_hp,omitempty"`
	MaxHP              int              `json:"max_hp,omitempty"`
	TemporaryHP        int              `json:"temporary_hp,omitempty"`
	Conditions         []string         `json:"conditions,omitempty"`
	CyberStrain        int              `json:"cyber_strain,omitempty"`
	Yen                int              `json:"yen,omitempty"`
	Portrait           string           `json:"portrait,omitempty"`
	HitDice            string           `json:"hit_dice,omitempty"`
	Weapons            []CharacterEntry `json:"weapons,omitempty"`
	Spells             []CharacterEntry `json:"spells,omitempty"`
	Feats              []CharacterEntry `json:"feats,omitempty"`
	Equipment          []CharacterEntry `json:"equipment,omitempty"`
	Notes              string           `json:"notes,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}

type characterFile struct {
	Users map[string][]CharacterSheet `json:"users"`
}

type characterStore struct {
	path string
	mu   sync.Mutex
}

func newCharacterStore(path string) *characterStore {
	if strings.TrimSpace(path) == "" {
		path = "data/characters.json"
	}
	return &characterStore{path: path}
}

var savedCharacters = newCharacterStore(os.Getenv("CHARACTER_DATA_FILE"))

const guestCharacterCookie = "5etools_character_guest"

func (s *characterStore) list(user string) ([]CharacterSheet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.read()
	if err != nil {
		return nil, err
	}
	characters := file.Users[user]
	if characters == nil {
		characters = []CharacterSheet{}
	}
	return characters, nil
}

func (s *characterStore) create(user string, character CharacterSheet) (CharacterSheet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.read()
	if err != nil {
		return CharacterSheet{}, err
	}
	id, err := newCharacterID()
	if err != nil {
		return CharacterSheet{}, err
	}
	now := time.Now().UTC()
	character.ID = id
	character.CreatedAt = now
	character.UpdatedAt = now
	file.Users[user] = append(file.Users[user], character)
	if err := s.write(file); err != nil {
		return CharacterSheet{}, err
	}
	return character, nil
}

func (s *characterStore) update(user, id string, character CharacterSheet) (CharacterSheet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.read()
	if err != nil {
		return CharacterSheet{}, err
	}
	characters := file.Users[user]
	for index := range characters {
		if characters[index].ID != id {
			continue
		}
		character.ID = id
		character.CreatedAt = characters[index].CreatedAt
		character.UpdatedAt = time.Now().UTC()
		characters[index] = character
		file.Users[user] = characters
		if err := s.write(file); err != nil {
			return CharacterSheet{}, err
		}
		return character, nil
	}
	return CharacterSheet{}, os.ErrNotExist
}

func (s *characterStore) delete(user, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.read()
	if err != nil {
		return err
	}
	characters := file.Users[user]
	for index := range characters {
		if characters[index].ID != id {
			continue
		}
		file.Users[user] = append(characters[:index], characters[index+1:]...)
		return s.write(file)
	}
	return os.ErrNotExist
}

// find returns one sheet owned by user.
func (s *characterStore) find(user, id string) (CharacterSheet, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := s.read()
	if err != nil {
		return CharacterSheet{}, false
	}
	for _, character := range file.Users[user] {
		if character.ID == id {
			return character, true
		}
	}
	return CharacterSheet{}, false
}

// patch applies change to one sheet under the store lock and saves the result.
func (s *characterStore) patch(user, id string, change func(*CharacterSheet)) (CharacterSheet, error) {
	return s.patchChecked(user, id, func(sheet *CharacterSheet) error {
		change(sheet)
		return nil
	})
}

func (s *characterStore) patchChecked(user, id string, change func(*CharacterSheet) error) (CharacterSheet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := s.read()
	if err != nil {
		return CharacterSheet{}, err
	}
	characters := file.Users[user]
	for index := range characters {
		if characters[index].ID != id {
			continue
		}
		if err := change(&characters[index]); err != nil {
			return CharacterSheet{}, err
		}
		characters[index].UpdatedAt = time.Now().UTC()
		if err := s.write(file); err != nil {
			return CharacterSheet{}, err
		}
		return characters[index], nil
	}
	return CharacterSheet{}, os.ErrNotExist
}

// adopt moves every sheet owned by from onto to. It is a no-op when from has none.
func (s *characterStore) adopt(from, to string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.read()
	if err != nil {
		return err
	}
	moving := file.Users[from]
	if len(moving) == 0 {
		return nil
	}
	file.Users[to] = append(file.Users[to], moving...)
	delete(file.Users, from)
	return s.write(file)
}

func (s *characterStore) read() (characterFile, error) {
	file := characterFile{Users: make(map[string][]CharacterSheet)}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return file, nil
	}
	if err != nil {
		return characterFile{}, err
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return characterFile{}, fmt.Errorf("decode character store: %w", err)
	}
	if file.Users == nil {
		file.Users = make(map[string][]CharacterSheet)
	}
	return file, nil
}

func (s *characterStore) write(file characterFile) error {
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	directory := filepath.Dir(s.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".characters-*.tmp")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, s.path)
}

func newCharacterID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func charactersPageHandler(w http.ResponseWriter, r *http.Request) {
	user, guest := characterUser(w, r)
	if user == "" {
		http.Error(w, "Authentication required. Sign in through Authelia to manage characters.", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	displayUser := user
	if guest {
		displayUser = "Temporary guest access"
	}
	if err := tmpl.ExecuteTemplate(w, "characters", map[string]any{"User": displayUser, "Guest": guest}); err != nil {
		log.Println("character page template error:", err)
		http.Error(w, "Could not render character sheet", http.StatusInternalServerError)
	}
}

func charactersAPIHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := characterUser(w, r)
	if user == "" {
		writeCharacterError(w, http.StatusUnauthorized, "Sign in through Authelia to manage characters.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")

	switch r.Method {
	case http.MethodGet:
		characters, err := savedCharacters.list(user)
		if err != nil {
			writeCharacterError(w, http.StatusInternalServerError, "Could not read saved characters.")
			return
		}
		writeCharacterJSON(w, http.StatusOK, characters)
	case http.MethodPost:
		if !sameOriginMutation(w, r) {
			return
		}
		character, ok := decodeCharacter(w, r)
		if !ok {
			return
		}
		if err := validateCharacter(character); err != nil {
			writeCharacterError(w, http.StatusBadRequest, err.Error())
			return
		}
		created, err := savedCharacters.create(user, character)
		if err != nil {
			writeCharacterError(w, http.StatusInternalServerError, "Could not save character.")
			return
		}
		writeCharacterJSON(w, http.StatusCreated, created)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeCharacterError(w, http.StatusMethodNotAllowed, "Method not allowed.")
	}
}

func characterAPIHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := characterUser(w, r)
	if user == "" {
		writeCharacterError(w, http.StatusUnauthorized, "Sign in through Authelia to manage characters.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	id := r.PathValue("id")

	switch r.Method {
	case http.MethodPut:
		if !sameOriginMutation(w, r) {
			return
		}
		character, ok := decodeCharacter(w, r)
		if !ok {
			return
		}
		if err := validateCharacter(character); err != nil {
			writeCharacterError(w, http.StatusBadRequest, err.Error())
			return
		}
		updated, err := savedCharacters.update(user, id, character)
		if errors.Is(err, os.ErrNotExist) {
			writeCharacterError(w, http.StatusNotFound, "Character not found.")
			return
		}
		if err != nil {
			writeCharacterError(w, http.StatusInternalServerError, "Could not save character.")
			return
		}
		campaigns.characterChanged(user, updated)
		writeCharacterJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		if !sameOriginMutation(w, r) {
			return
		}
		if err := savedCharacters.delete(user, id); errors.Is(err, os.ErrNotExist) {
			writeCharacterError(w, http.StatusNotFound, "Character not found.")
			return
		} else if err != nil {
			writeCharacterError(w, http.StatusInternalServerError, "Could not delete character.")
			return
		}
		campaigns.characterDeleted(user, id)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "PUT, DELETE")
		writeCharacterError(w, http.StatusMethodNotAllowed, "Method not allowed.")
	}
}

// characterHPHandler saves only the hit point fields so HP edits can be pushed live to campaigns.
func characterHPHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	var body struct {
		CurrentHP int `json:"current_hp"`
		MaxHP     int `json:"max_hp"`
	}
	if !decodeCampaignBody(w, r, &body) {
		return
	}
	if body.CurrentHP < 0 || body.CurrentHP > 9999 || body.MaxHP < 0 || body.MaxHP > 9999 {
		writeCharacterError(w, http.StatusBadRequest, "Hit point values are outside the supported range.")
		return
	}
	updated, err := savedCharacters.patch(user, r.PathValue("id"), func(sheet *CharacterSheet) {
		sheet.CurrentHP, sheet.MaxHP = body.CurrentHP, body.MaxHP
	})
	if errors.Is(err, os.ErrNotExist) {
		writeCharacterError(w, http.StatusNotFound, "Character not found.")
		return
	}
	if err != nil {
		writeCharacterError(w, http.StatusInternalServerError, "Could not save character.")
		return
	}
	campaigns.characterChanged(user, updated)
	w.WriteHeader(http.StatusNoContent)
}

func characterUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	user := strings.TrimSpace(r.Header.Get("Remote-User"))
	if user == "" || len(user) > 256 || strings.IndexFunc(user, unicode.IsControl) >= 0 {
		user = ""
	} else {
		adoptGuestCharacters(w, r, user)
		return user, false
	}
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("CHARACTER_GUEST_ACCESS")), "true") {
		return "", false
	}

	if cookie, err := r.Cookie(guestCharacterCookie); err == nil && validGuestID(cookie.Value) {
		return "guest:" + cookie.Value, true
	}
	guestID, err := newCharacterID()
	if err != nil {
		return "", false
	}
	http.SetCookie(w, &http.Cookie{
		Name:     guestCharacterCookie,
		Value:    guestID,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 365,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	return "guest:" + guestID, true
}

// adoptGuestCharacters moves sheets made as a guest in this browser into the
// account that just signed in, then retires the guest cookie.
func adoptGuestCharacters(w http.ResponseWriter, r *http.Request, user string) {
	cookie, err := r.Cookie(guestCharacterCookie)
	if err != nil || !validGuestID(cookie.Value) {
		return
	}
	if err := savedCharacters.adopt("guest:"+cookie.Value, user); err != nil {
		log.Println("adopt guest characters:", err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: guestCharacterCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
}

func validGuestID(value string) bool {
	if len(value) != 32 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func sameOriginMutation(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	parsedOrigin, err := url.Parse(origin)
	if origin == "" || err != nil || parsedOrigin.Host == "" || !strings.EqualFold(parsedOrigin.Host, r.Host) {
		writeCharacterError(w, http.StatusForbidden, "Invalid request origin.")
		return false
	}
	return true
}

func decodeCharacter(w http.ResponseWriter, r *http.Request) (CharacterSheet, bool) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeCharacterError(w, http.StatusUnsupportedMediaType, "Expected application/json.")
		return CharacterSheet{}, false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var character CharacterSheet
	if err := decoder.Decode(&character); err != nil {
		writeCharacterError(w, http.StatusBadRequest, "Invalid character data.")
		return CharacterSheet{}, false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeCharacterError(w, http.StatusBadRequest, "Request must contain one JSON object.")
		return CharacterSheet{}, false
	}
	return character, true
}

func validateCharacter(character CharacterSheet) error {
	character.Name = strings.TrimSpace(character.Name)
	if character.Name == "" || len(character.Name) > 100 {
		return errors.New("character name is required and must be at most 100 characters")
	}
	if character.Level < 1 || character.Level > 20 {
		return errors.New("level must be between 1 and 20")
	}
	if len(character.Classes) > 20 {
		return errors.New("too many classes")
	}
	if len(character.Classes) > 0 {
		total := 0
		names := make(map[string]bool)
		for _, class := range character.Classes {
			name := strings.TrimSpace(class.Name)
			if name == "" || len(name) > 150 || len(class.Subclass) > 150 || class.Level < 1 || class.Level > 20 {
				return errors.New("invalid class progression")
			}
			key := strings.ToLower(name)
			if names[key] {
				return errors.New("class progression contains a duplicate class")
			}
			names[key] = true
			total += class.Level
		}
		if total != character.Level || character.Class != character.Classes[0].Name || character.Subclass != character.Classes[0].Subclass {
			return errors.New("class progression must match the total level and primary class")
		}
	}
	if character.ExperiencePoints < 0 || character.ExperiencePoints > 10000000 {
		return errors.New("experience points are outside the supported range")
	}
	if character.Yen < 0 || character.Yen > 999999999 {
		return errors.New("yen is outside the supported range")
	}
	scores := []int{character.AbilityScores.Strength, character.AbilityScores.Dexterity, character.AbilityScores.Constitution, character.AbilityScores.Intelligence, character.AbilityScores.Wisdom, character.AbilityScores.Charisma}
	for _, score := range scores {
		if score < 1 || score > 30 {
			return errors.New("ability scores must be between 1 and 30")
		}
	}
	for _, ability := range character.SaveProficiencies {
		if !abilityNames[ability] {
			return errors.New("unknown saving throw proficiency")
		}
	}
	if len(character.SkillProficiencies) > 40 {
		return errors.New("too many skill proficiencies")
	}
	for _, skill := range character.SkillProficiencies {
		if len(skill) == 0 || len(skill) > 80 {
			return errors.New("invalid skill proficiency")
		}
	}
	if len(character.SpellSlots) > 9 {
		return errors.New("spell slots may cover at most 9 spell levels")
	}
	for _, slot := range character.SpellSlots {
		if slot.Level < 1 || slot.Level > 9 || slot.Current < 0 || slot.Max < 0 || slot.Current > 99 || slot.Max > 99 {
			return errors.New("spell slot values are outside the supported range")
		}
	}
	if slot := character.PactSlots; slot != nil && (slot.Level < 1 || slot.Level > 5 || slot.Current < 0 || slot.Max < 0 || slot.Current > slot.Max || slot.Max > 99) {
		return errors.New("pact slot values are outside the supported range")
	}
	if len(character.ClassResources) > 20 {
		return errors.New("too many class resources")
	}
	for _, resource := range character.ClassResources {
		if len(resource.ID) == 0 || len(resource.ID) > 60 || resource.Current < 0 || resource.Current > 100 {
			return errors.New("class resource values are outside the supported range")
		}
	}
	if character.MaxHP < 0 || character.MaxHP > 9999 || character.CurrentHP < 0 || character.CurrentHP > 9999 || character.TemporaryHP < 0 || character.TemporaryHP > 9999 {
		return errors.New("hit point values are outside the supported range")
	}
	if err := validateSharedFields(character.Conditions, character.CyberStrain, character.Portrait); err != nil {
		return err
	}
	if len(character.Notes) > 20000 {
		return errors.New("notes must be at most 20000 characters")
	}
	for _, entries := range [][]CharacterEntry{character.Weapons, character.Spells, character.Feats, character.Equipment} {
		if len(entries) > 100 {
			return errors.New("each character list may contain at most 100 entries")
		}
		for _, entry := range entries {
			if len(entry.Name) > 150 || len(entry.Detail) > 2000 || len(entry.SpellClass) > 150 || entry.Quantity < 0 || entry.Quantity > 9999 {
				return errors.New("entry names must be at most 150 characters, details at most 2000 characters, and quantities between 0 and 9999")
			}
		}
	}
	if character.Yen < 0 || character.Yen > maxYen {
		return errors.New("yen is outside the supported range")
	}
	return nil
}

func writeCharacterJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeCharacterError(w http.ResponseWriter, status int, message string) {
	writeCharacterJSON(w, status, map[string]string{"error": message})
}
