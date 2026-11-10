package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	errCampaignNotFound = errors.New("campaign not found")
	errCampaignForbid   = errors.New("not allowed")
	errCampaignInvalid  = errors.New("invalid request")
)

type campaignStore struct {
	path      string
	mu        sync.Mutex
	campaigns map[string]*Campaign
	now       func() time.Time
}

func newCampaignStore(path string) *campaignStore {
	if strings.TrimSpace(path) == "" {
		path = "data/campaigns.json"
	}
	store := &campaignStore{path: path, campaigns: map[string]*Campaign{}, now: time.Now}
	store.load()
	return store
}

var campaigns = newCampaignStore(os.Getenv("CAMPAIGN_DATA_FILE"))

// load reads persisted campaigns. A damaged file is kept aside rather than overwritten.
func (s *campaignStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var list []*Campaign
	if err := json.Unmarshal(data, &list); err != nil {
		_ = os.Rename(s.path, s.path+".corrupt")
		return
	}
	migrated := false
	for _, campaign := range list {
		if campaign != nil && campaign.ID != "" {
			previous := campaign.NextMemberNumber
			for _, member := range campaign.Members {
				if member.SendAlias == "" {
					migrated = true
				}
			}
			ensureMemberAliases(campaign)
			if previous != campaign.NextMemberNumber {
				migrated = true
			}
			s.campaigns[campaign.ID] = campaign
		}
	}
	if migrated {
		if err := s.persistLocked(); err != nil {
			log.Println("migrate campaign member send aliases:", err)
		}
	}
}

