package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/erdiegoant/gowhisper/internal/config"
	"github.com/erdiegoant/gowhisper/internal/models"
	"github.com/erdiegoant/gowhisper/internal/notify"
	"github.com/erdiegoant/gowhisper/internal/transcribe"
	"github.com/erdiegoant/gowhisper/internal/ui"
)

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
func handleModelSelect(size string, cfg *config.Manager, menu *ui.ModelMenu, tr **transcribe.Transcriber) {
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
	// If tr is still nil the model name didn't change in config, so OnChange
	// won't fire. Load the model directly so recording works immediately.
	if *tr == nil {
		timeout := time.Duration(cfg.ModelUnloadTimeoutSeconds()) * time.Second
		loaded, err := transcribe.New(cfg.ModelPath(), timeout)
		if err != nil {
			log.Printf("models: load after download failed: %v", err)
			notify.Show("GoWhisper", "Download complete but model failed to load: "+err.Error())
			menu.Update(models.LocalStatuses(cfg.ModelsDir()), cfg.ModelSize())
			return
		}
		*tr = loaded
		log.Printf("models: loaded %s after first download", size)
	}
	menu.Update(models.LocalStatuses(cfg.ModelsDir()), size)
	notify.Show("GoWhisper", "Model "+size+" downloaded and ready")
}
