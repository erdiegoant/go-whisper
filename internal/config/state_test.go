package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadState_missingFile(t *testing.T) {
	dir := t.TempDir()
	s, err := LoadState(dir)
	if err != nil {
		t.Fatalf("expected no error for missing state.json, got: %v", err)
	}
	if s.LastMode != "" || s.LastLanguage != "" {
		t.Errorf("expected zero State, got %+v", s)
	}
}

func TestSaveAndLoadState_roundtrip(t *testing.T) {
	dir := t.TempDir()
	want := State{LastMode: "Formal", LastLanguage: "en", CleanupDisabled: true}

	if err := SaveState(dir, want); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	got, err := LoadState(dir)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if got != want {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

func TestLoadState_cleanupDefaultsEnabled(t *testing.T) {
	// A state.json written before CleanupDisabled existed should default to enabled (false).
	dir := t.TempDir()
	_ = SaveState(dir, State{LastMode: "Standard"})
	got, _ := LoadState(dir)
	if got.CleanupDisabled {
		t.Error("expected CleanupDisabled to default to false (cleanup enabled)")
	}
}

func TestSaveState_atomic(t *testing.T) {
	// After SaveState, no .tmp file should remain.
	dir := t.TempDir()
	if err := SaveState(dir, State{LastMode: "Standard"}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	tmp := filepath.Join(dir, "state.json.tmp")
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error("expected .tmp file to be cleaned up after SaveState")
	}
}

func TestSaveState_overwrite(t *testing.T) {
	dir := t.TempDir()
	_ = SaveState(dir, State{LastMode: "Standard"})
	_ = SaveState(dir, State{LastMode: "Translate", LastLanguage: "es"})

	got, err := LoadState(dir)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if got.LastMode != "Translate" {
		t.Errorf("expected Translate, got %s", got.LastMode)
	}
}

func TestLoadState_malformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadState(dir)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