func (s *campaignStore) persistLocked() error {
	list := make([]*Campaign, 0, len(s.campaigns))
	for _, campaign := range s.campaigns {
		list = append(list, campaign)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	directory := filepath.Dir(s.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".campaigns-*.tmp")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
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
	return os.Rename(name, s.path)
}

func cloneCampaign(c *Campaign) Campaign {
	copied := *c
	copied.Members = make([]CampaignMember, len(c.Members))
	for i, member := range c.Members {
		member.Conditions = append([]string{}, member.Conditions...)
		if member.Initiative != nil {
			value := *member.Initiative
			member.Initiative = &value
		}
		copied.Members[i] = member
	}
	copied.Rolls = append([]CampaignRoll{}, c.Rolls...)
	copied.DropboxItems = append([]LootItem{}, c.DropboxItems...)
	copied.Deliveries = make([]CampaignDelivery, len(c.Deliveries))
	for i, delivery := range c.Deliveries {
		delivery.Items = append([]LootItem{}, delivery.Items...)
		copied.Deliveries[i] = delivery
	}
	copied.NPCs = make([]CampaignNPC, len(c.NPCs))
	for i, npc := range c.NPCs {
		npc.Conditions = append([]string{}, npc.Conditions...)
		if npc.Initiative != nil {
			value := *npc.Initiative
			npc.Initiative = &value
		}
		copied.NPCs[i] = npc
	}
	return copied
}

func (c *Campaign) isMember(user string) bool {
	if c.Owner == user {
		return true
	}
	for _, member := range c.Members {
		if member.Owner == user {
			return true
		}
	}
	return false
}

func (c *Campaign) memberIndex(characterID string) int {
	for i, member := range c.Members {
		if member.CharacterID == characterID {
			return i
		}
	}
	return -1
}

func (c *Campaign) touch(now time.Time) {
	c.LastActivity = now
	c.Version++
}

// regatherLocked re-reads every member from its character sheet and clears the
// dice history. Members whose sheet no longer exists are dropped.
func (s *campaignStore) regatherLocked(c *Campaign, now time.Time) {
	kept := c.Members[:0]
	for _, member := range c.Members {
		if sheet, ok := savedCharacters.find(member.Owner, member.CharacterID); ok {
			kept = append(kept, refreshMemberFromSheet(member, sheet))
		}
	}
	c.Members = kept
	c.Rolls = nil
	c.LastRegather = now
	c.touch(now)
}

// refreshLocked applies expiry rules and reports whether the campaign changed or was purged.
func (s *campaignStore) refreshLocked(c *Campaign) (changed, purged bool) {
	now := s.now()
	idle := now.Sub(c.LastActivity)
	if idle >= campaignPurgeAfter {
		delete(s.campaigns, c.ID)
		return true, true
	}
	if idle >= campaignIdleTTL {
		s.regatherLocked(c, now)
		return true, false
	}
	return false, false
}

func newCampaignToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func (s *campaignStore) create(owner, name string) (Campaign, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return Campaign{}, fmt.Errorf("%w: campaign name is required and must be at most 80 characters", errCampaignInvalid)
	}
	id, err := newCharacterID()
	if err != nil {
		return Campaign{}, err
	}
	token, err := newCampaignToken()
	if err != nil {
		return Campaign{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	owned := 0
	for _, existing := range s.campaigns {
		if existing.Owner == owner {
			owned++
		}
	}
	if owned >= maxCampaignsPerOwner {
		return Campaign{}, fmt.Errorf("%w: you can own at most %d campaigns", errCampaignInvalid, maxCampaignsPerOwner)
	}
	now := s.now().UTC()
	campaign := &Campaign{ID: id, Name: name, Owner: owner, JoinToken: token, Members: []CampaignMember{}, Rolls: []CampaignRoll{}, NPCs: []CampaignNPC{},
		DropboxItems: []LootItem{}, Deliveries: []CampaignDelivery{}, NextMemberNumber: 1,
		NextRollID: 1, Version: 1, CreatedAt: now, LastActivity: now, LastRegather: now}
	s.campaigns[id] = campaign
	if err := s.persistLocked(); err != nil {
		delete(s.campaigns, id)
		return Campaign{}, err
	}
	return cloneCampaign(campaign), nil
}

// get returns a copy of a campaign, applying expiry first.
func (s *campaignStore) get(id string) (Campaign, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	campaign, ok := s.campaigns[id]
	if !ok {
		return Campaign{}, false
	}
	if changed, purged := s.refreshLocked(campaign); changed {
		_ = s.persistLocked()
		if purged {
			return Campaign{}, false
		}
	}
	return cloneCampaign(campaign), true
}

func (s *campaignStore) byToken(token string) (Campaign, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, campaign := range s.campaigns {
		if token != "" && campaign.JoinToken == token {
			return cloneCampaign(campaign), true
		}
	}
	return Campaign{}, false
}

func (s *campaignStore) listFor(user string) []Campaign {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := []Campaign{}
	for _, campaign := range s.campaigns {
		if campaign.isMember(user) {
			result = append(result, cloneCampaign(campaign))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].LastActivity.After(result[j].LastActivity) })
	return result
}

// mutate runs change on a live campaign the caller may access, saves, and returns the updated copy.
func (s *campaignStore) mutate(id string, change func(*Campaign) error) (Campaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	campaign, ok := s.campaigns[id]
	if !ok {
		return Campaign{}, errCampaignNotFound
	}
	if changed, purged := s.refreshLocked(campaign); changed && purged {
		_ = s.persistLocked()
		return Campaign{}, errCampaignNotFound
	}
	backup := cloneCampaign(campaign)
	if err := change(campaign); err != nil {
		*campaign = backup
		return Campaign{}, err
	}
	campaign.touch(s.now().UTC())
	if err := s.persistLocked(); err != nil {
		*campaign = backup
		return Campaign{}, err
	}
	return cloneCampaign(campaign), nil
}

// join adds the user's chosen characters to the campaign identified by its join token.
func (s *campaignStore) join(token, user string, characterIDs []string) (Campaign, error) {
	found, ok := s.byToken(token)
	if !ok {
		return Campaign{}, errCampaignNotFound
	}
	if len(characterIDs) == 0 {
		return Campaign{}, fmt.Errorf("%w: choose at least one character", errCampaignInvalid)
	}
	return s.mutate(found.ID, func(c *Campaign) error {
		ensureMemberAliases(c)
		for _, characterID := range characterIDs {
			sheet, ok := savedCharacters.find(user, characterID)
			if !ok {
				return fmt.Errorf("%w: character not found", errCampaignInvalid)
			}
			if c.memberIndex(characterID) >= 0 {
				continue
			}
			if len(c.Members) >= maxCampaignMembers {
				return fmt.Errorf("%w: this campaign is full", errCampaignInvalid)
			}
			member := memberFromSheet(user, sheet)
			member.SendAlias = fmt.Sprintf("user%d", c.NextMemberNumber)
			c.NextMemberNumber++
			c.Members = append(c.Members, member)
		}
		return nil
	})
}

func (s *campaignStore) removeMember(id, actor, characterID string) (Campaign, error) {
	return s.mutate(id, func(c *Campaign) error {
		index := c.memberIndex(characterID)
		if index < 0 {
			return errCampaignNotFound
		}
		if actor != c.Owner && actor != c.Members[index].Owner {
			return errCampaignForbid
		}
		c.Members = append(c.Members[:index], c.Members[index+1:]...)
		return nil
	})
}

// MemberPatch holds the shared fields that may be edited from the campaign screen.
type MemberPatch struct {
	CurrentHP       *int      `json:"current_hp"`
	Conditions      *[]string `json:"conditions"`
	CyberStrain     *int      `json:"cyber_strain"`
	Initiative      *int      `json:"initiative"`
	ClearInitiative bool      `json:"clear_initiative"`
	Portrait        *string   `json:"portrait"`
}

func (p MemberPatch) validate() error {
	if p.CurrentHP != nil && (*p.CurrentHP < 0 || *p.CurrentHP > 9999) {
		return fmt.Errorf("%w: hit points are outside the supported range", errCampaignInvalid)
	}
	if p.Initiative != nil && (*p.Initiative < -20 || *p.Initiative > 50) {
		return fmt.Errorf("%w: initiative is outside the supported range", errCampaignInvalid)
	}
	conditions, strain, portrait := []string{}, 0, ""
	if p.Conditions != nil {
		conditions = *p.Conditions
	}
	if p.CyberStrain != nil {
		strain = *p.CyberStrain
	}
	if p.Portrait != nil {
		portrait = *p.Portrait
	}
	if err := validateSharedFields(conditions, strain, portrait); err != nil {
		return fmt.Errorf("%w: %s", errCampaignInvalid, err)
	}
	return nil
}

// updateMember lets a character's owner or the campaign creator edit shared fields.
// The change is written back to the owner's sheet so both views stay identical.
func (s *campaignStore) updateMember(id, actor, characterID string, patch MemberPatch) (Campaign, error) {
	if err := patch.validate(); err != nil {
		return Campaign{}, err
	}
	return s.mutate(id, func(c *Campaign) error {
		index := c.memberIndex(characterID)
		if index < 0 {
			return errCampaignNotFound
		}
		member := &c.Members[index]
		if actor != c.Owner && actor != member.Owner {
			return errCampaignForbid
		}
		sheet, err := savedCharacters.patch(member.Owner, member.CharacterID, func(sheet *CharacterSheet) {
			if patch.CurrentHP != nil {
				sheet.CurrentHP = *patch.CurrentHP
			}
			if patch.Conditions != nil {
				sheet.Conditions = cleanConditions(*patch.Conditions)
			}
			if patch.CyberStrain != nil {
				sheet.CyberStrain = *patch.CyberStrain
			}
			if patch.ClearInitiative {
				sheet.Initiative = 0
			} else if patch.Initiative != nil {
				sheet.Initiative = *patch.Initiative
			}
			if patch.Portrait != nil {
				sheet.Portrait = strings.TrimSpace(*patch.Portrait)
			}
		})
		if err != nil {
			return fmt.Errorf("could not save the character sheet: %w", err)
		}
		*member = refreshMemberFromSheet(*member, sheet)
		return nil
	})
}

func cleanConditions(conditions []string) []string {
	cleaned := []string{}
	for _, condition := range conditions {
		if condition = strings.TrimSpace(condition); condition != "" {
			cleaned = append(cleaned, condition)
		}
	}
	return cleaned
}

// RollInput is a client-side digital dice roll to record in shared history.
type RollInput struct {
	Label  string `json:"label"`
	Detail string `json:"detail"`
	Total  *int   `json:"total"`
	Note   string `json:"note"`
}

func (r RollInput) validate() error {
	if strings.TrimSpace(r.Label) == "" || len(r.Label) > 120 || len(r.Detail) > 300 || len(r.Note) > 60 {
		return fmt.Errorf("%w: roll text is missing or too long", errCampaignInvalid)
	}
	return nil
}

// recordRoll adds a roll to every campaign containing the character and returns their IDs.
func (s *campaignStore) recordRoll(owner, characterID string, roll RollInput) ([]string, error) {
	if err := roll.validate(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	var ids []string
	for id, campaign := range s.campaigns {
		if index := campaign.memberIndex(characterID); index >= 0 && campaign.Members[index].Owner == owner {
			ids = append(ids, id)
		}
	}
	s.mu.Unlock()
	var updated []string
	for _, id := range ids {
		_, err := s.mutate(id, func(c *Campaign) error {
			index := c.memberIndex(characterID)
			if index < 0 {
				return errCampaignNotFound
			}
			c.Rolls = append(c.Rolls, CampaignRoll{ID: c.NextRollID, At: s.now().UTC(), CharacterID: characterID, Player: owner, Character: c.Members[index].Name,
				Label: strings.TrimSpace(roll.Label), Detail: roll.Detail, Total: roll.Total, Note: roll.Note})
			c.NextRollID++
			if len(c.Rolls) > maxCampaignRolls {
				c.Rolls = c.Rolls[len(c.Rolls)-maxCampaignRolls:]
			}
			return nil
		})
		if err == nil {
			updated = append(updated, id)
		}
	}
	return updated, nil
}

func (s *campaignStore) remove(id, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	campaign, ok := s.campaigns[id]
	if !ok {
		return errCampaignNotFound
	}
	if campaign.Owner != actor {
		return errCampaignForbid
	}
	delete(s.campaigns, id)
	return s.persistLocked()
}

// sweep applies expiry to every campaign and returns the IDs that changed and the IDs purged.
func (s *campaignStore) sweep() (changedIDs, purgedIDs []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, campaign := range s.campaigns {
		if changed, purged := s.refreshLocked(campaign); changed {
			if purged {
				purgedIDs = append(purgedIDs, id)
			} else {
				changedIDs = append(changedIDs, id)
			}
		}
	}
	if len(changedIDs)+len(purgedIDs) > 0 {
		_ = s.persistLocked()
	}
	return changedIDs, purgedIDs
}

// characterChanged refreshes the snapshot of a saved sheet in every campaign it belongs to.
func (s *campaignStore) characterChanged(owner string, sheet CharacterSheet) {
	s.mu.Lock()
	var affected []string
	for id, campaign := range s.campaigns {
		if index := campaign.memberIndex(sheet.ID); index >= 0 && campaign.Members[index].Owner == owner {
			campaign.Members[index] = refreshMemberFromSheet(campaign.Members[index], sheet)
			campaign.touch(s.now().UTC())
			affected = append(affected, id)
		}
	}
	if len(affected) > 0 {
		_ = s.persistLocked()
	}
	s.mu.Unlock()
	for _, id := range affected {
		campaignHub.broadcast(id)
	}
}

// characterDeleted removes a deleted sheet from every campaign.
func (s *campaignStore) characterDeleted(owner, characterID string) {
	s.mu.Lock()
	var affected []string
	for id, campaign := range s.campaigns {
		if index := campaign.memberIndex(characterID); index >= 0 && campaign.Members[index].Owner == owner {
			campaign.Members = append(campaign.Members[:index], campaign.Members[index+1:]...)
			campaign.touch(s.now().UTC())
			affected = append(affected, id)
		}
	}
	if len(affected) > 0 {
		_ = s.persistLocked()
	}
	s.mu.Unlock()
	for _, id := range affected {
		campaignHub.broadcast(id)
	}
}

func (c *Campaign) npcIndex(id string) int {
	for i, npc := range c.NPCs {
		if npc.ID == id {
			return i
		}
	}
	return -1
}

// NPCInput creates or edits a non-player entity; nil fields are left unchanged on edit.
type NPCInput struct {
	Name            *string   `json:"name"`
	Entity          string    `json:"entity"` // only used when adding
	CurrentHP       *int      `json:"current_hp"`
	MaxHP           *int      `json:"max_hp"`
	Initiative      *int      `json:"initiative"`
	ClearInitiative bool      `json:"clear_initiative"`
	Conditions      *[]string `json:"conditions"`
}

func (n NPCInput) validate() error {
	if n.Name != nil && (strings.TrimSpace(*n.Name) == "" || len(*n.Name) > 80) {
		return fmt.Errorf("%w: name is required and must be at most 80 characters", errCampaignInvalid)
	}
	for _, hp := range []*int{n.CurrentHP, n.MaxHP} {
		if hp != nil && (*hp < 0 || *hp > 9999) {
			return fmt.Errorf("%w: hit points are outside the supported range", errCampaignInvalid)
		}
	}
	if len(n.Entity) > 80 {
		return fmt.Errorf("%w: entity name is too long", errCampaignInvalid)
	}
	if n.Initiative != nil && (*n.Initiative < -20 || *n.Initiative > 50) {
		return fmt.Errorf("%w: initiative is outside the supported range", errCampaignInvalid)
	}
	if n.Conditions != nil {
		if err := validateSharedFields(*n.Conditions, 0, ""); err != nil {
			return fmt.Errorf("%w: %s", errCampaignInvalid, err)
		}
	}
	return nil
}

func (n NPCInput) apply(npc *CampaignNPC) {
	if n.Name != nil {
		npc.Name = strings.TrimSpace(*n.Name)
	}
	if n.MaxHP != nil {
		npc.MaxHP = *n.MaxHP
	}
	if n.CurrentHP != nil {
		npc.CurrentHP = *n.CurrentHP
	}
	if n.ClearInitiative {
		npc.Initiative = nil
	} else if n.Initiative != nil {
		value := *n.Initiative
		npc.Initiative = &value
	}
	if n.Conditions != nil {
		npc.Conditions = cleanConditions(*n.Conditions)
	}
}

// addNPC lets the campaign creator add a non-player entity.
func (s *campaignStore) addNPC(id, actor string, input NPCInput) (Campaign, error) {
	if err := input.validate(); err != nil {
		return Campaign{}, err
	}
	if input.Name == nil {
		return Campaign{}, fmt.Errorf("%w: name is required", errCampaignInvalid)
	}
	npcID, err := newCampaignToken()
	if err != nil {
		return Campaign{}, err
	}
	return s.mutate(id, func(c *Campaign) error {
		if actor != c.Owner {
			return errCampaignForbid
		}
		if len(c.NPCs) >= maxCampaignNPCs {
			return fmt.Errorf("%w: this campaign has too many non-player entities", errCampaignInvalid)
		}
		npc := CampaignNPC{ID: npcID, Conditions: []string{}}
		input.apply(&npc)
		npc.Entity = strings.TrimSpace(input.Entity)
		if input.MaxHP != nil && input.CurrentHP == nil {
			npc.CurrentHP = npc.MaxHP
		}
		c.NPCs = append(c.NPCs, npc)
		return nil
	})
}

func (s *campaignStore) updateNPC(id, actor, npcID string, input NPCInput) (Campaign, error) {
	if err := input.validate(); err != nil {
		return Campaign{}, err
	}
	return s.mutate(id, func(c *Campaign) error {
		if actor != c.Owner {
			return errCampaignForbid
		}
		index := c.npcIndex(npcID)
		if index < 0 {
			return errCampaignNotFound
		}
		input.apply(&c.NPCs[index])
		return nil
	})
}

func (s *campaignStore) removeNPC(id, actor, npcID string) (Campaign, error) {
	return s.mutate(id, func(c *Campaign) error {
		if actor != c.Owner {
			return errCampaignForbid
		}
		index := c.npcIndex(npcID)
		if index < 0 {
			return errCampaignNotFound
		}
		c.NPCs = append(c.NPCs[:index], c.NPCs[index+1:]...)
		return nil
	})
}

func ensureMemberAliases(c *Campaign) {
	next := c.NextMemberNumber
	for _, member := range c.Members {
		if !strings.HasPrefix(member.SendAlias, "user") {
			continue
		}
		number, err := strconv.Atoi(strings.TrimPrefix(member.SendAlias, "user"))
		if err == nil && number >= next {
			next = number + 1
		}
	}
	if next < 1 {
		next = 1
	}
	for i := range c.Members {
		if c.Members[i].SendAlias == "" {
			c.Members[i].SendAlias = fmt.Sprintf("user%d", next)
			next++
		}
	}
	c.NextMemberNumber = next
}

func validateLootItems(items []LootItem, allowEmpty bool) ([]LootItem, error) {
	if len(items) > 100 || (!allowEmpty && len(items) == 0) {
		return nil, fmt.Errorf("%w: choose between 1 and 100 items", errCampaignInvalid)
	}
	result := make([]LootItem, 0, len(items))
	for _, item := range items {
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" || len(item.Name) > 150 || item.Quantity < 1 || item.Quantity > 9999 {
			return nil, fmt.Errorf("%w: item names must be at most 150 characters and quantities between 1 and 9999", errCampaignInvalid)
		}
		result = append(result, item)
	}
	return result, nil
}

func addLootItems(items *[]LootItem, added []LootItem) error {
	for _, item := range added {
		found := false
		for i := range *items {
			if strings.EqualFold((*items)[i].Name, item.Name) {
				if (*items)[i].Quantity+item.Quantity > 9999 {
					return fmt.Errorf("%w: item quantity is outside the supported range", errCampaignInvalid)
				}
				(*items)[i].Quantity += item.Quantity
				found = true
				break
			}
		}
		if !found {
			if len(*items) >= 100 {
				return fmt.Errorf("%w: the dropbox can hold at most 100 item types", errCampaignInvalid)
			}
			*items = append(*items, item)
		}
	}
	return nil
}

func (s *campaignStore) addToDropbox(id, actor string, items []LootItem, yen int) (Campaign, error) {
	validated, err := validateLootItems(items, true)
	if err != nil {
		return Campaign{}, err
	}
	if yen < 0 || yen > maxYen {
		return Campaign{}, fmt.Errorf("%w: yen amount is outside the supported range", errCampaignInvalid)
	}
	if len(validated) == 0 && yen == 0 {
		return Campaign{}, fmt.Errorf("%w: add at least one item or some Yen", errCampaignInvalid)
	}
	return s.mutate(id, func(c *Campaign) error {
		if c.Owner != actor {
			return errCampaignForbid
		}
		if err := addLootItems(&c.DropboxItems, validated); err != nil {
			return err
		}
		if c.DropboxYen+yen > maxYen {
			return fmt.Errorf("%w: dropbox Yen is outside the supported range", errCampaignInvalid)
		}
		c.DropboxYen += yen
		return nil
	})
}

func (s *campaignStore) clearDropbox(id, actor string) (Campaign, error) {
	return s.mutate(id, func(c *Campaign) error {
		if c.Owner != actor {
			return errCampaignForbid
		}
		c.DropboxItems = []LootItem{}
		c.DropboxYen = 0
		return nil
	})
}

func (s *campaignStore) sendDropbox(id, actor, alias string) (Campaign, error) {
	deliveryID, err := newCampaignToken()
	if err != nil {
		return Campaign{}, err
	}
	return s.mutate(id, func(c *Campaign) error {
		if c.Owner != actor {
			return errCampaignForbid
		}
		target := -1
		for i, member := range c.Members {
			if member.SendAlias == alias {
				target = i
				break
			}
		}
		if target < 0 {
			return errCampaignNotFound
		}
		recipientID := c.Members[target].CharacterID
		for _, delivery := range c.Deliveries {
			if delivery.RecipientCharacterID == recipientID {
				return fmt.Errorf("%w: this character already has a pending delivery", errCampaignInvalid)
			}
		}
		if len(c.DropboxItems) == 0 && c.DropboxYen == 0 {
			return fmt.Errorf("%w: the dropbox is empty", errCampaignInvalid)
		}
		c.Deliveries = append(c.Deliveries, CampaignDelivery{
			ID: deliveryID, RecipientCharacterID: recipientID, SenderName: "Game Master",
			Items: append([]LootItem{}, c.DropboxItems...), Yen: c.DropboxYen, CreatedAt: s.now().UTC(),
		})
		c.DropboxItems = []LootItem{}
		c.DropboxYen = 0
		return nil
	})
}

func removeInventoryItems(sheet *CharacterSheet, items []LootItem) error {
	for _, wanted := range items {
		remaining := wanted.Quantity
		for i := 0; i < len(sheet.Equipment) && remaining > 0; {
			entry := &sheet.Equipment[i]
			if !strings.EqualFold(strings.TrimSpace(entry.Name), wanted.Name) {
				i++
				continue
			}
			quantity := entry.Quantity
			if quantity == 0 {
				quantity = 1
			}
			take := quantity
			if take > remaining {
				take = remaining
			}
			quantity -= take
			remaining -= take
			if quantity == 0 {
				sheet.Equipment = append(sheet.Equipment[:i], sheet.Equipment[i+1:]...)
			} else {
				entry.Quantity = quantity
				i++
			}
		}
		if remaining > 0 {
			return fmt.Errorf("%w: not enough %q in Inventory", errCampaignInvalid, wanted.Name)
		}
	}
	return nil
}

func addInventoryItems(sheet *CharacterSheet, items []LootItem) error {
	for _, item := range items {
		found := false
		for i := range sheet.Equipment {
			if strings.EqualFold(strings.TrimSpace(sheet.Equipment[i].Name), item.Name) {
				quantity := sheet.Equipment[i].Quantity
				if quantity == 0 {
					quantity = 1
				}
				if quantity+item.Quantity > 9999 {
					return fmt.Errorf("%w: Inventory quantity is outside the supported range", errCampaignInvalid)
				}
				sheet.Equipment[i].Quantity = quantity + item.Quantity
				found = true
				break
			}
		}
		if !found {
			if len(sheet.Equipment) >= 100 {
				return fmt.Errorf("%w: Inventory can contain at most 100 item types", errCampaignInvalid)
			}
			sheet.Equipment = append(sheet.Equipment, CharacterEntry{Name: item.Name, Quantity: item.Quantity})
		}
	}
	return nil
}

func (s *campaignStore) sendTrade(id, actor, senderID, recipientAlias string, items []LootItem, yen int) (Campaign, error) {
	validated, err := validateLootItems(items, true)
	if err != nil {
		return Campaign{}, err
	}
	if yen < 0 || yen > maxYen || len(validated) == 0 && yen == 0 {
		return Campaign{}, fmt.Errorf("%w: add items or Yen within the supported range", errCampaignInvalid)
	}
	deliveryID, err := newCampaignToken()
	if err != nil {
		return Campaign{}, err
	}
	return s.mutate(id, func(c *Campaign) error {
		senderIndex := c.memberIndex(senderID)
		if senderIndex < 0 || c.Members[senderIndex].Owner != actor {
			return errCampaignForbid
		}
		recipientIndex := -1
		for i, member := range c.Members {
			if member.SendAlias == recipientAlias {
				recipientIndex = i
				break
			}
		}
		if recipientIndex < 0 || recipientIndex == senderIndex {
			return errCampaignNotFound
		}
		recipientID := c.Members[recipientIndex].CharacterID
		for _, delivery := range c.Deliveries {
			if delivery.RecipientCharacterID == recipientID {
				return fmt.Errorf("%w: this character already has a pending delivery", errCampaignInvalid)
			}
		}
		updated, err := savedCharacters.patchChecked(actor, senderID, func(sheet *CharacterSheet) error {
			if removeErr := removeInventoryItems(sheet, validated); removeErr != nil {
				return removeErr
			}
			if sheet.Yen < yen {
				return fmt.Errorf("%w: not enough Yen", errCampaignInvalid)
			}
			sheet.Yen -= yen
			return nil
		})
		if err != nil {
			return err
		}
		c.Members[senderIndex] = refreshMemberFromSheet(c.Members[senderIndex], updated)
		c.Deliveries = append(c.Deliveries, CampaignDelivery{
			ID: deliveryID, RecipientCharacterID: recipientID, SenderCharacterID: senderID,
			SenderName: c.Members[senderIndex].Name, Items: validated, Yen: yen, CreatedAt: s.now().UTC(),
		})
		return nil
	})
}

func (s *campaignStore) pendingDeliveries(user, characterID string) []struct {
	CampaignID string `json:"campaign_id"`
	Campaign   string `json:"campaign_name"`
	CampaignDelivery
} {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := []struct {
		CampaignID string `json:"campaign_id"`
		Campaign   string `json:"campaign_name"`
		CampaignDelivery
	}{}
	for _, c := range s.campaigns {
		index := c.memberIndex(characterID)
		if index < 0 || c.Members[index].Owner != user {
			continue
		}
		for _, delivery := range c.Deliveries {
			if delivery.RecipientCharacterID == characterID {
				result = append(result, struct {
					CampaignID string `json:"campaign_id"`
					Campaign   string `json:"campaign_name"`
					CampaignDelivery
				}{c.ID, c.Name, delivery})
			}
		}
	}
	return result
}

type tradeTarget struct {
	CampaignID     string `json:"campaign_id"`
	CampaignName   string `json:"campaign_name"`
	RecipientID    string `json:"recipient_character_id"`
	RecipientAlias string `json:"recipient_alias"`
	RecipientName  string `json:"recipient_name"`
}

func (s *campaignStore) tradeTargets(user, senderID string) []tradeTarget {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]tradeTarget, 0)
	for _, c := range s.campaigns {
		senderIndex := c.memberIndex(senderID)
		if senderIndex < 0 || c.Members[senderIndex].Owner != user {
			continue
		}
		for i, member := range c.Members {
			if i != senderIndex {
				result = append(result, tradeTarget{
					CampaignID: c.ID, CampaignName: c.Name, RecipientID: member.CharacterID,
					RecipientAlias: member.SendAlias, RecipientName: member.Name,
				})
			}
		}
	}
	return result
}

