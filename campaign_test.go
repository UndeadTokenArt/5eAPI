package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func useCampaignStores(t *testing.T) *campaignStore {
	t.Helper()
	oldCharacters, oldCampaigns := savedCharacters, campaigns
	dir := t.TempDir()
	savedCharacters = newCharacterStore(filepath.Join(dir, "characters.json"))
	campaigns = newCampaignStore(filepath.Join(dir, "campaigns.json"))
	t.Cleanup(func() { savedCharacters, campaigns = oldCharacters, oldCampaigns })
	return campaigns
}

func campaignRequest(method, target string, body any, user string) *http.Request {
	var reader *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}
	request := httptest.NewRequest(method, target, reader)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://"+request.Host)
	request.Header.Set("Remote-User", user)
	return request
}

func serve(request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	setupRouter().ServeHTTP(recorder, request)
	return recorder
}

func TestCampaignJoinSharesAndSortsByInitiative(t *testing.T) {
	store := useCampaignStores(t)
	sheet := validCharacter("Aster")
	sheet.Species, sheet.Class, sheet.CurrentHP, sheet.MaxHP, sheet.Initiative = "Human", "Fighter", 9, 12, 14
	aster, _ := savedCharacters.create("alice", sheet)
	bobSheet := validCharacter("Bryn")
	bobSheet.Initiative = 18
	bryn, _ := savedCharacters.create("bob", bobSheet)
	idle, _ := savedCharacters.create("bob", validCharacter("Idle"))

	campaign, err := store.create("alice", "Neon Run")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.join(campaign.JoinToken, "alice", []string{aster.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.join(campaign.JoinToken, "bob", []string{bryn.ID, idle.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.join(campaign.JoinToken, "bob", []string{aster.ID}); err == nil {
		t.Fatal("joining with another user's character must fail")
	}
	if _, err := store.join("bogus", "bob", []string{bryn.ID}); err != errCampaignNotFound {
		t.Fatalf("bad token error = %v", err)
	}

	current, _ := store.get(campaign.ID)
	view := viewFor(current, "bob")
	if view.JoinPath != "" || view.IsOwner {
		t.Fatal("non-creator must not see the join link")
	}
	if len(view.Members) != 3 || view.Members[0].Name != "Bryn" || view.Members[1].Name != "Aster" || view.Members[2].Initiative != nil {
		t.Fatalf("members not sorted by initiative: %#v", view.Members)
	}
	if view.Members[1].CanEdit || !view.Members[0].CanEdit {
		t.Fatal("bob may edit only his own characters")
	}
	if !viewFor(current, "alice").Members[0].CanEdit {
		t.Fatal("creator may edit everyone")
	}
}

func TestCampaignEditWritesBackToSheetAndEnforcesPermissions(t *testing.T) {
	store := useCampaignStores(t)
	bryn, _ := savedCharacters.create("bob", validCharacter("Bryn"))
	campaign, _ := store.create("alice", "Run")
	if _, err := store.join(campaign.JoinToken, "bob", []string{bryn.ID}); err != nil {
		t.Fatal(err)
	}
	hp, strain := 7, 3
	conditions := []string{"Optic Glitch"}
	if _, err := store.updateMember(campaign.ID, "carol", bryn.ID, MemberPatch{CurrentHP: &hp}); err != errCampaignForbid {
		t.Fatalf("outsider edit = %v", err)
	}
	if _, err := store.updateMember(campaign.ID, "alice", bryn.ID, MemberPatch{CurrentHP: &hp, CyberStrain: &strain, Conditions: &conditions}); err != nil {
		t.Fatalf("creator edit: %v", err)
	}
	sheet, _ := savedCharacters.find("bob", bryn.ID)
	if sheet.CurrentHP != 7 || sheet.CyberStrain != 3 || len(sheet.Conditions) != 1 {
		t.Fatalf("sheet not updated: %#v", sheet)
	}
	bad := 99
	if _, err := store.updateMember(campaign.ID, "bob", bryn.ID, MemberPatch{CyberStrain: &bad}); err == nil {
		t.Fatal("out-of-range strain accepted")
	}
	// A sheet save flows into the campaign.
	sheet.CurrentHP = 2
	updated, _ := savedCharacters.update("bob", bryn.ID, sheet)
	store.characterChanged("bob", updated)
	current, _ := store.get(campaign.ID)
	if current.Members[0].CurrentHP != 2 {
		t.Fatalf("campaign not refreshed from sheet: %#v", current.Members[0])
	}
}

func TestCampaignRollsAreBoundedAndAttributed(t *testing.T) {
	store := useCampaignStores(t)
	bryn, _ := savedCharacters.create("bob", validCharacter("Bryn"))
	campaign, _ := store.create("alice", "Run")
	_, _ = store.join(campaign.JoinToken, "bob", []string{bryn.ID})
	total := 17
	for i := 0; i < maxCampaignRolls+5; i++ {
		if _, err := store.recordRoll("bob", bryn.ID, RollInput{Label: "Stealth check", Detail: "d20", Total: &total}); err != nil {
			t.Fatal(err)
		}
	}
	ids, _ := store.recordRoll("mallory", bryn.ID, RollInput{Label: "Spoof"})
	if len(ids) != 0 {
		t.Fatal("another user's roll must not be recorded")
	}
	current, _ := store.get(campaign.ID)
	if len(current.Rolls) != maxCampaignRolls || current.Rolls[0].Player != "bob" {
		t.Fatalf("rolls = %d first=%#v", len(current.Rolls), current.Rolls[0])
	}
}

func TestCampaignRegathersAfter48HoursAndPurgesLater(t *testing.T) {
	store := useCampaignStores(t)
	bryn, _ := savedCharacters.create("bob", validCharacter("Bryn"))
	gone, _ := savedCharacters.create("bob", validCharacter("Gone"))
	campaign, _ := store.create("alice", "Run")
	_, _ = store.join(campaign.JoinToken, "bob", []string{bryn.ID, gone.ID})
	_, _ = store.recordRoll("bob", bryn.ID, RollInput{Label: "Roll"})

	// Edit the sheets directly, bypassing the campaign, then let the clock run.
	_, _ = savedCharacters.patch("bob", bryn.ID, func(s *CharacterSheet) { s.CurrentHP = 5; s.Name = "Bryn II" })
	_ = savedCharacters.delete("bob", gone.ID)
	base := store.now()
	store.now = func() time.Time { return base.Add(47 * time.Hour) }
	if current, _ := store.get(campaign.ID); len(current.Rolls) != 1 {
		t.Fatal("campaign regathered too early")
	}
	// Activity at 47h keeps the window open, so test idle from creation time again.
	store.now = func() time.Time { return base.Add(47*time.Hour + campaignIdleTTL) }
	current, ok := store.get(campaign.ID)
	if !ok || len(current.Members) != 1 || current.Members[0].Name != "Bryn II" || current.Members[0].CurrentHP != 5 || len(current.Rolls) != 0 {
		t.Fatalf("regather result: ok=%v %#v", ok, current)
	}

	reloaded := newCampaignStore(store.path)
	if _, ok := reloaded.get(campaign.ID); !ok {
		t.Fatal("campaign not persisted")
	}
	store.now = func() time.Time { return base.Add(47*time.Hour + campaignIdleTTL + campaignPurgeAfter) }
	if _, ok := store.get(campaign.ID); ok {
		t.Fatal("stale campaign was not purged")
	}
}

func TestCampaignHTTPFlowAndWebsocketSync(t *testing.T) {
	useCampaignStores(t)
	bryn, _ := savedCharacters.create("bob", validCharacter("Bryn"))

	created := serve(campaignRequest("POST", "/api/campaigns", map[string]string{"name": "Neon"}, "alice"))
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", created.Code, created.Body)
	}
	var view campaignView
	_ = json.Unmarshal(created.Body.Bytes(), &view)
	token := strings.TrimPrefix(view.JoinPath, "/campaigns/join/")
	if token == "" {
		t.Fatal("creator did not receive a join link")
	}
	if page := serve(campaignRequest("GET", view.JoinPath, nil, "bob")); page.Code != http.StatusOK {
		t.Fatalf("join page = %d", page.Code)
	}
	if noOrigin := httptest.NewRequest("POST", "/api/campaigns", strings.NewReader(`{"name":"x"}`)); serve(noOrigin).Code == http.StatusCreated {
		t.Fatal("unauthenticated or cross-origin create accepted")
	}

	server := httptest.NewServer(setupRouter())
	defer server.Close()
	header := http.Header{"Origin": {server.URL}, "Remote-User": {"alice"}}
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws/campaigns/"+view.ID, header)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	readState := func() campaignMessage {
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		var message campaignMessage
		if err := conn.ReadJSON(&message); err != nil {
			t.Fatalf("read: %v", err)
		}
		return message
	}
	if first := readState(); first.Type != "state" || len(first.Campaign.Members) != 0 {
		t.Fatalf("initial state = %#v", first)
	}

	joined := serve(campaignRequest("POST", "/api/campaign-join/"+token, map[string][]string{"character_ids": {bryn.ID}}, "bob"))
	if joined.Code != http.StatusOK {
		t.Fatalf("join = %d %s", joined.Code, joined.Body)
	}
	if update := readState(); len(update.Campaign.Members) != 1 {
		t.Fatalf("join did not broadcast: %#v", update.Campaign)
	}
	roll := serve(campaignRequest("POST", "/api/characters/"+bryn.ID+"/rolls", RollInput{Label: "Hacking check", Detail: "d20 (14) + 3", Total: intPtr(17)}, "bob"))
	if roll.Code != http.StatusNoContent {
		t.Fatalf("roll = %d %s", roll.Code, roll.Body)
	}
	if update := readState(); len(update.Campaign.Rolls) != 1 || update.Campaign.Rolls[0].PlayerName != "Bryn" {
		t.Fatalf("roll did not broadcast: %#v", update.Campaign.Rolls)
	}

	outsider := http.Header{"Origin": {server.URL}, "Remote-User": {"mallory"}}
	if _, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws/campaigns/"+view.ID, outsider); err == nil || response.StatusCode != http.StatusNotFound {
		t.Fatal("non-member opened the campaign socket")
	}
	foreign := http.Header{"Origin": {"http://evil.example"}, "Remote-User": {"alice"}}
	if _, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws/campaigns/"+view.ID, foreign); err == nil {
		t.Fatal("cross-origin socket accepted")
	}
}

