package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	builtinGuideDir  = "static/data/neotokyo"
	referenceDirName = "_reference"
	reloadInterval   = time.Second
)

// guideDir is where the editable guide data lives (a Docker volume in production).
func guideDir() string {
	if dir := strings.TrimSpace(os.Getenv("NEOTOKYO_DATA_DIR")); dir != "" {
		return dir
	}
	return builtinGuideDir
}

func guideDataPath(parts ...string) string {
	return filepath.Join(append([]string{guideDir()}, parts...)...)
}

func referencePath(name string) string { return guideDataPath(referenceDirName, name) }

// seedGuideDir copies the built-in data into an empty configured folder on first start.
func seedGuideDir() {
	dir := guideDir()
	if filepath.Clean(dir) == filepath.Clean(builtinGuideDir) {
		return
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return
	}
	err := filepath.WalkDir(builtinGuideDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, _ := filepath.Rel(builtinGuideDir, path)
		target := filepath.Join(dir, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
	if err != nil {
		log.Println("seed neotokyo data:", err)
		return
	}
	log.Println("seeded NeoTokyo data folder:", dir)
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(to)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// guideFingerprint changes whenever any file under the data folder is added, removed or modified.
func guideFingerprint(dir string) string {
	var sb strings.Builder
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".json") {
			return nil
		}
		if info, err := entry.Info(); err == nil {
			fmt.Fprintf(&sb, "%s|%d|%d;", path, info.Size(), info.ModTime().UnixNano())
		}
		return nil
	})
	return sb.String()
}

// readJSONEntries reads a file holding either an array of entries or one entry.
func readJSONEntries(path string) ([]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") {
		var entries []json.RawMessage
		if err := json.Unmarshal(data, &entries); err != nil {
			return nil, err
		}
		return entries, nil
	}
	var entry json.RawMessage
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return []json.RawMessage{entry}, nil
}

// loadGuideFromDir merges every category folder into one StarterGuide.
func loadGuideFromDir(dir string) (*StarterGuide, error) {
	folders, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read guide data folder: %w", err)
	}
	merged := map[string][]json.RawMessage{}
	for _, folder := range folders {
		if !folder.IsDir() || strings.HasPrefix(folder.Name(), "_") || strings.HasPrefix(folder.Name(), ".") {
			continue
		}
		key := strings.ReplaceAll(strings.ToLower(folder.Name()), "-", " ")
		files, err := os.ReadDir(filepath.Join(dir, folder.Name()))
		if err != nil {
			return nil, err
		}
		sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".json") || strings.HasPrefix(file.Name(), ".") {
				continue
			}
			path := filepath.Join(dir, folder.Name(), file.Name())
			entries, err := readJSONEntries(path)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			merged[key] = append(merged[key], entries...)
		}
	}
	subclassAdditions := merged[subclassesFolder]
	delete(merged, subclassesFolder)
	raw := make(map[string]json.RawMessage, len(merged))
	for key, entries := range merged {
		encoded, err := json.Marshal(entries)
		if err != nil {
			return nil, err
		}
		raw[key] = encoded
	}
	combined, err := json.Marshal(map[string]any{"catagory": raw})
	if err != nil {
		return nil, err
	}
	var guide StarterGuide
	if err := json.Unmarshal(combined, &guide); err != nil {
		return nil, fmt.Errorf("decode guide data: %w", err)
	}
	guide.CustomCategories, err = loadCustomCategoryDefinitions(dir)
	if err != nil {
		return nil, err
	}
	attachSubclassAdditions(&guide, subclassAdditions)
	return &guide, nil
}

// subclassesFolder holds subclass additions; each entry names its parent class
// and is nested into that class when the guide loads.
const subclassesFolder = "subclasses"

type SubclassAddition struct {
	Class string `json:"class"`
	Subclass
}

func findClass(classes []Class, name string) int {
	for i := range classes {
		if strings.EqualFold(strings.TrimSpace(classes[i].Name), strings.TrimSpace(name)) {
			return i
		}
	}
	return -1
}

func attachSubclassAdditions(guide *StarterGuide, entries []json.RawMessage) {
	for _, raw := range entries {
		var addition SubclassAddition
		if err := json.Unmarshal(raw, &addition); err != nil {
			log.Println("skipping subclass addition:", err)
			continue
		}
		i := findClass(guide.Category.Classes, addition.Class)
		if i < 0 {
			log.Printf("skipping subclass %q: parent class %q not found", addition.Name, addition.Class)
			continue
		}
		guide.Category.Classes[i].Subclasses = append(guide.Category.Classes[i].Subclasses, addition.Subclass)
	}
}

type guideSnapshot struct {
	guide   *StarterGuide
	version string
}

var guideCache struct {
	sync.Mutex
	snapshot  guideSnapshot
	checkedAt time.Time
	loaded    bool
	dir       string
}

func invalidateGuideCache() {
	guideCache.Lock()
	guideCache.checkedAt = time.Time{}
	guideCache.Unlock()
}

// currentGuide returns the guide data, reloading it when files in the folder changed.
// A broken edit keeps the last good data live and is logged.
func currentGuide() (*StarterGuide, string, error) {
	guideCache.Lock()
	defer guideCache.Unlock()
	dir := guideDir()
	if guideCache.loaded && guideCache.dir == dir && time.Since(guideCache.checkedAt) < reloadInterval {
		return guideCache.snapshot.guide, guideCache.snapshot.version, nil
	}
	guideCache.checkedAt = time.Now()
	if guideCache.dir != dir {
		guideCache.loaded = false
		guideCache.dir = dir
	}
	version := guideFingerprint(dir)
	if guideCache.loaded && version == guideCache.snapshot.version {
		return guideCache.snapshot.guide, version, nil
	}
	guide, err := loadGuideFromDir(dir)
	if err != nil {
		if guideCache.loaded {
			log.Println("neotokyo data reload failed, keeping previous data:", err)
			return guideCache.snapshot.guide, guideCache.snapshot.version, nil
		}
		return nil, "", err
	}
	guideCache.snapshot = guideSnapshot{guide: guide, version: version}
	guideCache.loaded = true
	return guide, version, nil
}

// versionedCache memoises something derived from the guide until the data changes.
type versionedCache[T any] struct {
	mu      sync.Mutex
	version string
	value   T
	ready   bool
}

func (c *versionedCache[T]) get(build func(*StarterGuide) T) (T, error) {
	guide, version, err := currentGuide()
	if err != nil {
		var zero T
		return zero, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.ready || c.version != version {
		c.value = build(guide)
		c.version = version
		c.ready = true
	}
	return c.value, nil
}
