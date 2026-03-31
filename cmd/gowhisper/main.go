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

	// Init Claude cleanup client (optional — requires ANTHROPIC_API_KEY or claude.api_key in config).
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

	// Start the systray on the main goroutine — this call blocks until Quit.
	tray := ui.New()
	tray.Run(func() {
		// Build the microphone submenu from available capture devices.
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

			modeManager := &mode.Manager{}

			// Restore last-used mode from state.json.
			if state, err := config.LoadState(cfg.Dir()); err == nil && state.LastMode != "" {
				modeManager.SetByName(state.LastMode)
			}

			// React to config file changes.
			cfg.OnChange(func(newCombos config.Combos, combosChanged bool, newModel string, modelChanged bool) {
				if combosChanged {
					hkManager.Rebind(
						ghotkey.Combo(newCombos.Toggle),
						ghotkey.Combo(newCombos.Mode),
						ghotkey.Combo(newCombos.Cancel),
					)
					log.Println("config: hotkeys reloaded")
				}
				if modelChanged {
					go func() {
						log.Printf("config: loading new model: %s", newModel)
						if err := tr.Swap(newModel); err != nil {
							log.Printf("config: model swap failed: %v", err)
						} else {
							log.Printf("config: model swapped to %s", newModel)
						}
					}()
				}
			})

			runEventLoop(capturer, tr, hkManager, tray, modeManager, llmClient, cfg)
		}()
	})
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
) {
	tray.SetIdle(modeManager.Current().Name)
	log.Println("ready — ⌥Space to record, Esc to cancel, ⌥⇧K to change mode")

	for action := range hkManager.C() {
		switch action {
		case ghotkey.ActionToggle:
			handleToggle(capturer, tr, hkManager, tray, modeManager, llmClient)

		case ghotkey.ActionCancel:
			capturer.Cancel()
			hkManager.DisableCancel()
			tray.SetIdle(modeManager.Current().Name)
			log.Println("recording cancelled")

		case ghotkey.ActionMode:
			m := modeManager.Next()
			tray.SetIdle(m.Name)
			log.Printf("mode: switched to %s", m.Name)
			go persistState(cfg, m)
		}
	}
}

// handleToggle manages the IDLE→RECORDING→PROCESSING→IDLE transition.
func handleToggle(capturer *audio.Capturer, tr *transcribe.Transcriber, hkManager *ghotkey.Manager, tray *ui.Tray, modeManager *mode.Manager, llmClient *llm.Client) {
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

		// Snapshot mode at recording-stop time so a mid-flight mode change doesn't affect this result.
		m := modeManager.Current()

		// Transcribe and paste in a goroutine so the hotkey loop stays responsive.
		go func() {
			result, err := tr.Transcribe(transcribe.TranscribeRequest{
				Samples:   samples,
				Language:  m.Language,
				Translate: m.Translate,
			})
			capturer.SetIdle()
			tray.SetIdle(modeManager.Current().Name)

			if err != nil {
				log.Printf("transcription: %v", err)
				return
			}

			log.Printf("transcript: %s", result)

			text := result
			if llmClient != nil {
				if cleaned, err := llmClient.Process(llm.CleanupPrompt, result); err != nil {
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
		// Ignore — transcription already in flight.
		log.Println("busy — transcription in progress")
	}
}

// persistState saves the current mode to state.json asynchronously.
func persistState(cfg *config.Manager, m mode.Mode) {
	if err := config.SaveState(cfg.Dir(), config.State{
		LastMode:     m.Name,
		LastLanguage: m.Language,
	}); err != nil {
		log.Printf("state: save failed: %v", err)
	}
}