func (s *campaignStore) respondToDelivery(id, actor, characterID, deliveryID string, accept bool) (CampaignDelivery, error) {
	var response CampaignDelivery
	_, err := s.mutate(id, func(c *Campaign) error {
		memberIndex := c.memberIndex(characterID)
		if memberIndex < 0 || c.Members[memberIndex].Owner != actor {
			return errCampaignForbid
		}
		deliveryIndex := -1
		for i, delivery := range c.Deliveries {
			if delivery.ID == deliveryID && delivery.RecipientCharacterID == characterID {
				deliveryIndex = i
				response = delivery
				break
			}
		}
		if deliveryIndex < 0 {
			return errCampaignNotFound
		}
		if accept {
			_, err := savedCharacters.patchChecked(actor, characterID, func(sheet *CharacterSheet) error {
				if addErr := addInventoryItems(sheet, response.Items); addErr != nil {
					return addErr
				}
				if sheet.Yen+response.Yen > maxYen {
					return fmt.Errorf("%w: Yen balance is outside the supported range", errCampaignInvalid)
				}
				sheet.Yen += response.Yen
				return nil
			})
			if err != nil {
				return err
			}
		} else if response.SenderCharacterID == "" {
			if err := addLootItems(&c.DropboxItems, response.Items); err != nil {
				return err
			}
			if c.DropboxYen+response.Yen > maxYen {
				return fmt.Errorf("%w: dropbox Yen is outside the supported range", errCampaignInvalid)
			}
			c.DropboxYen += response.Yen
		} else {
			senderIndex := c.memberIndex(response.SenderCharacterID)
			if senderIndex < 0 {
				if err := addLootItems(&c.DropboxItems, response.Items); err != nil {
					return err
				}
				if c.DropboxYen+response.Yen > maxYen {
					return fmt.Errorf("%w: dropbox Yen is outside the supported range", errCampaignInvalid)
				}
				c.DropboxYen += response.Yen
			} else {
				sender := c.Members[senderIndex]
				returned, err := savedCharacters.patchChecked(sender.Owner, sender.CharacterID, func(sheet *CharacterSheet) error {
					if addErr := addInventoryItems(sheet, response.Items); addErr != nil {
						return addErr
					}
					if sheet.Yen+response.Yen > maxYen {
						return fmt.Errorf("%w: Yen balance is outside the supported range", errCampaignInvalid)
					}
					sheet.Yen += response.Yen
					return nil
				})
				if err != nil {
					return err
				}
				c.Members[senderIndex] = refreshMemberFromSheet(sender, returned)
			}
		}
		c.Deliveries = append(c.Deliveries[:deliveryIndex], c.Deliveries[deliveryIndex+1:]...)
		return nil
	})
	return response, err
}

