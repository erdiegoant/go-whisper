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

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	transcribe.SuppressLogs()

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot resolve home dir: %w", err)
	}
	absModel := filepath.Join(home, ".config", "gowhisper", "models", "ggml-small.bin")

	fmt.Println("GoWhisper — Phase 2 test")
	fmt.Printf("Loading model: %s\n", absModel)

	tr, err := transcribe.New(absModel)
	if err != nil {
		return fmt.Errorf("failed to load model: %w", err)
	}
	defer tr.Close()

	fmt.Println("Model loaded. Recording for 5 seconds — speak now...")

	capturer, err := audio.New()
	if err != nil {
		return fmt.Errorf("failed to init audio: %w", err)
	}
	defer capturer.Close()

	if err := capturer.Start(); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-time.After(5 * time.Second):
	case <-sig:
		fmt.Println("\nCancelled.")
		capturer.Cancel()
		return nil
	}

	samples, err := capturer.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop: %w", err)
	}
	fmt.Printf("Captured %d samples (%.1fs). Transcribing...\n", len(samples), float64(len(samples))/16000.0)

	result, err := tr.Transcribe(transcribe.TranscribeRequest{
		Samples:  samples,
		Language: "auto",
	})
	capturer.SetIdle()

	if err != nil {
		fmt.Printf("No speech detected: %v\n", err)
		return nil
	}

	fmt.Printf("\nTranscript: %s\n", result)
	return nil
}
