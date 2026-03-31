package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/erdiegoant/gowhisper/internal/audio"
	"github.com/erdiegoant/gowhisper/internal/clipboard"
	"github.com/erdiegoant/gowhisper/internal/config"
	ghotkey "github.com/erdiegoant/gowhisper/internal/hotkey"
	"github.com/erdiegoant/gowhisper/internal/llm"
	"github.com/erdiegoant/gowhisper/internal/mode"
	"github.com/erdiegoant/gowhisper/internal/transcribe"
	"github.com/erdiegoant/gowhisper/internal/ui"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to config.yaml (default: ~/.config/gowhisper/config.yaml)")
	flag.Parse()

	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("cannot resolve home dir: %v", err)
		}
		configPath = filepath.Join(home, ".config", "gowhisper", "config.yaml")
	}

	transcribe.SuppressLogs()

	// Accessibility permission check.
	if !ghotkey.CheckAccessibility() {
		fmt.Println("GoWhisper needs Accessibility access.")
		fmt.Println("Please grant it in System Settings → Privacy & Security → Accessibility, then relaunch.")
		os.Exit(0)
	}

	// Load config (creates defaults if missing).
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	defer cfg.Close()

	log.Printf("loading model: %s", cfg.ModelPath())
	tr, err := transcribe.New(cfg.ModelPath())
	if err != nil {
		log.Fatalf("failed to load model: %v", err)
	}
	defer tr.Close()
	log.Println("model loaded")

	// Init Claude cleanup client (optional).
	var llmClient *llm.Client
	if cc := cfg.ClaudeConfig(); cc.APIKey != "" {
		llmClient = llm.New(cc.APIKey, cc.Model, cc.TimeoutSeconds)
		log.Println("llm: Claude cleanup ready")
	} else {
		log.Println("llm: no API key set — transcripts will not be cleaned up")
	}

	// Init audio capturer.
	capturer, err := audio.New()
	if err != nil {
		log.Fatalf("failed to init audio: %v", err)
	}
	defer capturer.Close()

	tray := ui.New()
	tray.Run(func() {
		// Microphone submenu.
		if devices, err := capturer.ListDevices(); err == nil {
			names := make([]string, 0, len(devices)+1)
			names = append(names, "Default")
			for _, d := range devices {
				names = append(names, d.Name())
			}
			tray.AddDeviceMenu(names, func(name string) {
				if name == "Default" {
					capturer.SetDevice(nil)
					log.Println("mic: using system default device")
					return
				}
				for i := range devices {
					if devices[i].Name() == name {
						id := devices[i].ID
						capturer.SetDevice(&id)
						log.Printf("mic: selected %q", name)
						return
					}
				}
			})
		} else {
			log.Printf("mic: could not list devices: %v", err)
		}

		tray.AddOpenConfigItem(configPath)

		go func() {
			combos := cfg.Combos()
			hkManager, err := ghotkey.New(
				ghotkey.Combo(combos.Toggle),
				ghotkey.Combo(combos.Mode),
			)
			if err != nil {
				log.Fatalf("failed to register hotkeys: %v", err)
			}
			defer hkManager.Close()

			// Init mode manager from config (falls back to Standard+Translate if no modes block).
			modeManager := mode.NewManager(cfg.Modes())

			// Restore last-used mode from state.json.
			if state, err := config.LoadState(cfg.Dir()); err == nil && state.LastMode != "" {
				modeManager.SetByName(state.LastMode)
			}

			// Channel for tray mode-picker clicks to reach the event loop.
			setModeCh := make(chan string, 4)

			// Channel for tray cleanup toggle to reach the event loop.
			cleanupCh := make(chan bool, 2)

			// Build initial mode menu.
			updateModeMenu := tray.AddModeMenu(
				modeItems(modeManager.All()),
				func(name string) { setModeCh <- name },
			)
			updateModeMenu(modeManager.Current().Name)

			// Cleanup toggle — default enabled; restore from state.
			cleanupEnabled := true
			if state, err := config.LoadState(cfg.Dir()); err == nil {
				cleanupEnabled = !state.CleanupDisabled
			}
			tray.AddCleanupToggle(cleanupEnabled, func(v bool) { cleanupCh <- v })

			// React to config file changes.
			cfg.OnChange(func(evt config.ChangeEvent) {
				if evt.CombosChanged {
					hkManager.Rebind(
						ghotkey.Combo(evt.Combos.Toggle),
						ghotkey.Combo(evt.Combos.Mode),
						ghotkey.Combo(evt.Combos.Cancel),
					)
					log.Println("config: hotkeys reloaded")
				}
				if evt.ModelChanged {
					go func() {
						log.Printf("config: loading new model: %s", evt.Model)
						if err := tr.Swap(evt.Model); err != nil {
							log.Printf("config: model swap failed: %v", err)
						} else {
							log.Printf("config: model swapped to %s", evt.Model)
						}
					}()
				}
				if evt.ModesChanged {
					// Reload runs in the watcher goroutine; send to event loop via channel.
					go func() { setModeCh <- "" }() // empty string = reload signal
				}
			})

			runEventLoop(capturer, tr, hkManager, tray, modeManager, llmClient, cfg, setModeCh, cleanupCh, cleanupEnabled, updateModeMenu)
		}()
	})
}

