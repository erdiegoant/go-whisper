package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/erdiegoant/gowhisper/internal/audio"
	"github.com/erdiegoant/gowhisper/internal/clipboard"
	"github.com/erdiegoant/gowhisper/internal/config"
	"github.com/erdiegoant/gowhisper/internal/history"
	ghotkey "github.com/erdiegoant/gowhisper/internal/hotkey"
	"github.com/erdiegoant/gowhisper/internal/llm"
	"github.com/erdiegoant/gowhisper/internal/mode"
	"github.com/erdiegoant/gowhisper/internal/models"
	"github.com/erdiegoant/gowhisper/internal/notify"
	"github.com/erdiegoant/gowhisper/internal/transcribe"
	"github.com/erdiegoant/gowhisper/internal/ui"
)

func main() {
	// Subcommand: gowhisper download-model [tiny|small|medium]
	if len(os.Args) >= 2 && os.Args[1] == "download-model" {
		size := "small"
		if len(os.Args) >= 3 {
			size = os.Args[2]
		}
		dir := defaultModelsDir()
		if err := models.Download(size, dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

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

	if !ghotkey.CheckAccessibility() {
		fmt.Println("GoWhisper needs Accessibility access.")
		fmt.Println("Please grant it in System Settings → Privacy & Security → Accessibility, then relaunch.")
		os.Exit(0)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	defer cfg.Close()

	logCloser := setupLogging(cfg.Dir(), cfg.LogLevel())
	defer logCloser.Close()

	var tr *transcribe.Transcriber
	modelPath := cfg.ModelPath()
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		log.Println("model: not found — use the Models menu to download one")
	} else {
		log.Printf("loading model: %s", modelPath)
		timeout := time.Duration(cfg.ModelUnloadTimeoutSeconds()) * time.Second
		loaded, err := transcribe.New(modelPath, timeout)
		if err != nil {
			log.Fatalf("failed to load model: %v", err)
		}
		tr = loaded
		log.Println("model loaded")
	}
	defer func() {
		if tr != nil {
			tr.Close()
		}
	}()

	hist, err := history.Open(cfg.Dir())
	if err != nil {
		log.Printf("history: could not open database: %v — history disabled", err)
	} else {
		defer hist.Close()
	}

	var llmClient llm.Processor
	defaultPrompt := llm.CleanupPrompt
	if oc := cfg.OllamaConfig(); oc.Model != "" {
		llmClient = llm.NewOllama(oc.Model, oc.Host, oc.TimeoutSeconds)
		defaultPrompt = llm.OllamaCleanupPrompt
		log.Printf("llm: Ollama (%s @ %s)", oc.Model, oc.Host)
	} else if cc := cfg.ClaudeConfig(); cc.APIKey != "" {
		llmClient = llm.New(cc.APIKey, cc.Model, cc.TimeoutSeconds)
		log.Printf("llm: Claude (%s)", cc.Model)
	} else {
		log.Println("llm: no backend configured — cleanup disabled")
	}
	if p := cfg.Prompt(); p != "" {
		defaultPrompt = p
	}

	capturer, err := audio.New()
	if err != nil {
		log.Fatalf("failed to init audio: %v", err)
	}
	defer capturer.Close()

	tray := ui.New()
	tray.Run(func() {
		tray.AddOpenConfigItem(configPath)

		cleanupCh := make(chan bool, 2)
		cleanupEnabled := true
		if state, err := config.LoadState(cfg.Dir()); err == nil {
			cleanupEnabled = !state.CleanupDisabled
		}
		tray.AddCleanupToggle(cleanupEnabled, func(v bool) { cleanupCh <- v })

		if devices, err := capturer.ListDevices(); err == nil {
			names := make([]string, 0, len(devices)+1)
			names = append(names, "Default")
			for _, d := range devices {
				names = append(names, d.Name())
			}

			if state, err := config.LoadState(cfg.Dir()); err == nil && state.LastDevice != "" {
				restored := false
				for i := range devices {
					if devices[i].Name() == state.LastDevice {
						id := devices[i].ID
						capturer.SetDevice(&id)
						log.Printf("mic: restored saved device %q", state.LastDevice)
						restored = true
						break
					}
				}
				if !restored {
					log.Printf("mic: saved device %q not available — using default", state.LastDevice)
				}
			}

			tray.AddDeviceMenu(names, func(name string) {
				if name == "Default" {
					capturer.SetDevice(nil)
					log.Println("mic: using system default device")
					go saveDevice(cfg.Dir(), "")
					return
				}
				for i := range devices {
					if devices[i].Name() == name {
						id := devices[i].ID
						capturer.SetDevice(&id)
						log.Printf("mic: selected %q", name)
						go saveDevice(cfg.Dir(), name)
						return
					}
				}
			})
		} else {
			log.Printf("mic: could not list devices: %v", err)
		}

		// Model menu — populate from disk immediately, then check for updates in background.
		// var declared first so the closure below captures it by reference.
		var modelMenu *ui.ModelMenu
		modelMenu = tray.AddModelMenu(
			models.LocalStatuses(cfg.ModelsDir()),
			cfg.ModelSize(),
			func(size string) { go handleModelSelect(size, cfg, modelMenu, &tr) },
		)
		go func() {
			statuses := models.AllStatuses(cfg.ModelsDir())
			modelMenu.Update(statuses, cfg.ModelSize())
			for _, s := range statuses {
				if s.HasUpdate {
					modelMenu.SetHasUpdates(true)
					notify.Show("GoWhisper", "Whisper model updates available — check the Models menu")
					break
				}
			}
		}()

		// History menu — populate immediately from DB, refresh after each transcription.
		historyMenu := tray.AddHistoryMenu(func(text string) {
			clipboard.Write(text)
		})
		refreshHistory := func() {
			if hist == nil {
				return
			}
			entries, err := hist.Recent(ui.HistorySlots)
			if err != nil {
				log.Printf("history: read failed: %v", err)
				return
			}
			items := make([]ui.HistoryEntry, len(entries))
			for i, e := range entries {
				items[i] = ui.HistoryEntry{Text: e.ProcessedText, Mode: e.ModeName, Timestamp: e.Timestamp}
			}
			historyMenu.Update(items)
		}
		refreshHistory()

		if hist != nil {
			historyMenu.AddClearItem(func() {
				if err := hist.Clear(); err != nil {
					log.Printf("history: clear failed: %v", err)
					return
				}
				log.Println("history: cleared")
				refreshHistory()
			})
		}

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

			modeManager := mode.NewManager(cfg.Modes())
			if state, err := config.LoadState(cfg.Dir()); err == nil && state.LastMode != "" {
				modeManager.SetByName(state.LastMode)
			}

			setModeCh := make(chan string, 4)
			updateModeMenu := tray.AddModeMenu(
				ui.ModeItems(modeManager.All()),
				func(name string) { setModeCh <- name },
			)
			updateModeMenu(modeManager.Current().Name)

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
						if tr == nil {
							timeout := time.Duration(cfg.ModelUnloadTimeoutSeconds()) * time.Second
							loaded, err := transcribe.New(evt.Model, timeout)
							if err != nil {
								log.Printf("config: model load failed: %v", err)
								return
							}
							tr = loaded
							log.Printf("config: model loaded: %s", evt.Model)
						} else if err := tr.Swap(evt.Model); err != nil {
							log.Printf("config: model swap failed: %v", err)
						} else {
							log.Printf("config: model swapped to %s", evt.Model)
						}
					}()
				}
				if evt.ModesChanged {
					go func() { setModeCh <- "" }()
				}
				if evt.UnloadTimeoutChanged {
					go func() {
						if tr != nil {
							tr.SetTimeout(time.Duration(evt.UnloadTimeout) * time.Second)
							log.Printf("config: model unload timeout updated to %ds", evt.UnloadTimeout)
						}
					}()
				}
			})

			if tr == nil {
				notify.Show("GoWhisper", "No model installed — open the Models menu to download one")
			}
			runEventLoop(capturer, &tr, hkManager, tray, modeManager, llmClient, defaultPrompt, hist, cfg, setModeCh, cleanupCh, cleanupEnabled, updateModeMenu, refreshHistory)
		}()
	})
}

// setupLogging configures log and slog to write to stderr and a rotating file.
func setupLogging(configDir, logLevel string) io.Closer {
	logPath := filepath.Join(configDir, "gowhisper.log")
	roller := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10,
		MaxBackups: 3,
		Compress:   false,
	}
	log.SetOutput(io.MultiWriter(os.Stderr, roller))
	log.SetFlags(log.LstdFlags)

	level := slog.LevelInfo
	if logLevel == "debug" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(roller, &slog.HandlerOptions{Level: level})))

	fmt.Printf("log file: %s\n", logPath)
	return roller
}
