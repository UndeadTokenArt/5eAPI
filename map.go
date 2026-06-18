package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
)

var mapTmpl *template.Template

func init() {
	mapTmpl = template.Must(template.New("").Funcs(funcMap).Parse(mapPageTemplate + mapEditorPageTemplate))
}

func mapHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := mapTmpl.ExecuteTemplate(w, "mappage", nil); err != nil {
		log.Println("map template error:", err)
	}
}

func mapEditorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := mapTmpl.ExecuteTemplate(w, "mapeditor", nil); err != nil {
		log.Println("map editor template error:", err)
	}
}

func mapBiomesHandler(w http.ResponseWriter, r *http.Request) {
	biomes, err := discoverMapBiomes("tiles")
	if err != nil {
		http.Error(w, "Could not discover map biomes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(biomes)
}

func discoverMapBiomes(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	biomes := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		glob := filepath.Join(root, e.Name(), "*.svg")
		matches, gErr := filepath.Glob(glob)
		if gErr != nil {
			continue
		}
		if len(matches) == 0 {
			continue
		}
		biomes = append(biomes, e.Name())
	}

	sort.Strings(biomes)
	return biomes, nil
}

const mapPageTemplate = `
{{define "mappage"}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Map Generator — D&amp;D 5e SRD</title>
  <link rel="stylesheet" href="/static/css/map.css">
</head>
<body>
<nav>
  <a href="/" class="brand">⚔ D&amp;D 5e SRD</a>
  <span class="nav-sub">System Reference Document Explorer</span>
  <a href="/encounter" class="nav-enc">🎲 Encounter Generator</a>
  <a href="/map" class="nav-enc">🗺 Map Generator</a>
  <a href="/map/editor" class="nav-enc">🛠 Tile Editor</a>
</nav>

<div class="map-page">
  <h1>🗺 Random Map Generator</h1>

  <!-- Toolbar -->
  <div class="map-toolbar">
    <!-- Map type -->
    <div class="toolbar-group">
      <span class="toolbar-label">Type</span>
      <div id="map-type-buttons" style="display:flex;gap:.5rem;"></div>
    </div>

    <div class="toolbar-sep"></div>

    <!-- Generate -->
    <div class="toolbar-group">
      <button class="gen-btn" onclick="generate()">🎲 Generate</button>
    </div>

    <div class="toolbar-sep"></div>

    <!-- Themes -->
    <div class="toolbar-group">
      <span class="toolbar-label">Theme</span>
      <div id="theme-swatches" style="display:flex;gap:.35rem;"></div>
    </div>

    <div class="toolbar-sep"></div>

    <!-- Props -->
    <div class="toolbar-group">
      <span class="toolbar-label">Props</span>
      <select id="prop-type-select" class="map-select" onchange="setPropType(this.value)"></select>
    </div>

    <!-- Zoom -->
    <div class="zoom-group">
      <button class="zoom-btn" onclick="zoomOut()" title="Zoom out">−</button>
      <span class="zoom-val" id="zoom-label">100%</span>
      <button class="zoom-btn" onclick="zoomIn()" title="Zoom in">+</button>
      <button class="zoom-btn" onclick="zoomReset()" title="Reset zoom" style="width:auto;padding:0 .4rem;font-size:.7rem;">⌂</button>
    </div>
  </div>

  <!-- Map viewport -->
  <div class="map-viewport" id="map-viewport">
    <div class="map-zoom-wrap" id="zoom-wrap">
      <svg id="map-svg" width="800" height="800" xmlns="http://www.w3.org/2000/svg">
        <defs id="tile-defs"></defs>
        <defs id="prop-defs"></defs>
        <g id="map-layer"></g>
        <g id="prop-layer"></g>
        <!-- placeholder shown before first generate -->
        <foreignObject width="800" height="800" id="map-placeholder-fo">
          <body xmlns="http://www.w3.org/1999/xhtml">
            <div class="map-placeholder">
              <span class="big-icon">🗺</span>
              <span>Click <strong>🎲 Generate</strong> to create a map</span>
            </div>
          </body>
        </foreignObject>
      </svg>
    </div>
  </div>
  <div class="map-status" id="map-status"></div>
</div>

<footer>
  Data provided by <a href="https://www.dnd5eapi.co/" target="_blank" rel="noopener">dnd5eapi.co</a>
  &middot; <a href="https://github.com/5e-bits/5e-srd-api" target="_blank" rel="noopener">GitHub</a>
  &middot; 5e SRD content &copy; Wizards of the Coast
</footer>

<script src="/static/js/map-tiles.js"></script>
<script src="/static/js/map-engine.js"></script>
<script src="/static/js/map-props.js"></script>
<script src="/static/js/map.js"></script>
</body>
</html>
{{end}}
`

const mapEditorPageTemplate = `
{{define "mapeditor"}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Tile Editor — D&amp;D 5e SRD</title>
  <link rel="stylesheet" href="/static/css/map-editor.css">
</head>
<body>
<nav>
  <a href="/" class="brand">⚔ D&amp;D 5e SRD</a>
  <span class="nav-sub">Tile authoring workspace</span>
  <a href="/encounter" class="nav-enc">🎲 Encounter Generator</a>
  <a href="/map" class="nav-enc">🗺 Map Generator</a>
</nav>

<div class="editor-page">
  <div class="editor-hero">
    <div>
      <h1>🛠 Tile Editor</h1>
      <div class="hero-copy">Edit existing SVG tiles, change their properties, create new tile drafts, and export the result as SVG plus metadata JSON for manual save.</div>
    </div>
    <div class="status-pill" id="editor-status">Pick a tile to begin</div>
  </div>

  <div class="editor-grid">
    <aside class="panel">
      <div class="panel-head">
        <div class="panel-title">Tile Library</div>
        <div class="mini-note" id="tile-count"></div>
      </div>
      <div class="panel-body">
        <div class="tile-search">
          <input id="tile-filter" type="search" placeholder="Search tiles...">
          <select id="type-filter"></select>
        </div>
        <div class="tile-list" id="tile-list"></div>
      </div>
    </aside>

    <section class="panel">
      <div class="panel-head">
        <div class="panel-title">SVG Preview + Source</div>
        <div class="tool-row" style="margin:0;">
          <button class="btn" id="reload-btn" type="button">Reload source</button>
          <button class="btn" id="new-btn" type="button">New tile</button>
          <button class="btn" id="import-svg-btn" type="button">Import SVG</button>
          <button class="btn" id="load-json-btn" type="button">Load JSON</button>
          <button class="btn primary" id="save-all-btn" type="button">Save All JSON</button>
          <button class="btn primary" id="svg-export-btn" type="button">Export SVG</button>
          <button class="btn primary" id="json-export-btn" type="button">Export JSON</button>
        </div>
      </div>
      <div class="panel-body preview-stage">
        <input id="import-svg-input" type="file" accept=".svg,image/svg+xml" multiple style="display:none;" />
        <input id="load-json-input" type="file" accept=".json,application/json" style="display:none;" />
        <div class="preview-themebar">
          <span class="toolbar-label">Theme</span>
          <div id="editor-theme-swatches" style="display:flex;gap:.35rem;flex-wrap:wrap;"></div>
        </div>
        <div class="preview-frame"><iframe id="svg-preview" title="Tile preview"></iframe></div>
        <div class="source-box field">
          <label>Insert Shape Template</label>
          <div class="shape-template-row">
            <select id="shape-template-type"></select>
            <button class="btn" id="insert-shape-btn" type="button">Insert Template</button>
          </div>
          <div id="shape-template-fields" class="shape-template-fields"></div>
          <label for="svg-source">SVG source</label>
          <textarea id="svg-source" spellcheck="false"></textarea>
          <div class="hint">Edit the SVG markup directly for now. The preview updates as you type, and export downloads the current source unchanged.</div>
        </div>
      </div>
    </section>

    <aside class="panel">
      <div class="panel-head">
        <div class="panel-title">Tile Properties</div>
        <button class="btn danger" id="reset-btn" type="button">Reset</button>
      </div>
      <div class="panel-body">
        <div class="field-grid">
          <div class="field"><label for="tile-id">Id</label><input id="tile-id" type="text"></div>
          <div class="field"><label for="tile-type">Type</label><select id="tile-type"></select></div>
        </div>
        <div class="field-grid">
          <div class="field"><label for="tile-asset">Asset</label><input id="tile-asset" type="text"></div>
          <div class="field"><label for="tile-weight">Weight</label><input id="tile-weight" type="number" min="0" step="1"></div>
        </div>

        <div class="field">
          <label>Edges</label>
          <div class="field-grid-4">
            <select id="edge-n"></select>
            <select id="edge-e"></select>
            <select id="edge-s"></select>
            <select id="edge-w"></select>
          </div>
          <div class="mini-note">N, E, S, W edge types used by the generator.</div>
        </div>

        <div class="field">
          <label for="tile-groups">Groups</label>
          <input id="tile-groups" type="text" placeholder="corridor, corridor-vertical">
        </div>

        <div class="field">
          <label for="tile-rules">Neighbor weight rules JSON</label>
          <textarea id="tile-rules" spellcheck="false" placeholder='[{"tileIds":["d-corr-v"],"weightOverride":0}]'></textarea>
          <div class="mini-note">Use exact tile ids and/or groups. The current editor stores the JSON directly for export.</div>
        </div>

        <div class="field-grid">
          <button class="btn" id="apply-btn" type="button">Apply changes</button>
          <button class="btn" id="download-manifest-btn" type="button">Download bundle</button>
        </div>
        <div class="footer-status" id="editor-help"></div>
      </div>
    </aside>
  </div>
</div>

<script src="/static/js/map-tiles.js"></script>
<script src="/static/js/map-editor.js"></script>
</body>
</html>
{{end}}
`
