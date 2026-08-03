package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
)

// types for the shop
type shop struct {
	Name             string               `xml:"name"`
	Description      string               `xml:"description"`
	Type             string               `xml:"type"`
	Owner            shopOwner            `xml:"owner"`
	InventoryProfile shopInventoryProfile `xml:"inventory_profile"`
}

type item struct {
	Name        string `xml:"name"`
	Rarity      string `xml:"rarity"`
	Type        string `xml:"type"`
	Description string `xml:"description"`
	Uses        string `xml:"uses"`
	Price       string `xml:"price"`
	Quantity    int    `xml:"quantity"`
}

type itemsData struct {
	Items []item `xml:"item"`
}

type shopOwner struct {
	Name        string `xml:"name"`
	Species     string `xml:"species"`
	Description string `xml:"description"`
}

type shopInventoryProfile struct {
	RarityLevels shopRarityLevels `xml:"rarity_levels"`
}

type shopRarityLevels struct {
	Rarities []string `xml:"rarity"`
}

type shopsData struct {
	Shops []shop `xml:"shop"`
}

func shopHandler(w http.ResponseWriter, r *http.Request) {
	// Get the rarity and quantity from the query parameters
	rarity := r.URL.Query().Get("rarity")
	quantity := 15 // default quantity
	if q := r.URL.Query().Get("quantity"); q != "" {
		fmt.Sscanf(q, "%d", &quantity)
	}
	if quantity < 1 {
		quantity = 5
	}

	// Get the list of items based on rarity and quantity
	items := Shoplist(rarity, quantity)
	shopData, err := getRandomShop(rarity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pageData := map[string]any{
		"shop": map[string]any{
			"name":        shopData.Name,
			"description": shopData.Description,
			"type":        shopData.Type,
			"owner": map[string]any{
				"name":        shopData.Owner.Name,
				"species":     shopData.Owner.Species,
				"description": shopData.Owner.Description,
			},
			"items": toTemplateItems(items),
		},
	}

	// Render the shop template with the items
	var rendered bytes.Buffer
	err = tmpl.ExecuteTemplate(&rendered, "shop.tmpl", pageData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(rendered.Bytes())
}

// gets a random item from the items.xml document and returns a list of items matching the rarity
func Shoplist(rarity string, quantity int) []item {
	matchingItems := getItemsByRarity(rarity)
	if len(matchingItems) == 0 {
		return nil
	}

	rand.Shuffle(len(matchingItems), func(i, j int) {
		matchingItems[i], matchingItems[j] = matchingItems[j], matchingItems[i]
	})
	if quantity > len(matchingItems) {
		quantity = len(matchingItems)
	}
	return matchingItems[:quantity]
}

// gets a list of items matching the rarity from the items.xml document
func getItemsByRarity(rarity string) []item {
	data, err := loadItemsXML("data/items.xml")
	if err != nil {
		log.Fatalf("Error loading items.xml: %v", err)
	}

	if rarity == "" {
		return append([]item(nil), data.Items...)
	}

	var matchingItems []item
	for _, candidate := range data.Items {
		if candidate.Rarity == rarity {
			matchingItems = append(matchingItems, candidate)
		}
	}
	return matchingItems
}

// loads the items.xml document and returns the itemsData struct
func loadItemsXML(filename string) (*itemsData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data itemsData
	decoder := xml.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func loadShopsXML(filename string) (*shopsData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data shopsData
	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}

func getRandomShop(rarity string) (shop, error) {
	data, err := loadShopsXML("data/shop.xml")
	if err != nil {
		return shop{}, err
	}
	if len(data.Shops) == 0 {
		return shop{}, fmt.Errorf("no shops available")
	}

	matchingShops := data.Shops
	if rarity != "" {
		matchingShops = nil
		for _, candidate := range data.Shops {
			if shopSupportsRarity(candidate, rarity) {
				matchingShops = append(matchingShops, candidate)
			}
		}
		if len(matchingShops) == 0 {
			matchingShops = data.Shops
		}
	}

	return matchingShops[rand.Intn(len(matchingShops))], nil
}

func shopSupportsRarity(candidate shop, rarity string) bool {
	for _, level := range candidate.InventoryProfile.RarityLevels.Rarities {
		if level == rarity {
			return true
		}
	}
	return false
}

func toTemplateItems(items []item) []map[string]any {
	templateItems := make([]map[string]any, 0, len(items))
	for _, entry := range items {
		templateItems = append(templateItems, map[string]any{
			"name":        entry.Name,
			"rarity":      entry.Rarity,
			"type":        entry.Type,
			"description": entry.Description,
			"uses":        entry.Uses,
			"price":       entry.Price,
			"quantity":    entry.Quantity,
		})
	}
	return templateItems
}