func intPtr(value int) *int { return &value }

func TestCampaignNPCsDiceAndCharacterNames(t *testing.T) {
	store := useCampaignStores(t)
	bryn, _ := savedCharacters.create("bob", validCharacter("Bryn"))
	campaign, _ := store.create("alice", "Run")
	_, _ = store.join(campaign.JoinToken, "bob", []string{bryn.ID})

	name, hp, init := "Guard Drone", 12, 15
	if _, err := store.addNPC(campaign.ID, "bob", NPCInput{Name: &name}); err != errCampaignForbid {
		t.Fatalf("non-creator NPC add = %v", err)
	}

	withNPC, err := store.addNPC(campaign.ID, "alice", NPCInput{Name: &name, MaxHP: &hp, Initiative: &init})
	if err != nil || withNPC.NPCs[0].CurrentHP != 12 {
		t.Fatalf("add NPC: %v %#v", err, withNPC.NPCs)
	}
	npcID := withNPC.NPCs[0].ID
	damaged, cond := 4, []string{"Neural Freeze"}
	if _, err := store.updateNPC(campaign.ID, "alice", npcID, NPCInput{CurrentHP: &damaged, Conditions: &cond}); err != nil {
		t.Fatal(err)
	}
	current, _ := store.get(campaign.ID)
	view := viewFor(current, "alice")
	if len(view.Members) != 2 || view.Members[0].Kind != "npc" || view.Members[0].CurrentHP != 4 || !view.Members[0].CanRoll {
		t.Fatalf("npc view = %#v", view.Members)
	}
	if bobView := viewFor(current, "bob"); bobView.Members[0].CanEdit || bobView.Members[0].CanRoll {
		t.Fatal("players must not edit or roll as NPCs")
	}

	total := 9
	if _, err := store.rollInCampaign(campaign.ID, "alice", bryn.ID, RollInput{Label: "x", Total: &total}); err != errCampaignForbid {
		t.Fatalf("rolling as another player's character = %v", err)
	}
	if _, err := store.rollInCampaign(campaign.ID, "bob", bryn.ID, RollInput{Label: "Rolled 1d20", Total: &total}); err != nil {
		t.Fatal(err)
	}
	_, _ = store.recordRoll("bob", bryn.ID, RollInput{Label: "Sheet roll"})
	if _, err := store.rollInCampaign(campaign.ID, "alice", npcID, RollInput{Label: "Rolled 1d6", Total: &total}); err != nil {
		t.Fatal(err)
	}
	current, _ = store.get(campaign.ID)
	rolls := viewFor(current, "bob").Rolls
	if len(rolls) != 3 || rolls[0].PlayerName != "Guard Drone" || rolls[1].PlayerName != "Bryn" || rolls[2].PlayerName != "Bryn" {
		t.Fatalf("roll names = %#v", rolls)
	}
	if _, err := store.removeNPC(campaign.ID, "alice", npcID); err != nil {
		t.Fatal(err)
	}
}

