package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	hubWriteWait      = 10 * time.Second
	hubPongWait       = 70 * time.Second
	hubPingEvery      = 25 * time.Second
	hubSweepEvery     = 10 * time.Minute
	maxClientsPerRoom = 100
)

// campaignClient is one browser connection. The send buffer is small on purpose:
// a client that cannot keep up is dropped and resyncs from a full snapshot when it reconnects.
type campaignClient struct {
	user string
	conn *websocket.Conn
	send chan []byte
	once sync.Once
}

func (c *campaignClient) close() {
	c.once.Do(func() {
		close(c.send)
		_ = c.conn.Close()
	})
}

type campaignHubT struct {
	mu        sync.Mutex
	rooms     map[string]map[*campaignClient]bool
	startOnce sync.Once
}

var campaignHub = &campaignHubT{rooms: map[string]map[*campaignClient]bool{}}

// Every message carries the complete campaign state, so a missed message can never desync a client.
type campaignMessage struct {
	Type     string        `json:"type"`
	Campaign *campaignView `json:"campaign,omitempty"`
}

type memberView struct {
	CampaignMember
	Player  string      `json:"player"`
	Kind    string      `json:"kind"`
	CanEdit bool        `json:"can_edit"`
	CanRoll bool        `json:"can_roll"`
	Attacks []NPCAttack `json:"attacks,omitempty"`
}

type rollView struct {
	CampaignRoll
	PlayerName string `json:"player_name"`
}

