package main

// ─── Models ────────────────────────────────────────────────────────────────────

type APIList struct {
	Count   int    `json:"count"`
	Results []Stub `json:"results"`
}

type Stub struct {
	Index string `json:"index"`
	Name  string `json:"name"`
	URL   string `json:"url"`
	Level int    `json:"level,omitempty"`
}

type Property struct {
	Label   string
	Values  []string
	IsMulti bool
}