func TestCharacterHPEndpointUpdatesCampaign(t *testing.T) {
	store := useCampaignStores(t)
	bryn, _ := savedCharacters.create("bob", validCharacter("Bryn"))
	campaign, _ := store.create("alice", "Run")
	_, _ = store.join(campaign.JoinToken, "bob", []string{bryn.ID})

	response := serve(campaignRequest("PATCH", "/api/characters/"+bryn.ID+"/hp", map[string]int{"current_hp": 6, "max_hp": 20}, "bob"))
	if response.Code != http.StatusNoContent {
		t.Fatalf("hp = %d %s", response.Code, response.Body)
	}

	current, _ := store.get(campaign.ID)
	if current.Members[0].CurrentHP != 6 || current.Members[0].MaxHP != 20 {
		t.Fatalf("campaign not updated: %#v", current.Members[0])
	}
	if bad := serve(campaignRequest("PATCH", "/api/characters/"+bryn.ID+"/hp", map[string]int{"current_hp": -1}, "bob")); bad.Code != http.StatusBadRequest {
		t.Fatalf("negative hp = %d", bad.Code)
	}
	if other := serve(campaignRequest("PATCH", "/api/characters/"+bryn.ID+"/hp", map[string]int{"current_hp": 1}, "mallory")); other.Code != http.StatusNotFound {
		t.Fatalf("other user hp = %d", other.Code)
	}
}

