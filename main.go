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
	http.Handle("/tiles/", http.StripPrefix("/tiles/", http.FileServer(http.Dir("tiles"))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", router)
	log.Printf("⚔  D&D 5e SRD Interface → http://localhost%s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

// import template for website
var pageTemplates = template.Must(template.New("main.tmpl").Funcs(funcMap).ParseGlob("templates/main.tmpl"))
