package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ClassResource is a limited-use class feature (such as Channel Divinity) tracked on the sheet.
type ClassResource struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Recharge    string `json:"recharge"` // "short" or "long"
	MinLevel    int    `json:"min_level,omitempty"`
	Uses        []int  `json:"uses"` // maximum uses at character level 1..20
	Description string `json:"description,omitempty"`
}

type classResourceData struct {
	Resources map[string][]ClassResource `json:"class_resources"`
}

var classResourceCache struct {
	sync.Mutex
	path    string
	version string
	data    classResourceData
	ok      bool
}

// loadClassResources reads class-resources.json from the data folder, falling back to the
// built-in copy so volumes seeded before this file existed keep working.
func loadClassResources() (classResourceData, error) {
	path := referencePath("class-resources.json")
	info, err := os.Stat(path)
	if err != nil {
		path = filepath.Join(builtinGuideDir, referenceDirName, "class-resources.json")
		if info, err = os.Stat(path); err != nil {
			return classResourceData{}, err
		}
	}
	version := fmt.Sprintf("%d|%d", info.Size(), info.ModTime().UnixNano())
	classResourceCache.Lock()
	defer classResourceCache.Unlock()
	if classResourceCache.ok && classResourceCache.path == path && classResourceCache.version == version {
		return classResourceCache.data, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return classResourceData{}, err
	}
	var data classResourceData
	if err := json.Unmarshal(raw, &data); err != nil {
		return classResourceData{}, fmt.Errorf("%s: %w", path, err)
	}
	classResourceCache.path, classResourceCache.version, classResourceCache.data, classResourceCache.ok = path, version, data, true
	return data, nil
}

// resourcesForClass returns the tracked resources for a Neo Tokyo class label.
func resourcesForClass(label string, labels map[string]string, data classResourceData) []ClassResource {
	original := originalClassFor(label, labels)
	if original == "" {
		return nil
	}
	return data.Resources[original]
}