// modeItems converts []mode.Mode to []ui.ModeItem, building tooltips.
func modeItems(modes []mode.Mode) []ui.ModeItem {
	items := make([]ui.ModeItem, len(modes))
	for i, m := range modes {
		tooltip := modeTooltip(m)
		items[i] = ui.ModeItem{Name: m.Name, Tooltip: tooltip}
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

// runEventLoop handles hotkey actions and drives the recording state machine.
func runEventLoop(
	capturer *audio.Capturer,
	tr *transcribe.Transcriber,
	hkManager *ghotkey.Manager,
	tray *ui.Tray,
	modeManager *mode.Manager,
	llmClient *llm.Client,
	cfg *config.Manager,
	setModeCh <-chan string,
	cleanupCh <-chan bool,
	cleanupEnabled bool,
	updateModeMenu func(string),
) {
	tray.SetIdle(modeManager.Current().Name)
	log.Println("ready — ⌥Space to record, Esc to cancel, ⌥⇧K to change mode")

	saveCurrentState := func() {
		m := modeManager.Current()
		go func() {
			if err := config.SaveState(cfg.Dir(), config.State{
				LastMode:        m.Name,
				LastLanguage:    m.Language,
				CleanupDisabled: !cleanupEnabled,
			}); err != nil {
				log.Printf("state: save failed: %v", err)
			}
		}()
	}

	activateMode := func(m mode.Mode) {
		tray.SetIdle(m.Name)
		updateModeMenu(m.Name)
		log.Printf("mode: switched to %s", m.Name)
		saveCurrentState()
	}

	for {
		select {
		case action, ok := <-hkManager.C():
			if !ok {
				return
			}
			switch action {
			case ghotkey.ActionToggle:
				handleToggle(capturer, tr, hkManager, tray, modeManager, llmClient, cleanupEnabled, updateModeMenu)

			case ghotkey.ActionCancel:
				capturer.Cancel()
				hkManager.DisableCancel()
				tray.SetIdle(modeManager.Current().Name)
				log.Println("recording cancelled")

			case ghotkey.ActionMode:
				activateMode(modeManager.Next())
			}

		case name := <-setModeCh:
			if name == "" {
				// Modes list changed in config — reload manager and rebuild menu.
				newModes := cfg.Modes()
				modeManager.Reload(newModes)
				log.Printf("config: modes reloaded (%d modes)", len(newModes))
				updateModeMenu(modeManager.Current().Name)
			} else {
				// Tray click — switch to the selected mode.
				if modeManager.SetByName(name) {
					activateMode(modeManager.Current())
				}
			}

		case v := <-cleanupCh:
			cleanupEnabled = v
			log.Printf("cleanup: %s", map[bool]string{true: "enabled", false: "disabled"}[v])
			saveCurrentState()
		}
	}
}

// handleToggle manages the IDLE→RECORDING→PROCESSING→IDLE transition.
func handleToggle(
	capturer *audio.Capturer,
	tr *transcribe.Transcriber,
	hkManager *ghotkey.Manager,
	tray *ui.Tray,
	modeManager *mode.Manager,
	llmClient *llm.Client,
	cleanupEnabled bool,
	updateModeMenu func(string),
) {
	switch capturer.CurrentState() {
	case audio.StateIdle:
		if err := capturer.Start(); err != nil {
			log.Printf("failed to start recording: %v", err)
			return
		}
		hkManager.EnableCancel()
		tray.SetRecording(modeManager.Current().Name)
		log.Println("recording started")

	case audio.StateRecording:
		samples, err := capturer.Stop()
		if err != nil {
			log.Printf("failed to stop recording: %v", err)
			return
		}
		hkManager.DisableCancel()
		tray.SetProcessing(modeManager.Current().Name)

		var sum float64
		for _, s := range samples {
			sum += float64(s) * float64(s)
		}
		rms := 0.0
		if len(samples) > 0 {
			rms = sum / float64(len(samples))
		}
		log.Printf("captured %d samples — RMS energy: %.6f — transcribing...", len(samples), rms)

		// Snapshot mode at stop time so a mid-flight change doesn't affect this result.
		m := modeManager.Current()

		go func() {
			result, err := tr.Transcribe(transcribe.TranscribeRequest{
				Samples:   samples,
				Language:  m.Language,
				Translate: m.Translate,
			})
			capturer.SetIdle()
			tray.SetIdle(modeManager.Current().Name)
			updateModeMenu(modeManager.Current().Name)

			if err != nil {
				log.Printf("transcription: %v", err)
				return
			}

			log.Printf("transcript: %s", result)

			text := result
			if llmClient != nil && cleanupEnabled {
				prompt := m.Prompt
				if prompt == "" {
					prompt = llm.CleanupPrompt
				}
				if cleaned, err := llmClient.Process(prompt, result); err != nil {
					log.Printf("llm: cleanup failed, using raw transcript: %v", err)
				} else {
					log.Printf("llm: cleaned: %s", cleaned)
					text = cleaned
				}
			}

			if err := clipboard.Paste(text); err != nil {
				log.Printf("paste failed: %v", err)
			}
		}()

	case audio.StateProcessing:
		log.Println("busy — transcription in progress")
	}
}

