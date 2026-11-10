package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type classProgressionPreview struct {
	Name          string              `json:"name"`
	OriginalClass string              `json:"original_class"`
	HitDie        int                 `json:"hit_die"`
	Levels        []classLevelPreview `json:"levels"`
}

type classLevelPreview struct {
	Level    int               `json:"level"`
	Status   string            `json:"status"`
	Features []awakenedFeature `json:"features"`
	Notes    *string           `json:"notes"`
}

func loadClassProgressionPreview(original string) (classProgressionPreview, error) {
	filename, ok := progressionFiles2024[original]
	if !ok {
		return classProgressionPreview{}, fmt.Errorf("2024 progression for %q is not available", original)
	}
	path := guideDataPath("classesv2", filename)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && filepath.Clean(guideDir()) != filepath.Clean(builtinGuideDir) {
		path = filepath.Join(builtinGuideDir, "classesv2", filename)
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		return classProgressionPreview{}, fmt.Errorf("read %s: %w", path, err)
	}
	var data classProgressionPreview
	if err := json.Unmarshal(raw, &data); err != nil {
		return data, fmt.Errorf("decode %s: %w", path, err)
	}
	if data.OriginalClass != original || data.Name == "" || (data.HitDie != 6 && data.HitDie != 8 && data.HitDie != 10 && data.HitDie != 12) || len(data.Levels) != 20 {
		return data, fmt.Errorf("%s: invalid class identity, hit die, or levels", path)
	}
	for i, level := range data.Levels {
		if level.Level != i+1 || (level.Status != "unfilled" && level.Status != "complete") || level.Features == nil {
			return data, fmt.Errorf("%s: invalid level %d", path, i+1)
		}
		for _, feature := range level.Features {
			if feature.ID == "" || feature.Name == "" || feature.Description == "" {
				return data, fmt.Errorf("%s: incomplete feature at level %d", path, i+1)
			}
		}
	}
	return data, nil
}

func classProgressionPreviewHandler(w http.ResponseWriter, r *http.Request) {
	if user, _ := characterUser(w, r); user == "" {
		writeCharacterError(w, http.StatusUnauthorized, "Sign in through Authelia to manage characters.")
		return
	}
	labels, err := classLabels()
	if err != nil {
		log.Printf("load class labels: %v", err)
		writeCharacterError(w, http.StatusInternalServerError, "Could not load class progression.")
		return
	}
	original := originalClassFor(r.URL.Query().Get("class"), labels)
	if _, ok := progressionFiles2024[original]; !ok {
		writeCharacterError(w, http.StatusBadRequest, "Class progression is not available.")
		return
	}
	data, err := loadClassProgressionPreview(original)
	if err != nil {
		log.Printf("load class progression: %v", err)
		writeCharacterError(w, http.StatusInternalServerError, "Could not load class progression.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeCharacterJSON(w, http.StatusOK, data)
}
