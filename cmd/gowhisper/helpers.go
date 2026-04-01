package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/erdiegoant/gowhisper/internal/config"
	"github.com/erdiegoant/gowhisper/internal/mode"
	"github.com/erdiegoant/gowhisper/internal/ui"
)

// modeItems converts []mode.Mode to []ui.ModeItem, building tooltips.
func modeItems(modes []mode.Mode) []ui.ModeItem {
	items := make([]ui.ModeItem, len(modes))
	for i, m := range modes {
		items[i] = ui.ModeItem{Name: m.Name, Tooltip: modeTooltip(m)}
	}
	return items
}

// modeTooltip returns a short description for a mode's tray tooltip.
func modeTooltip(m mode.Mode) string {
	if m.Prompt != "" {
		if len(m.Prompt) > 60 {
			return m.Prompt[:60] + "…"
		}
		return m.Prompt
	}
	if m.Translate {
		return m.Name + " — ES→EN (Whisper native)"
	}
	return m.Name + " — auto transcription"
}

// defaultModelsDir returns ~/.config/gowhisper/models.
func defaultModelsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "gowhisper", "models")
}

// saveDevice persists the selected microphone name to state.json.
// Loads current state first so other fields are not overwritten.
// Pass an empty string to record that the user chose the system default.
func saveDevice(dir, name string) {
	state, _ := config.LoadState(dir)
	state.LastDevice = name
	if err := config.SaveState(dir, state); err != nil {
		log.Printf("state: save device failed: %v", err)
	}
}