type campaignView struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Creator      string       `json:"creator"`
	IsOwner      bool         `json:"is_owner"`
	JoinPath     string       `json:"join_path,omitempty"`
	Members      []memberView `json:"members"`
	Rolls        []rollView   `json:"rolls"`
	DropboxItems []LootItem   `json:"dropbox_items,omitempty"`
	DropboxYen   int          `json:"dropbox_yen,omitempty"`
	Round        int          `json:"round"`
	ActiveID     string       `json:"active_id,omitempty"`
	Version      int          `json:"version"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

// campaignCombatants lists players and NPCs in initiative order.
func campaignCombatants(c Campaign) []CampaignMember {
	members := append([]CampaignMember{}, c.Members...)
	for _, npc := range c.NPCs {
		members = append(members, CampaignMember{CharacterID: npc.ID, Name: npc.Name, CurrentHP: npc.CurrentHP, MaxHP: npc.MaxHP,
			Initiative: npc.Initiative, Conditions: append([]string{}, npc.Conditions...)})
	}
	sort.SliceStable(members, func(i, j int) bool {
		a, b := members[i].Initiative, members[j].Initiative
		switch {
		case a != nil && b != nil && *a != *b:
			return *a > *b
		case (a == nil) != (b == nil):
			return a != nil
		}
		return strings.ToLower(members[i].Name) < strings.ToLower(members[j].Name)
	})
	return members
}

// turnActive reports whether a turn is in progress with a combatant that still exists.
func turnActive(c Campaign) bool {
	if c.ActiveID == "" || c.Round < 1 {
		return false
	}
	return c.memberIndex(c.ActiveID) >= 0 || c.npcIndex(c.ActiveID) >= 0
}

func displayPlayer(user string) string {
	if strings.HasPrefix(user, "guest:") {
		return "Guest"
	}
	return user
}

// viewFor builds the per-user snapshot: join links only for the creator, and edit rights per member.
func viewFor(c Campaign, user string) campaignView {
	view := campaignView{ID: c.ID, Name: c.Name, Creator: displayPlayer(c.Owner), IsOwner: c.Owner == user, Version: c.Version,
		ExpiresAt: c.LastActivity.Add(campaignIdleTTL), Members: []memberView{}, Rolls: []rollView{}}
	if view.IsOwner {
		view.JoinPath = "/campaigns/join/" + c.JoinToken
		view.DropboxItems = append([]LootItem{}, c.DropboxItems...)
		view.DropboxYen = c.DropboxYen
	}
	members := campaignCombatants(c)
	npcIDs := map[string]bool{}
	npcAttacks := map[string][]NPCAttack{}
	for _, npc := range c.NPCs {
		npcIDs[npc.ID] = true
		if view.IsOwner {
			source := npc.Entity
			if source == "" {
				source = npc.Name
			}
			npcAttacks[npc.ID] = npcAttacksByName(source)
		}
	}
	view.Round = c.Round
	if turnActive(c) {
		view.ActiveID = c.ActiveID
	}
	for _, member := range members {
		if npcIDs[member.CharacterID] {
			view.Members = append(view.Members, memberView{CampaignMember: member, Player: "NPC", Kind: "npc", CanEdit: view.IsOwner, CanRoll: view.IsOwner, Attacks: npcAttacks[member.CharacterID]})
			continue
		}
		own := member.Owner == user
		view.Members = append(view.Members, memberView{CampaignMember: member, Player: displayPlayer(member.Owner), Kind: "pc", CanEdit: view.IsOwner || own, CanRoll: own})
	}
	for i := len(c.Rolls) - 1; i >= 0; i-- {
		name := c.Rolls[i].Character
		if name == "" {
			name = displayPlayer(c.Rolls[i].Player)
		}
		view.Rolls = append(view.Rolls, rollView{CampaignRoll: c.Rolls[i], PlayerName: name})
	}
	for i := range view.Members {
		view.Members[i].Owner = ""
		if !view.IsOwner {
			view.Members[i].SendAlias = ""
		}
	}
	return view
}

func encodeState(c Campaign, user string) []byte {
	view := viewFor(c, user)
	data, _ := json.Marshal(campaignMessage{Type: "state", Campaign: &view})
	return data
}

func (h *campaignHubT) start() {
	h.startOnce.Do(func() {
		go func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					log.Println("campaign janitor recovered:", recovered)
				}
			}()
			ticker := time.NewTicker(hubSweepEvery)
			defer ticker.Stop()
			for range ticker.C {
				changed, purged := campaigns.sweep()
				for _, id := range changed {
					h.broadcast(id)
				}
				for _, id := range purged {
					h.closeRoom(id)
				}
			}
		}()
	})
}

func (h *campaignHubT) add(id string, client *campaignClient) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	room := h.rooms[id]
	if room == nil {
		room = map[*campaignClient]bool{}
		h.rooms[id] = room
	}
	if len(room) >= maxClientsPerRoom {
		return false
	}
	room[client] = true
	return true
}

func (h *campaignHubT) remove(id string, client *campaignClient) {
	h.mu.Lock()
	if room := h.rooms[id]; room != nil {
		delete(room, client)
		if len(room) == 0 {
			delete(h.rooms, id)
		}
	}
	h.mu.Unlock()
	client.close()
}

func (h *campaignHubT) clients(id string) []*campaignClient {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := make([]*campaignClient, 0, len(h.rooms[id]))
	for client := range h.rooms[id] {
		list = append(list, client)
	}
	return list
}

func (h *campaignHubT) closeRoom(id string) {
	for _, client := range h.clients(id) {
		h.remove(id, client)
	}
}

// deliver queues a message without ever blocking the caller; a full buffer drops the client.
func (h *campaignHubT) deliver(id string, client *campaignClient, message []byte) {
	defer func() { _ = recover() }() // the send channel may have been closed concurrently
	select {
	case client.send <- message:
	default:
		h.remove(id, client)
	}
}

func (h *campaignHubT) broadcast(id string) {
	clients := h.clients(id)
	if len(clients) == 0 {
		return
	}
	campaign, ok := campaigns.get(id)
	if !ok {
		h.closeRoom(id)
		return
	}
	for _, client := range clients {
		// Members removed from the campaign stop receiving updates.
		if !campaign.isMember(client.user) {
			h.remove(id, client)
			continue
		}
		h.deliver(id, client, encodeState(campaign, client.user))
	}
}

var campaignUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		parsed, err := url.Parse(r.Header.Get("Origin"))
		return err == nil && parsed.Host != "" && strings.EqualFold(parsed.Host, r.Host)
	},
}

func campaignSocketHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := characterUser(w, r)
	if user == "" {
		http.Error(w, "Sign in to join campaigns.", http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	campaign, ok := campaigns.get(id)
	if !ok || !campaign.isMember(user) {
		http.Error(w, "Campaign not found.", http.StatusNotFound)
		return
	}
	campaignHub.start()
	conn, err := campaignUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &campaignClient{user: user, conn: conn, send: make(chan []byte, 8)}
	if !campaignHub.add(id, client) {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "campaign is full"), time.Now().Add(hubWriteWait))
		_ = conn.Close()
		return
	}
	client.send <- encodeState(campaign, user)
	go campaignHub.writePump(id, client)
	campaignHub.readPump(id, client)
}

func (h *campaignHubT) writePump(id string, client *campaignClient) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Println("campaign writer recovered:", recovered)
		}
		h.remove(id, client)
	}()
	ticker := time.NewTicker(hubPingEvery)
	defer ticker.Stop()
	for {
		select {
		case message, ok := <-client.send:
			_ = client.conn.SetWriteDeadline(time.Now().Add(hubWriteWait))
			if !ok {
				_ = client.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = client.conn.SetWriteDeadline(time.Now().Add(hubWriteWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump only handles keep-alives and explicit "sync" requests; all changes go through the HTTP API.
func (h *campaignHubT) readPump(id string, client *campaignClient) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Println("campaign reader recovered:", recovered)
		}
		h.remove(id, client)
	}()
	client.conn.SetReadLimit(1024)
	_ = client.conn.SetReadDeadline(time.Now().Add(hubPongWait))
	client.conn.SetPongHandler(func(string) error {
		return client.conn.SetReadDeadline(time.Now().Add(hubPongWait))
	})
	for {
		_, data, err := client.conn.ReadMessage()
		if err != nil {
			return
		}
		_ = client.conn.SetReadDeadline(time.Now().Add(hubPongWait))
		var message struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(data, &message) == nil && message.Type == "sync" {
			if campaign, ok := campaigns.get(id); ok && campaign.isMember(client.user) {
				h.deliver(id, client, encodeState(campaign, client.user))
			}
		}
	}
}
