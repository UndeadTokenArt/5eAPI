package main

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

const (
	campaignIdleTTL      = 48 * time.Hour      // shared info is regathered from sheets after this much inactivity
	campaignPurgeAfter   = 30 * 24 * time.Hour // campaigns untouched this long are deleted
	maxCampaignRolls     = 100
	maxCampaignMembers   = 40
	maxCampaignNPCs      = 40
	maxCampaignsPerOwner = 20
	maxConditions        = 12
	maxCyberStrain       = 20
)

// CampaignMember is the shared snapshot of one character. The character sheet
// remains the source of truth; this is refreshed on every sheet save and when
// the campaign's 48 hour window lapses.
type CampaignMember struct {
	CharacterID string   `json:"character_id"`
	SendAlias   string   `json:"send_alias,omitempty"`
	Owner       string   `json:"owner"`
	Name        string   `json:"name"`
	Species     string   `json:"species,omitempty"`
	Class       string   `json:"class,omitempty"`
	CurrentHP   int      `json:"current_hp"`
	MaxHP       int      `json:"max_hp"`
	ArmorClass  int      `json:"armor_class"`
	Initiative  *int     `json:"initiative"`
	Conditions  []string `json:"conditions"`
	CyberStrain int      `json:"cyber_strain"`
	Portrait    string   `json:"portrait,omitempty"`
}

type LootItem struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type CampaignDelivery struct {
	ID                   string     `json:"id"`
	RecipientCharacterID string     `json:"recipient_character_id"`
	SenderCharacterID    string     `json:"sender_character_id,omitempty"`
	SenderName           string     `json:"sender_name"`
	Items                []LootItem `json:"items"`
	Yen                  int        `json:"yen"`
	CreatedAt            time.Time  `json:"created_at"`
}

// CampaignRoll is one entry in the shared dice history.
type CampaignRoll struct {
	ID          int       `json:"id"`
	At          time.Time `json:"at"`
	CharacterID string    `json:"character_id"`
	Player      string    `json:"player"`
	Character   string    `json:"character,omitempty"`
	Label       string    `json:"label"`
	Detail      string    `json:"detail,omitempty"`
	Total       *int      `json:"total,omitempty"`
	Note        string    `json:"note,omitempty"`
}

type Campaign struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Owner            string             `json:"owner"`
	JoinToken        string             `json:"join_token"`
	Members          []CampaignMember   `json:"members"`
	Rolls            []CampaignRoll     `json:"rolls"`
	NPCs             []CampaignNPC      `json:"npcs"`
	DropboxItems     []LootItem         `json:"dropbox_items"`
	DropboxYen       int                `json:"dropbox_yen"`
	Deliveries       []CampaignDelivery `json:"deliveries"`
	NextMemberNumber int                `json:"next_member_number"`
	NextRollID       int                `json:"next_roll_id"`
	Version          int                `json:"version"`
	CreatedAt        time.Time          `json:"created_at"`
	LastActivity     time.Time          `json:"last_activity"`
	LastRegather     time.Time          `json:"last_regather"`
}

const maxYen = 999999999

// CampaignNPC is a non-player entity added by the creator. It lives only in the campaign.
type CampaignNPC struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	CurrentHP  int      `json:"current_hp"`
	MaxHP      int      `json:"max_hp"`
	Initiative *int     `json:"initiative"`
	Conditions []string `json:"conditions"`
}

func validateSharedFields(conditions []string, strain int, portrait string) error {
	if len(conditions) > maxConditions {
		return errors.New("too many conditions")
	}
	for _, condition := range conditions {
		if strings.TrimSpace(condition) == "" || len(condition) > 60 {
			return errors.New("conditions must be 1 to 60 characters")
		}
	}
	if strain < 0 || strain > maxCyberStrain {
		return errors.New("cyber strain is outside the supported range")
	}
	return validatePortrait(portrait)
}

func validatePortrait(portrait string) error {
	if portrait == "" {
		return nil
	}
	if len(portrait) > 500 {
		return errors.New("portrait URL must be at most 500 characters")
	}
	parsed, err := url.Parse(portrait)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("portrait must be an http(s) image URL")
	}
	return nil
}

// memberFromSheet builds a fresh snapshot, keeping an unrolled initiative empty.
func memberFromSheet(owner string, sheet CharacterSheet) CampaignMember {
	member := CampaignMember{
		CharacterID: sheet.ID, Owner: owner, Name: sheet.Name, Species: sheet.Species, Class: sheet.Class,
		CurrentHP: sheet.CurrentHP, MaxHP: sheet.MaxHP, ArmorClass: sheet.ArmorClass,
		Conditions: append([]string{}, sheet.Conditions...), CyberStrain: sheet.CyberStrain, Portrait: sheet.Portrait,
	}
	if sheet.Initiative != 0 {
		value := sheet.Initiative
		member.Initiative = &value
	}
	return member
}

func refreshMemberFromSheet(member CampaignMember, sheet CharacterSheet) CampaignMember {
	updated := memberFromSheet(member.Owner, sheet)
	updated.SendAlias = member.SendAlias
	return updated
}
