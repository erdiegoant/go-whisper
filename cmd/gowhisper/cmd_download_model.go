package main

import (
	"fmt"
	"os"

	"github.com/erdiegoant/gowhisper/internal/models"
)

// runDownloadModel handles the `gowhisper download-model [size]` subcommand.
func runDownloadModel(args []string) {
	size := "small"
	if len(args) >= 1 {
		size = args[0]
	}
	dir := defaultModelsDir()
	if err := models.Download(size, dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
