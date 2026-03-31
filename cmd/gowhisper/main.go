package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/erdiegoant/gowhisper/internal/audio"
	"github.com/erdiegoant/gowhisper/internal/clipboard"
	ghotkey "github.com/erdiegoant/gowhisper/internal/hotkey"
	"github.com/erdiegoant/gowhisper/internal/mode"
	"github.com/erdiegoant/gowhisper/internal/transcribe"
	"github.com/erdiegoant/gowhisper/internal/ui"
)

func main() {
	transcribe.SuppressLogs()

	// Step 3 — Accessibility permission check.
	if !ghotkey.CheckAccessibility() {
		fmt.Println("GoWhisper needs Accessibility access.")
		fmt.Println("Please grant it in System Settings → Privacy & Security → Accessibility, then relaunch.")
		// AXIsProcessTrustedWithOptions with kAXTrustedCheckOptionPrompt:YES
		// already opened the dialog; we exit cleanly so the user can grant and relaunch.
		os.Exit(0)
	}

	// Load Whisper model.
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("cannot resolve home dir: %v", err)
	}
	modelPath := filepath.Join(home, ".config", "gowhisper", "models", "ggml-small.bin")

	log.Printf("loading model: %s", modelPath)
	tr, err := transcribe.New(modelPath)
	if err != nil {
		log.Fatalf("failed to load model: %v", err)
	}
	defer tr.Close()
	log.Println("model loaded")

	// Init audio capturer.
	capturer, err := audio.New()
	if err != nil {
		log.Fatalf("failed to init audio: %v", err)
	}
	defer capturer.Close()

	// Start the systray on the main goroutine — this call blocks until Quit.
	// Hotkeys are registered inside onReady, from a background goroutine, so
	// that golang.design/x/hotkey's dispatch_sync(main_queue) does not deadlock.
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

		go func() {
			hkManager, err := ghotkey.New()
			if err != nil {
				log.Fatalf("failed to register hotkeys: %v", err)
			}
			defer hkManager.Close()
			modeManager := &mode.Manager{}
			runEventLoop(capturer, tr, hkManager, tray, modeManager)
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
) {
	tray.SetIdle(modeManager.Current().Name)
	log.Println("ready — ⌥Space to record, Esc to cancel, ⌥⇧K to change mode")

	for action := range hkManager.C() {
		switch action {
		case ghotkey.ActionToggle:
			handleToggle(capturer, tr, hkManager, tray, modeManager)

		case ghotkey.ActionCancel:
			capturer.Cancel()
			hkManager.DisableCancel()
			tray.SetIdle(modeManager.Current().Name)
			log.Println("recording cancelled")

		case ghotkey.ActionMode:
			m := modeManager.Next()
			tray.SetIdle(m.Name)
			log.Printf("mode: switched to %s", m.Name)
		}
	}
}

// handleToggle manages the IDLE→RECORDING→PROCESSING→IDLE transition.
func handleToggle(capturer *audio.Capturer, tr *transcribe.Transcriber, hkManager *ghotkey.Manager, tray *ui.Tray, modeManager *mode.Manager) {
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

			if err := clipboard.Paste(result); err != nil {
				log.Printf("paste failed: %v", err)
			}
		}()

	case audio.StateProcessing:
		// Ignore — transcription already in flight.
		log.Println("busy — transcription in progress")
	}
}
