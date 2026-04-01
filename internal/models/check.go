package models

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ModelStatus describes a single whisper model's local and remote state.
type ModelStatus struct {
	Size      string // "tiny" | "small" | "medium"
	Installed bool   // file exists with non-zero size
	HasUpdate bool   // remote Content-Length differs from local file size
}

var modelSizes = []string{"tiny", "small", "medium"}

// LocalStatuses returns one ModelStatus per supported model based only on disk
// state. Fast — no network calls. HasUpdate is always false.
func LocalStatuses(modelsDir string) []ModelStatus {
	statuses := make([]ModelStatus, len(modelSizes))
	for i, size := range modelSizes {
		statuses[i] = ModelStatus{Size: size}
		path := filepath.Join(modelsDir, "ggml-"+size+".bin")
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			statuses[i].Installed = true
		}
	}
	return statuses
}

// AllStatuses returns one ModelStatus per supported model, including a remote
// update check via HTTP HEAD. Errors from HEAD requests are non-fatal —
// HasUpdate stays false on failure. Suitable for running in a background goroutine.
func AllStatuses(modelsDir string) []ModelStatus {
	statuses := LocalStatuses(modelsDir)
	for i, s := range statuses {
		if !s.Installed {
			continue
		}
		path := filepath.Join(modelsDir, "ggml-"+s.Size+".bin")
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		remote := remoteSize("ggml-" + s.Size + ".bin")
		if remote > 0 && remote != info.Size() {
			statuses[i].HasUpdate = true
		}
	}
	return statuses
}

// remoteSize does an HTTP HEAD request and returns the Content-Length for
// the given model filename. Returns -1 on any error.
func remoteSize(filename string) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, baseURL+"/"+filename, nil)
	if err != nil {
		return -1
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return -1
	}
	return resp.ContentLength
}