func TestCampaignDropboxDeliveriesAcceptAndDeny(t *testing.T) {
	store := useCampaignStores(t)
	player, err := savedCharacters.create("bob", validCharacter("Bryn"))
	if err != nil {
		t.Fatal(err)
	}
	player.Equipment = []CharacterEntry{{Name: "Potion", Quantity: 2}}
	player, err = savedCharacters.update("bob", player.ID, player)
	if err != nil {
		t.Fatal(err)
	}
	campaign, err := store.create("alice", "Loot Run")
	if err != nil {
		t.Fatal(err)
	}
	joined, err := store.join(campaign.JoinToken, "bob", []string{player.ID})
	if err != nil {
		t.Fatal(err)
	}
	if joined.Members[0].SendAlias != "user1" {
		t.Fatalf("first member alias = %q", joined.Members[0].SendAlias)
	}
	if _, err := store.addToDropbox(campaign.ID, "alice", []LootItem{{Name: "Potion", Quantity: 2}, {Name: "Key", Quantity: 1}}, 45); err != nil {
		t.Fatal(err)
	}
	sent, err := store.sendDropbox(campaign.ID, "alice", "user1")
	if err != nil {
		t.Fatal(err)
	}
	if len(sent.DropboxItems) != 0 || sent.DropboxYen != 0 || len(sent.Deliveries) != 1 {
		t.Fatalf("dropbox not emptied into delivery: %#v", sent)
	}
	if _, err := store.sendDropbox(campaign.ID, "alice", "user1"); err == nil {
		t.Fatal("second delivery to a recipient with a pending offer should fail")
	}
	if _, err := store.respondToDelivery(campaign.ID, "bob", player.ID, sent.Deliveries[0].ID, true); err != nil {
		t.Fatal(err)
	}
	updated, ok := savedCharacters.find("bob", player.ID)
	if !ok || updated.Yen != 45 || len(updated.Equipment) != 2 ||
		updated.Equipment[0].Quantity != 4 || updated.Equipment[1].Name != "Key" {
		t.Fatalf("accepted delivery did not update Inventory/Yen: %#v", updated)
	}
	if _, err := store.addToDropbox(campaign.ID, "alice", []LootItem{{Name: "Rope", Quantity: 1}}, 12); err != nil {
		t.Fatal(err)
	}
	sent, err = store.sendDropbox(campaign.ID, "alice", "user1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.respondToDelivery(campaign.ID, "bob", player.ID, sent.Deliveries[0].ID, false); err != nil {
		t.Fatal(err)
	}
	returned, _ := store.get(campaign.ID)
	if len(returned.DropboxItems) != 1 || returned.DropboxItems[0] != (LootItem{Name: "Rope", Quantity: 1}) || returned.DropboxYen != 12 {
		t.Fatalf("denied delivery not returned to dropbox: %#v", returned)
	}
}

