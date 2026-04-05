package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const helpText = `GoWhisper — local voice transcription for macOS

Usage:
  gowhisper                            launch the menubar app
  gowhisper download-model [size]      download a Whisper model (tiny|small|medium, default: small)
  gowhisper history [n]                print the last n transcriptions (default: 20)
  gowhisper help                       show this help

Options:
  -config <path>   path to config.yaml (default: ~/.config/gowhisper/config.yaml)
`

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "help", "--help", "-h":
			fmt.Print(helpText)
			return

		case "download-model":
			runDownloadModel(os.Args[2:])
			return

		case "history":
			runHistory(os.Args[2:])
			return

		default:
			fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
			fmt.Fprint(os.Stderr, helpText)
			os.Exit(1)
		}
	}

	var configPath string
	flag.StringVar(&configPath, "config", "", "path to config.yaml (default: ~/.config/gowhisper/config.yaml)")
	flag.Parse()

	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "cannot resolve home dir:", err)
			os.Exit(1)
		}
		configPath = filepath.Join(home, ".config", "gowhisper", "config.yaml")
	}

	runApp(configPath)
}
