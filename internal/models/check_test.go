package models

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStatuses_emptyDir(t *testing.T) {
	dir := t.TempDir()
	statuses := LocalStatuses(dir)

	if len(statuses) != len(modelSizes) {
		t.Fatalf("want %d statuses, got %d", len(modelSizes), len(statuses))
	}
	for _, s := range statuses {
		if s.Installed {
			t.Errorf("model %q: expected not installed in empty dir", s.Size)
		}
		if s.HasUpdate {
			t.Errorf("model %q: HasUpdate should be false without network check", s.Size)
		}
	}
}

func TestLocalStatuses_installedModel(t *testing.T) {
	dir := t.TempDir()

	// Create a non-empty file for "small"
	path := filepath.Join(dir, "ggml-small.bin")
	if err := os.WriteFile(path, []byte("fake model data"), 0o644); err != nil {
		t.Fatalf("write fake model: %v", err)
	}

	statuses := LocalStatuses(dir)
	for _, s := range statuses {
		if s.Size == "small" {
			if !s.Installed {
				t.Error("small: expected Installed=true")
			}
		} else {
			if s.Installed {
				t.Errorf("%s: expected Installed=false", s.Size)
			}
		}
	}
}

func TestLocalStatuses_zeroBytefile(t *testing.T) {
	dir := t.TempDir()

	// A zero-byte file should not count as installed
	path := filepath.Join(dir, "ggml-tiny.bin")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("write zero-byte file: %v", err)
	}

	statuses := LocalStatuses(dir)
	for _, s := range statuses {
		if s.Size == "tiny" && s.Installed {
			t.Error("tiny: zero-byte file should not be considered installed")
		}
	}
}

func TestLocalStatuses_allSizesPresent(t *testing.T) {
	dir := t.TempDir()
	statuses := LocalStatuses(dir)

	sizes := make(map[string]bool)
	for _, s := range statuses {
		sizes[s.Size] = true
	}
	for _, want := range modelSizes {
		if !sizes[want] {
			t.Errorf("missing status for model size %q", want)
		}
	}
}
