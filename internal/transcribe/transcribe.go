package transcribe

import (
	"fmt"
	"strings"
	"sync"

	whisper "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

// TranscribeRequest holds all parameters for a single transcription call.
type TranscribeRequest struct {
	Samples  []float32
	Language string // "auto" | "es" | "en" | etc.
	Translate bool  // true = translate TO English via Whisper native
}

// Transcriber holds a loaded whisper model and serialises transcription calls.
type Transcriber struct {
	mu    sync.Mutex
	model whisper.Model
}

// New loads a whisper model from the given absolute path.
// The model is kept in memory for the lifetime of the Transcriber.
func New(modelPath string) (*Transcriber, error) {
	model, err := whisper.New(modelPath)
	if err != nil {
		return nil, fmt.Errorf("transcribe: load model %q: %w", modelPath, err)
	}
	return &Transcriber{model: model}, nil
}

// Transcribe converts a float32 PCM buffer to text.
// It is safe to call concurrently — calls are serialised via a mutex.
// Returns an error if the result is empty (silence or failed detection).
func (t *Transcriber) Transcribe(req TranscribeRequest) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ctx, err := t.model.NewContext()
	if err != nil {
		return "", fmt.Errorf("transcribe: new context: %w", err)
	}

	// Language — default to "auto" if unset.
	lang := req.Language
	if lang == "" {
		lang = "auto"
	}
	if err := ctx.SetLanguage(lang); err != nil {
		return "", fmt.Errorf("transcribe: set language %q: %w", lang, err)
	}

	if req.Translate {
		ctx.SetTranslate(true)
	}

	// Disable token timestamps and progress callbacks — reduce CGo overhead.
	ctx.SetTokenTimestamps(false)

	if err := ctx.Process(req.Samples, nil, nil, nil); err != nil {
		return "", fmt.Errorf("transcribe: process: %w", err)
	}

	var sb strings.Builder
	for {
		segment, err := ctx.NextSegment()
		if err != nil {
			break
		}
		sb.WriteString(segment.Text)
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "", fmt.Errorf("transcribe: empty result (silence or unrecognised audio)")
	}
	return result, nil
}

// Swap replaces the loaded model with a new one at the given path.
// Used by the config watcher in Phase 5 when the model setting changes.
func (t *Transcriber) Swap(modelPath string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	newModel, err := whisper.New(modelPath)
	if err != nil {
		return fmt.Errorf("transcribe: swap model %q: %w", modelPath, err)
	}
	t.model.Close()
	t.model = newModel
	return nil
}

// Close releases the model from memory.
func (t *Transcriber) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.model.Close()
}
