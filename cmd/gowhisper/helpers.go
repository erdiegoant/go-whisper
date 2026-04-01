package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/erdiegoant/gowhisper/internal/config"
	"github.com/erdiegoant/gowhisper/internal/mode"
	"github.com/erdiegoant/gowhisper/internal/models"
	"github.com/erdiegoant/gowhisper/internal/notify"
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

// handleModelSelect is called when the user clicks a model in the Models menu.
// If the model is installed it switches to it; otherwise it downloads it first.
// Runs in its own goroutine.
func handleModelSelect(size string, cfg *config.Manager, menu *ui.ModelMenu) {
	statuses := models.LocalStatuses(cfg.ModelsDir())

	for _, s := range statuses {
		if s.Size == size && s.Installed {
			if err := cfg.SetModel(size); err != nil {
				log.Printf("models: SetModel %s: %v", size, err)
				return
			}
			// Refresh menu to move checkmark; config watcher triggers tr.Swap automatically.
			menu.Update(models.LocalStatuses(cfg.ModelsDir()), size)
			return
		}
	}

	// Not installed — download with progress reported in the menu item.
	log.Printf("models: downloading %s", size)
	err := models.DownloadWithProgress(size, cfg.ModelsDir(), func(pct float64) {
		menu.SetDownloadProgress(size, pct)
	})
	if err != nil {
		log.Printf("models: download %s failed: %v", size, err)
		notify.Show("GoWhisper", "Download failed: "+err.Error())
		menu.Update(models.LocalStatuses(cfg.ModelsDir()), cfg.ModelSize())
		return
	}
	if err := cfg.SetModel(size); err != nil {
		log.Printf("models: SetModel %s after download: %v", size, err)
	}
	menu.Update(models.LocalStatuses(cfg.ModelsDir()), size)
	notify.Show("GoWhisper", "Model "+size+" downloaded and ready")
}
