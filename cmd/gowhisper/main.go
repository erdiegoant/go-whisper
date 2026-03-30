package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/erdiegoant/gowhisper/internal/audio"
	"github.com/erdiegoant/gowhisper/internal/transcribe"
)

const modelPath = "~/.config/gowhisper/models/ggml-small.bin"

func main() {
	// Expand ~ in model path.
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot resolve home dir: %v\n", err)
		os.Exit(1)
	}
	absModel := filepath.Join(home, ".config", "gowhisper", "models", "ggml-small.bin")

	fmt.Println("GoWhisper — Phase 2 test")
	fmt.Printf("Loading model: %s\n", absModel)

	tr, err := transcribe.New(absModel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load model: %v\n", err)
		os.Exit(1)
	}
	defer tr.Close()
	fmt.Println("Model loaded. Recording for 5 seconds — speak now...")

	capturer, err := audio.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init audio: %v\n", err)
		os.Exit(1)
	}
	defer capturer.Close()

	if err := capturer.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start recording: %v\n", err)
		os.Exit(1)
	}

	// Handle Cmd+C for early cancel.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-time.After(5 * time.Second):
		// Normal stop.
	case <-sig:
		fmt.Println("\nCancelled.")
		capturer.Cancel()
		return
	}

	samples, err := capturer.Stop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to stop: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Captured %d samples (%.1fs). Transcribing...\n", len(samples), float64(len(samples))/16000.0)

	result, err := tr.Transcribe(transcribe.TranscribeRequest{
		Samples:  samples,
		Language: "auto",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "transcription failed: %v\n", err)
		capturer.SetIdle()
		os.Exit(1)
	}

	capturer.SetIdle()
	fmt.Printf("\nTranscript: %s\n", result)
}
