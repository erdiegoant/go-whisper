package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/erdiegoant/gowhisper/internal/history"
)

// runHistory handles the `gowhisper history [n]` subcommand.
func runHistory(args []string) {
	n := 20
	if len(args) >= 1 {
		if _, err := fmt.Sscanf(args[0], "%d", &n); err != nil || n <= 0 {
			fmt.Fprintln(os.Stderr, "usage: gowhisper history [n]")
			os.Exit(1)
		}
	}

	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "gowhisper")
	hist, err := history.Open(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "history: open failed:", err)
		os.Exit(1)
	}
	defer hist.Close()

	entries, err := hist.Recent(n)
	if err != nil {
		fmt.Fprintln(os.Stderr, "history: read failed:", err)
		os.Exit(1)
	}
	if len(entries) == 0 {
		fmt.Println("No history yet.")
		return
	}
	for _, e := range entries {
		fmt.Printf("[%s] (%s) %s\n", e.Timestamp.Local().Format("2006-01-02 15:04"), e.ModeName, e.ProcessedText)
	}
}
