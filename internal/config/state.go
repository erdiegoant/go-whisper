package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// State holds runtime state persisted between launches.
// It lives in state.json alongside config.yaml and is never written back to config.yaml.
type State struct {
	LastMode     string `json:"last_mode"`
	LastLanguage string `json:"last_language"`
}

// LoadState reads state.json from dir. Returns a zero State (not an error) if the file
// does not exist yet.
func LoadState(dir string) (State, error) {
	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if errors.Is(err, os.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, err
	}
	return s, nil
}

// SaveState atomically writes s to state.json in dir.
func SaveState(dir string, s State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "state.json.tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "state.json"))
}
