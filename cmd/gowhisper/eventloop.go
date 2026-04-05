package main

import (
	"log"
	"time"

	"github.com/erdiegoant/gowhisper/internal/audio"
	"github.com/erdiegoant/gowhisper/internal/chunk"
	"github.com/erdiegoant/gowhisper/internal/clipboard"
	"github.com/erdiegoant/gowhisper/internal/config"
	"github.com/erdiegoant/gowhisper/internal/history"
	ghotkey "github.com/erdiegoant/gowhisper/internal/hotkey"
	"github.com/erdiegoant/gowhisper/internal/llm"
	"github.com/erdiegoant/gowhisper/internal/mode"
	"github.com/erdiegoant/gowhisper/internal/notify"
	"github.com/erdiegoant/gowhisper/internal/sound"
	"github.com/erdiegoant/gowhisper/internal/transcribe"
	"github.com/erdiegoant/gowhisper/internal/ui"
)

// runEventLoop handles hotkey actions and drives the recording state machine.
func runEventLoop(
	capturer *audio.Capturer,
	tr **transcribe.Transcriber,
	hkManager *ghotkey.Manager,
	tray *ui.Tray,
	modeManager *mode.Manager,
	llmClient llm.Processor,
	defaultPrompt string,
	hist *history.Log,
	cfg *config.Manager,
	setModeCh <-chan string,
	cleanupCh <-chan bool,
	cleanupEnabled bool,
	updateModeMenu func(string),
	refreshHistory func(),
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
				handleToggle(capturer, tr, hkManager, tray, modeManager, llmClient, defaultPrompt, hist, cfg, cleanupEnabled, updateModeMenu, refreshHistory)

			case ghotkey.ActionCancel:
				capturer.Cancel()
				hkManager.DisableCancel()
				tray.SetIdle(modeManager.Current().Name)
				log.Println("recording cancelled")
				if cfg.SoundEnabled() {
					sound.Play(sound.Cancel)
				}

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
	tr **transcribe.Transcriber,
	hkManager *ghotkey.Manager,
	tray *ui.Tray,
	modeManager *mode.Manager,
	llmClient llm.Processor,
	defaultPrompt string,
	hist *history.Log,
	cfg *config.Manager,
	cleanupEnabled bool,
	updateModeMenu func(string),
	refreshHistory func(),
) {
	switch capturer.CurrentState() {
	case audio.StateIdle:
		if *tr == nil {
			notify.Show("GoWhisper", "No model installed — open the Models menu to download one")
			return
		}
		// KeepAlive atomically checks if the model is loaded and resets the idle
		// timer, preventing it from firing mid-session. If it returns false the
		// model was already unloaded — reload it now with visual feedback.
		if !(*tr).KeepAlive() {
			tray.SetLoading(modeManager.Current().Name)
			log.Println("model: reloading after idle unload…")
			if err := (*tr).EnsureLoaded(); err != nil {
				log.Printf("model: reload failed: %v", err)
				notify.Show("GoWhisper", "Failed to reload model: "+err.Error())
				tray.SetIdle(modeManager.Current().Name)
				return
			}
		}
		if cfg.SoundEnabled() {
			sound.Play(sound.Start)
		}
		if err := capturer.Start(); err != nil {
			log.Printf("failed to start recording: %v", err)
			return
		}
		hkManager.EnableCancel()
		tray.SetRecording(modeManager.Current().Name)
		log.Println("recording started")

	case audio.StateRecording:
		recordingStart := time.Now()
		samples, err := capturer.Stop()
		if err != nil {
			log.Printf("failed to stop recording: %v", err)
			return
		}
		hkManager.DisableCancel()
		tray.SetProcessing(modeManager.Current().Name)
		if cfg.SoundEnabled() {
			sound.Play(sound.Stop)
		}

		// Enforce hard cap on recording length.
		if maxSecs := cfg.MaxRecordingSeconds(); maxSecs > 0 {
			maxSamples := maxSecs * 16000
			if len(samples) > maxSamples {
				log.Printf("recording capped at %ds (%d samples dropped)", maxSecs, len(samples)-maxSamples)
				samples = samples[:maxSamples]
			}
		}

		var sum float64
		for _, s := range samples {
			sum += float64(s) * float64(s)
		}
		rms := 0.0
		if len(samples) > 0 {
			rms = sum / float64(len(samples))
		}
		log.Printf("captured %d samples (%.1fs) — RMS energy: %.6f — transcribing...",
			len(samples), float64(len(samples))/16000, rms)

		// Snapshot mode at stop time so a mid-flight change doesn't affect this result.
		m := modeManager.Current()

		go func() {
			defer func() {
				capturer.SetIdle()
				tray.SetIdle(modeManager.Current().Name)
				updateModeMenu(modeManager.Current().Name)
			}()

			chunks := chunk.Split(samples, 25, 5)
			if len(chunks) > 1 {
				log.Printf("chunking: %d chunks for %.1fs of audio", len(chunks), float64(len(samples))/16000)
			}

			var transcripts []string
			vocab := m.Vocabulary
			if len(vocab) == 0 {
				vocab = cfg.Vocabulary()
			}
			for i, c := range chunks {
				res, err := (*tr).Transcribe(transcribe.TranscribeRequest{
					Samples:    c,
					Language:   m.Language,
					Translate:  m.Translate,
					Vocabulary: vocab,
				})
				if err != nil {
					log.Printf("transcription chunk %d/%d: %v (skipped)", i+1, len(chunks), err)
					continue
				}
				transcripts = append(transcripts, res)
			}

			if len(transcripts) == 0 {
				log.Println("transcription: no speech detected")
				return
			}

			result := chunk.Stitch(transcripts)
			log.Printf("transcript: %s", result)

			text := result
			if llmClient != nil && cleanupEnabled {
				prompt := m.Prompt
				if prompt == "" {
					prompt = defaultPrompt
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
				return
			}

			if hist != nil {
				prompt := m.Prompt
				if prompt == "" && llmClient != nil && cleanupEnabled {
					prompt = "[default cleanup prompt]"
				}
				go func() {
					if err := hist.Add(history.Entry{
						Timestamp:     time.Now().UTC(),
						ModeName:      m.Name,
						PromptUsed:    prompt,
						RawText:       result,
						ProcessedText: text,
						DurationMs:    time.Since(recordingStart).Milliseconds(),
						Language:      m.Language,
					}); err != nil {
						log.Printf("history: write failed: %v", err)
					}
					refreshHistory()
				}()
			}

			if cfg.NotificationsEnabled() {
				notify.Show("GoWhisper", text)
			}
		}()

	case audio.StateProcessing:
		log.Println("busy — transcription in progress")
	}
}