// dmRollSource is the pseudo character ID the campaign creator uses to roll as the DM.
const dmRollSource = "dm"

// rollInCampaign records a roll made on the campaign screen. The actor rolls as one of their own
// characters in this campaign; the creator may also roll as a non-player entity.
func (s *campaignStore) rollInCampaign(id, actor, sourceID string, roll RollInput) (Campaign, error) {
	if err := roll.validate(); err != nil {
		return Campaign{}, err
	}
	return s.mutate(id, func(c *Campaign) error {
		name := ""
		if sourceID == dmRollSource {
			if actor != c.Owner {
				return errCampaignForbid
			}
			name = "DM"
		} else if index := c.memberIndex(sourceID); index >= 0 {
			if c.Members[index].Owner != actor {
				return errCampaignForbid
			}
			name = c.Members[index].Name
		} else if index := c.npcIndex(sourceID); index >= 0 {
			if actor != c.Owner {
				return errCampaignForbid
			}
			name = c.NPCs[index].Name
		} else {
			return errCampaignNotFound
		}
		c.Rolls = append(c.Rolls, CampaignRoll{ID: c.NextRollID, At: s.now().UTC(), CharacterID: sourceID, Player: actor, Character: name,
			Label: strings.TrimSpace(roll.Label), Detail: roll.Detail, Total: roll.Total, Note: roll.Note})
		c.NextRollID++
		if len(c.Rolls) > maxCampaignRolls {
			c.Rolls = c.Rolls[len(c.Rolls)-maxCampaignRolls:]
		}
		return nil
	})
}

