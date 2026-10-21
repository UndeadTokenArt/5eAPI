package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadGuideFromDirMergesFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spells", "a.json"), `[{"name":"One"}]`)
	writeFile(t, filepath.Join(dir, "spells", "b.json"), `{"name":"Two"}`)
	guide, err := loadGuideFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(guide.Category.Spells); got != 2 {
		t.Fatalf("expected 2 spells, got %d", got)
	}
}

func TestLoadGuideFromDirReportsBadFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spells", "bad.json"), `[{`)
	if _, err := loadGuideFromDir(dir); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestBuiltinGuideCounts(t *testing.T) {
	guide, err := loadGuideFromDir(builtinGuideDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(guide.Category.Spells) < 319 || len(guide.Category.Classes) < 10 {
		t.Fatalf("unexpected counts: spells=%d classes=%d", len(guide.Category.Spells), len(guide.Category.Classes))
	}
}

func TestCurrentGuideReloadsAndKeepsLastGood(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NEOTOKYO_DATA_DIR", dir)
	guideCache.Lock()
	guideCache.loaded = false
	guideCache.Unlock()
	file := filepath.Join(dir, "spells", "a.json")
	writeFile(t, file, `[{"name":"One"}]`)
	guide, _, err := currentGuide()
	if err != nil || len(guide.Category.Spells) != 1 {
		t.Fatalf("initial load: %v", err)
	}
	reset := func() { guideCache.Lock(); guideCache.checkedAt = time.Time{}; guideCache.Unlock() }
	writeFile(t, filepath.Join(dir, "spells", "b.json"), `[{"name":"Two"}]`)
	reset()
	if guide, _, _ = currentGuide(); len(guide.Category.Spells) != 2 {
		t.Fatal("new file not picked up")
	}
	writeFile(t, filepath.Join(dir, "spells", "c.json"), `[{`)
	reset()
	if guide, _, _ = currentGuide(); len(guide.Category.Spells) != 2 {
		t.Fatal("last good data should stay live on a bad edit")
	}
}

func TestSeedGuideDirCopiesOnlyWhenEmpty(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "neotokyo")
	t.Setenv("NEOTOKYO_DATA_DIR", dir)
	seedGuideDir()
	if _, err := os.Stat(filepath.Join(dir, "spells")); err != nil {
		t.Fatalf("not seeded: %v", err)
	}
	marker := filepath.Join(dir, "spells", "spells.json")
	writeFile(t, marker, `[]`)
	seedGuideDir()
	if b, _ := os.ReadFile(marker); string(b) != `[]` {
		t.Fatal("seed overwrote an existing file")
	}
}

func TestSpellLevelAcceptsNumberOrString(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spells", "a.json"), `[{"name":"A","level":2},{"name":"B","level":"3"},{"name":"C"}]`)
	guide, err := loadGuideFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{guide.Category.Spells[0].Level, guide.Category.Spells[1].Level, guide.Category.Spells[2].Level}
	if got[0] != "2" || got[1] != "3" || got[2] != "" {
		t.Fatalf("levels = %q", got)
	}
}
