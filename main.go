package main

import (
	"html/template"
	"io"
	"log"
	"net/http"
)

const (
	dndAPI     = "https://www.dnd5eapi.co/api/2014"
	listenAddr = ":8080"
)

// ─── API Client ────────────────────────────────────────────────────────────────

func fetchAPI(path string) ([]byte, int, error) {
	resp, err := http.Get(dndAPI + path) //nolint:gosec
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

// ─── Template setup ────────────────────────────────────────────────────────────

var funcMap = template.FuncMap{
	"prettyCat": prettyCat,
	"iconFor":   iconFor,
}

var tmpl = pageTemplates

func main() {
	mux := setupRouter()

	log.Printf("D&D 5e SRD Interface → http://localhost%s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, mux))
}

// import template for website
var pageTemplates = template.Must(template.ParseGlob("templates/*.tmpl"))
