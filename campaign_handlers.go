package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
)

func campaignError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errCampaignNotFound):
		writeCharacterError(w, http.StatusNotFound, "Campaign not found.")
	case errors.Is(err, errCampaignForbid):
		writeCharacterError(w, http.StatusForbidden, "You are not allowed to do that in this campaign.")
	case errors.Is(err, errCampaignInvalid):
		writeCharacterError(w, http.StatusBadRequest, strings.TrimPrefix(err.Error(), errCampaignInvalid.Error()+": "))
	default:
		log.Println("campaign error:", err)
		writeCharacterError(w, http.StatusInternalServerError, "Could not save the campaign.")
	}
}

func decodeCampaignBody(w http.ResponseWriter, r *http.Request, target any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeCharacterError(w, http.StatusUnsupportedMediaType, "Expected application/json.")
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeCharacterError(w, http.StatusBadRequest, "Invalid request data.")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeCharacterError(w, http.StatusBadRequest, "Request must contain one JSON object.")
		return false
	}
	return true
}

// campaignAuth authenticates, applies the origin check to mutations, and disables caching.
func campaignAuth(w http.ResponseWriter, r *http.Request) (string, bool) {
	user, _ := characterUser(w, r)
	if user == "" {
		writeCharacterError(w, http.StatusUnauthorized, "Sign in to use campaigns.")
		return "", false
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet && !sameOriginMutation(w, r) {
		return "", false
	}
	return user, true
}

// campaignMemberOnly loads the campaign and 404s for non-members so IDs do not leak existence.
func campaignMemberOnly(w http.ResponseWriter, user, id string) (Campaign, bool) {
	campaign, ok := campaigns.get(id)
	if !ok || !campaign.isMember(user) {
		writeCharacterError(w, http.StatusNotFound, "Campaign not found.")
		return Campaign{}, false
	}
	return campaign, true
}

type campaignSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IsOwner bool   `json:"is_owner"`
	Members int    `json:"members"`
}

func campaignPageHandler(w http.ResponseWriter, r *http.Request) {
	user, guest := characterUser(w, r)
	if user == "" {
		http.Error(w, "Authentication required. Sign in through Authelia to use campaigns.", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	displayUser := user
	if guest {
		displayUser = "Temporary guest access"
	}
	data := map[string]any{"User": displayUser, "Guest": guest, "Mode": "list", "CampaignID": "", "JoinToken": ""}
	switch {
	case strings.HasPrefix(r.URL.Path, "/campaigns/join/"):
		data["Mode"] = "join"
		data["JoinToken"] = r.PathValue("token")
	case r.PathValue("id") != "":
		data["Mode"] = "view"
		data["CampaignID"] = r.PathValue("id")
	}
	if err := tmpl.ExecuteTemplate(w, "campaigns", data); err != nil {
		log.Println("campaign page template error:", err)
		http.Error(w, "Could not render campaigns", http.StatusInternalServerError)
	}
}

func campaignsListHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		var body struct {
			Name string `json:"name"`
		}
		if !decodeCampaignBody(w, r, &body) {
			return
		}
		created, err := campaigns.create(user, body.Name)
		if err != nil {
			campaignError(w, err)
			return
		}
		writeCharacterJSON(w, http.StatusCreated, viewFor(created, user))
		return
	}
	summaries := []campaignSummary{}
	for _, campaign := range campaigns.listFor(user) {
		summaries = append(summaries, campaignSummary{ID: campaign.ID, Name: campaign.Name, IsOwner: campaign.Owner == user, Members: len(campaign.Members)})
	}
	writeCharacterJSON(w, http.StatusOK, summaries)
}

func campaignGetHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	campaign, ok := campaignMemberOnly(w, user, r.PathValue("id"))
	if !ok {
		return
	}
	writeCharacterJSON(w, http.StatusOK, viewFor(campaign, user))
}

func campaignDeleteHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, ok := campaignMemberOnly(w, user, id); !ok {
		return
	}
	if err := campaigns.remove(id, user); err != nil {
		campaignError(w, err)
		return
	}
	campaignHub.closeRoom(id)
	w.WriteHeader(http.StatusNoContent)
}

// campaignJoinInfoHandler shows the name behind a join link, so the join page can ask which characters to add.
func campaignJoinInfoHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	campaign, found := campaigns.byToken(r.PathValue("token"))
	if !found {
		writeCharacterError(w, http.StatusNotFound, "This join link is not valid.")
		return
	}
	writeCharacterJSON(w, http.StatusOK, map[string]any{"id": campaign.ID, "name": campaign.Name, "already_member": campaign.isMember(user)})
}

func campaignJoinHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	var body struct {
		CharacterIDs []string `json:"character_ids"`
	}
	if !decodeCampaignBody(w, r, &body) {
		return
	}
	if len(body.CharacterIDs) > maxCampaignMembers {
		writeCharacterError(w, http.StatusBadRequest, "Too many characters.")
		return
	}
	joined, err := campaigns.join(r.PathValue("token"), user, body.CharacterIDs)
	if err != nil {
		campaignError(w, err)
		return
	}
	campaignHub.broadcast(joined.ID)
	writeCharacterJSON(w, http.StatusOK, map[string]string{"id": joined.ID})
}

func campaignMemberUpdateHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, ok := campaignMemberOnly(w, user, id); !ok {
		return
	}
	var patch MemberPatch
	if !decodeCampaignBody(w, r, &patch) {
		return
	}
	updated, err := campaigns.updateMember(id, user, r.PathValue("character"), patch)
	if err != nil {
		campaignError(w, err)
		return
	}
	campaignHub.broadcast(id)
	writeCharacterJSON(w, http.StatusOK, viewFor(updated, user))
}

func campaignMemberDeleteHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, ok := campaignMemberOnly(w, user, id); !ok {
		return
	}
	if _, err := campaigns.removeMember(id, user, r.PathValue("character")); err != nil {
		campaignError(w, err)
		return
	}
	campaignHub.broadcast(id)
	w.WriteHeader(http.StatusNoContent)
}

// characterRollHandler records a roll made on a character sheet in every campaign that character has joined.
func characterRollHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	var roll RollInput
	if !decodeCampaignBody(w, r, &roll) {
		return
	}
	updated, err := campaigns.recordRoll(user, r.PathValue("id"), roll)
	if err != nil {
		campaignError(w, err)
		return
	}
	for _, id := range updated {
		campaignHub.broadcast(id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func campaignNPCCreateHandler(w http.ResponseWriter, r *http.Request) {
	campaignNPCHandler(w, r, func(id, user string, input NPCInput) (Campaign, error) { return campaigns.addNPC(id, user, input) })
}

func campaignNPCUpdateHandler(w http.ResponseWriter, r *http.Request) {
	campaignNPCHandler(w, r, func(id, user string, input NPCInput) (Campaign, error) {
		return campaigns.updateNPC(id, user, r.PathValue("npc"), input)
	})
}

func campaignNPCHandler(w http.ResponseWriter, r *http.Request, apply func(id, user string, input NPCInput) (Campaign, error)) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, ok := campaignMemberOnly(w, user, id); !ok {
		return
	}
	var input NPCInput
	if !decodeCampaignBody(w, r, &input) {
		return
	}
	updated, err := apply(id, user, input)
	if err != nil {
		campaignError(w, err)
		return
	}
	campaignHub.broadcast(id)
	writeCharacterJSON(w, http.StatusOK, viewFor(updated, user))
}

func campaignNPCDeleteHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, ok := campaignMemberOnly(w, user, id); !ok {
		return
	}
	if _, err := campaigns.removeNPC(id, user, r.PathValue("npc")); err != nil {
		campaignError(w, err)
		return
	}
	campaignHub.broadcast(id)
	w.WriteHeader(http.StatusNoContent)
}

// campaignRollHandler records a dice roll made from the campaign screen.
func campaignRollHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, ok := campaignMemberOnly(w, user, id); !ok {
		return
	}
	var body struct {
		RollInput
		CharacterID string `json:"character_id"`
	}
	if !decodeCampaignBody(w, r, &body) {
		return
	}
	if _, err := campaigns.rollInCampaign(id, user, body.CharacterID, body.RollInput); err != nil {
		campaignError(w, err)
		return
	}
	campaignHub.broadcast(id)
	w.WriteHeader(http.StatusNoContent)
}

func campaignDropboxHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	if _, ok := campaignMemberOnly(w, user, r.PathValue("id")); !ok {
		return
	}
	var body struct {
		Items []LootItem `json:"items"`
		Yen   int        `json:"yen"`
	}
	if !decodeCampaignBody(w, r, &body) {
		return
	}
	id := r.PathValue("id")
	updated, err := campaigns.addToDropbox(id, user, body.Items, body.Yen)
	if err != nil {
		campaignError(w, err)
		return
	}
	campaignHub.broadcast(id)
	writeCharacterJSON(w, http.StatusOK, viewFor(updated, user))
}

func campaignDropboxClearHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, ok := campaignMemberOnly(w, user, id); !ok {
		return
	}
	updated, err := campaigns.clearDropbox(id, user)
	if err != nil {
		campaignError(w, err)
		return
	}
	campaignHub.broadcast(id)
	writeCharacterJSON(w, http.StatusOK, viewFor(updated, user))
}

func campaignLootHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	if _, ok := campaignMemberOnly(w, user, r.PathValue("id")); !ok {
		return
	}
	var body struct {
		CR float64 `json:"cr"`
	}
	if !decodeCampaignBody(w, r, &body) {
		return
	}
	guide, _, err := currentGuide()
	if err != nil {
		log.Println("load guide for loot table:", err)
		writeCharacterError(w, http.StatusInternalServerError, "Could not load loot items.")
		return
	}
	items, yen, err := lootForCR(guide, body.CR)
	if err != nil {
		campaignError(w, err)
		return
	}
	id := r.PathValue("id")
	updated, err := campaigns.addToDropbox(id, user, items, yen)
	if err != nil {
		campaignError(w, err)
		return
	}
	campaignHub.broadcast(id)
	writeCharacterJSON(w, http.StatusOK, viewFor(updated, user))
}

func campaignDropboxSendHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if _, ok := campaignMemberOnly(w, user, id); !ok {
		return
	}
	updated, err := campaigns.sendDropbox(id, user, r.PathValue("alias"))
	if err != nil {
		campaignError(w, err)
		return
	}
	campaignHub.broadcast(id)
	writeCharacterJSON(w, http.StatusOK, viewFor(updated, user))
}

func campaignTradeHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	var body struct {
		SenderID string     `json:"sender_character_id"`
		Items    []LootItem `json:"items"`
		Yen      int        `json:"yen"`
	}
	if !decodeCampaignBody(w, r, &body) {
		return
	}
	id := r.PathValue("id")
	if _, ok := campaignMemberOnly(w, user, id); !ok {
		return
	}
	updated, err := campaigns.sendTrade(id, user, body.SenderID, r.PathValue("alias"), body.Items, body.Yen)
	if err != nil {
		campaignError(w, err)
		return
	}
	campaignHub.broadcast(id)
	writeCharacterJSON(w, http.StatusOK, viewFor(updated, user))
}

func characterDeliveryHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	characterID := r.PathValue("id")
	if _, found := savedCharacters.find(user, characterID); !found {
		writeCharacterError(w, http.StatusNotFound, "Character not found.")
		return
	}
	if r.Method == http.MethodGet {
		writeCharacterJSON(w, http.StatusOK, campaigns.pendingDeliveries(user, characterID))
		return
	}
	var body struct {
		Accept bool `json:"accept"`
	}
	if !decodeCampaignBody(w, r, &body) {
		return
	}
	campaignID := r.URL.Query().Get("campaign")
	_, err := campaigns.respondToDelivery(campaignID, user, characterID, r.PathValue("delivery"), body.Accept)
	if err != nil {
		campaignError(w, err)
		return
	}
	campaignHub.broadcast(campaignID)
	writeCharacterJSON(w, http.StatusOK, campaigns.pendingDeliveries(user, characterID))
}

func characterTradeTargetsHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := campaignAuth(w, r)
	if !ok {
		return
	}
	characterID := r.PathValue("id")
	if _, found := savedCharacters.find(user, characterID); !found {
		writeCharacterError(w, http.StatusNotFound, "Character not found.")
		return
	}
	writeCharacterJSON(w, http.StatusOK, campaigns.tradeTargets(user, characterID))
}

func registerCampaignRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /campaigns", campaignPageHandler)
	mux.HandleFunc("GET /campaigns/join/{token}", campaignPageHandler)
	mux.HandleFunc("GET /campaigns/{id}", campaignPageHandler)
	mux.HandleFunc("GET /api/campaigns", campaignsListHandler)
	mux.HandleFunc("POST /api/campaigns", campaignsListHandler)
	mux.HandleFunc("GET /api/campaign-join/{token}", campaignJoinInfoHandler)
	mux.HandleFunc("POST /api/campaign-join/{token}", campaignJoinHandler)
	mux.HandleFunc("GET /api/campaigns/{id}", campaignGetHandler)
	mux.HandleFunc("DELETE /api/campaigns/{id}", campaignDeleteHandler)
	mux.HandleFunc("PATCH /api/campaigns/{id}/members/{character}", campaignMemberUpdateHandler)
	mux.HandleFunc("DELETE /api/campaigns/{id}/members/{character}", campaignMemberDeleteHandler)
	mux.HandleFunc("GET /api/non-player-entities", npcCatalogHandler)
	mux.HandleFunc("POST /api/campaigns/{id}/npcs", campaignNPCCreateHandler)
	mux.HandleFunc("PATCH /api/campaigns/{id}/npcs/{npc}", campaignNPCUpdateHandler)
	mux.HandleFunc("DELETE /api/campaigns/{id}/npcs/{npc}", campaignNPCDeleteHandler)
	mux.HandleFunc("POST /api/campaigns/{id}/rolls", campaignRollHandler)
	mux.HandleFunc("POST /api/campaigns/{id}/dropbox", campaignDropboxHandler)
	mux.HandleFunc("DELETE /api/campaigns/{id}/dropbox", campaignDropboxClearHandler)
	mux.HandleFunc("POST /api/campaigns/{id}/loot", campaignLootHandler)
	mux.HandleFunc("POST /api/campaigns/{id}/senditem/{alias}", campaignDropboxSendHandler)
	mux.HandleFunc("POST /api/campaigns/{id}/trades/{alias}", campaignTradeHandler)
	mux.HandleFunc("GET /api/characters/{id}/deliveries", characterDeliveryHandler)
	mux.HandleFunc("POST /api/characters/{id}/deliveries/{delivery}", characterDeliveryHandler)
	mux.HandleFunc("GET /api/characters/{id}/trade-targets", characterTradeTargetsHandler)
	mux.HandleFunc("GET /ws/campaigns/{id}", campaignSocketHandler)
}