// advanceTurn starts combat or moves to the next combatant that has an initiative score.
// The creator, or the owner of the character whose turn it is, may advance.
func (s *campaignStore) advanceTurn(id, actor string) (Campaign, error) {
	return s.mutate(id, func(c *Campaign) error {
		var order []CampaignMember
		for _, member := range campaignCombatants(*c) {
			if member.Initiative != nil {
				order = append(order, member)
			}
		}
		if len(order) == 0 {
			return errCampaignInvalid
		}
		if !turnActive(*c) {
			if actor != c.Owner {
				return errCampaignForbid
			}
			if c.Round < 1 {
				c.Round = 1
			}
			c.ActiveID = order[0].CharacterID
			return nil
		}
		if actor != c.Owner {
			index := c.memberIndex(c.ActiveID)
			if index < 0 || c.Members[index].Owner != actor {
				return errCampaignForbid
			}
		}
		next := 0
		for i, member := range order {
			if member.CharacterID == c.ActiveID {
				next = i + 1
				break
			}
		}
		if next >= len(order) {
			next = 0
			c.Round++
		}
		c.ActiveID = order[next].CharacterID
		return nil
	})
}

func (s *campaignStore) endCombat(id, actor string) (Campaign, error) {
	return s.mutate(id, func(c *Campaign) error {
		if actor != c.Owner {
			return errCampaignForbid
		}
		c.Round, c.ActiveID = 0, ""
		return nil
	})
}