func TestCampaignDropboxCanBeClearedByOwnerOnly(t *testing.T) {
	store := useCampaignStores(t)
	player, err := savedCharacters.create("bob", validCharacter("Bryn"))
	if err != nil {
		t.Fatal(err)
	}
	campaign, err := store.create("alice", "Clear Test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.join(campaign.JoinToken, "bob", []string{player.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.addToDropbox(campaign.ID, "alice", []LootItem{{Name: "Rope", Quantity: 2}}, 50); err != nil {
		t.Fatal(err)
	}
	if _, err := store.clearDropbox(campaign.ID, "bob"); err != errCampaignForbid {
		t.Fatalf("non-owner clear = %v", err)
	}
	cleared, err := store.clearDropbox(campaign.ID, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(cleared.DropboxItems) != 0 || cleared.DropboxYen != 0 {
		t.Fatalf("dropbox still has contents after clear: %#v", cleared)
	}
}

func TestPlayerTradeUsesInventoryAndReturnsDeniedOffer(t *testing.T) {
	store := useCampaignStores(t)
	sender := validCharacter("Sage")
	sender.Equipment = []CharacterEntry{{Name: "Medkit", Quantity: 3}}
	sender.Yen = 20
	sender, err := savedCharacters.create("sam", sender)
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := savedCharacters.create("lee", validCharacter("Lee"))
	if err != nil {
		t.Fatal(err)
	}
	campaign, err := store.create("gm", "Trade Test")
	if err != nil {
		t.Fatal(err)
	}
	joined, err := store.join(campaign.JoinToken, "sam", []string{sender.ID})
	if err != nil {
		t.Fatal(err)
	}
	senderAlias := joined.Members[0].SendAlias
	joined, err = store.join(campaign.JoinToken, "lee", []string{recipient.ID})
	if err != nil {
		t.Fatal(err)
	}
	targetAlias := joined.Members[1].SendAlias
	if senderAlias != "user1" || targetAlias != "user2" {
		t.Fatalf("aliases = %q, %q", senderAlias, targetAlias)
	}
	sent, err := store.sendTrade(campaign.ID, "sam", sender.ID, targetAlias, []LootItem{{Name: "Medkit", Quantity: 2}}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if sent.Members[0].SendAlias != "user1" || sent.Members[1].SendAlias != "user2" {
		t.Fatalf("trade removed stable aliases: %#v", sent.Members)
	}
	if _, err := store.sendTrade(campaign.ID, "sam", sender.ID, targetAlias, []LootItem{{Name: "Medkit", Quantity: 1}}, 0); err == nil {
		t.Fatal("second pending offer to the same character should fail")
	}
	updated, _ := savedCharacters.find("sam", sender.ID)
	if updated.Yen != 15 || len(updated.Equipment) != 1 || updated.Equipment[0].Quantity != 1 {
		t.Fatalf("trade did not remove offered assets from sender: %#v", updated)
	}
	delivery := sent.Deliveries[0]
	if _, err := store.respondToDelivery(campaign.ID, "lee", recipient.ID, delivery.ID, false); err != nil {
		t.Fatal(err)
	}
	updated, _ = savedCharacters.find("sam", sender.ID)
	if updated.Yen != 20 || updated.Equipment[0].Quantity != 3 {
		t.Fatalf("declined trade did not return to sender: %#v", updated)
	}
}

func TestLootForCRUsesGuideItemsAndScales(t *testing.T) {
	guide, err := loadGuideFromDir(builtinGuideDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		cr       float64
		minItems int
		maxItems int
		minYen   int
		maxYen   int
	}{{0, 1, 2, 10, 200}, {5, 1, 3, 50, 1000}, {11, 2, 4, 100, 2000}, {17, 3, 5, 250, 5000}} {
		items, yen, err := lootForCR(guide, test.cr)
		if err != nil {
			t.Fatal(err)
		}
		total := 0
		for _, item := range items {
			total += item.Quantity
			if item.Name == "" {
				t.Fatal("generated item has no name")
			}
		}
		if total < test.minItems || total > test.maxItems || yen < test.minYen || yen > test.maxYen {
			t.Fatalf("CR %v yielded %d items and %d Yen", test.cr, total, yen)
		}
	}
	if _, _, err := lootForCR(guide, 31); err == nil {
		t.Fatal("CR above supported range should fail")
	}
}

func TestCampaignSendItemAndCharacterDeliveryRoutes(t *testing.T) {
	store := useCampaignStores(t)
	player, err := savedCharacters.create("bob", validCharacter("Bryn"))
	if err != nil {
		t.Fatal(err)
	}
	campaign, err := store.create("alice", "Route Test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.join(campaign.JoinToken, "bob", []string{player.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.addToDropbox(campaign.ID, "alice", []LootItem{{Name: "Rope", Quantity: 2}}, 25); err != nil {
		t.Fatal(err)
	}
	sent := serve(campaignRequest(http.MethodPost, "/api/campaigns/"+campaign.ID+"/senditem/user1", map[string]any{}, "alice"))
	if sent.Code != http.StatusOK {
		t.Fatalf("senditem route = %d: %s", sent.Code, sent.Body)
	}
	offers := serve(campaignRequest(http.MethodGet, "/api/characters/"+player.ID+"/deliveries", nil, "bob"))
	if offers.Code != http.StatusOK {
		t.Fatalf("delivery list = %d: %s", offers.Code, offers.Body)
	}
	var pending []struct {
		CampaignID string `json:"campaign_id"`
		ID         string `json:"id"`
	}
	if err := json.Unmarshal(offers.Body.Bytes(), &pending); err != nil || len(pending) != 1 {
		t.Fatalf("pending = %#v, error %v", pending, err)
	}
	accepted := serve(campaignRequest(http.MethodPost, "/api/characters/"+player.ID+"/deliveries/"+pending[0].ID+"?campaign="+pending[0].CampaignID, map[string]bool{"accept": true}, "bob"))
	if accepted.Code != http.StatusOK {
		t.Fatalf("accept route = %d: %s", accepted.Code, accepted.Body)
	}
	sheet, _ := savedCharacters.find("bob", player.ID)
	if sheet.Yen != 25 || len(sheet.Equipment) != 1 || sheet.Equipment[0].Quantity != 2 {
		t.Fatalf("delivery was not applied: %#v", sheet)
	}
}
