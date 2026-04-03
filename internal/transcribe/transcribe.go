package transcribe

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	whisper "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

// TranscribeRequest holds all parameters for a single transcription call.
type TranscribeRequest struct {
	Samples   []float32
	Language  string // "auto" | "es" | "en" | etc.
	Translate bool   // true = translate TO English via Whisper native
}

// Transcriber holds a loaded whisper model and serialises transcription calls.
// The model is unloaded automatically after a configurable idle timeout and
// reloaded transparently via EnsureLoaded before the next recording.
type Transcriber struct {
	mu        sync.Mutex
	model     whisper.Model // nil when unloaded
	modelPath string
	timeout   time.Duration // 0 = never unload
	timer     *time.Timer
	seq       int // incremented on every timer reset; guards against stale unload callbacks
}

// New loads a whisper model from the given absolute path.
// timeout is the idle duration after which the model is unloaded from memory;
// pass 0 to keep the model loaded permanently.
func New(modelPath string, timeout time.Duration) (*Transcriber, error) {
	model, err := whisper.New(modelPath)
	if err != nil {
		return nil, fmt.Errorf("transcribe: load model %q: %w", modelPath, err)
	}
	t := &Transcriber{
		model:     model,
		modelPath: modelPath,
		timeout:   timeout,
	}
	t.resetTimer()
	return t, nil
}

// Transcribe converts a float32 PCM buffer to text.
// It is safe to call concurrently — calls are serialised via a mutex.
// Returns an error if the model is not loaded or the result is empty.
func (t *Transcriber) Transcribe(req TranscribeRequest) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.model == nil {
		return "", fmt.Errorf("transcribe: model not loaded")
	}

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
	// Reset the idle timer after each real model invocation.
	t.resetTimer()
	if isSilence(result) {
		return "", fmt.Errorf("transcribe: no speech detected")
	}
	return result, nil
}

// EnsureLoaded loads the model if it has been unloaded by the idle timer.
// Returns immediately if the model is already in memory.
// This may block for several seconds while whisper.cpp loads the model file.
func (t *Transcriber) EnsureLoaded() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.model != nil {
		return nil
	}
	log.Printf("transcribe: reloading model %q after idle unload", t.modelPath)
	m, err := whisper.New(t.modelPath)
	if err != nil {
		return fmt.Errorf("transcribe: reload %q: %w", t.modelPath, err)
	}
	t.model = m
	t.resetTimer()
	return nil
}

// IsLoaded reports whether the model is currently in memory.
func (t *Transcriber) IsLoaded() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.model != nil
}

// KeepAlive resets the idle timer if the model is loaded, preventing an unload
// from occurring during an active session. Returns true if the model is loaded
// (and the timer has been reset), false if it has already been unloaded.
// Use this instead of IsLoaded when about to start recording so the timer
// cannot fire mid-session.
func (t *Transcriber) KeepAlive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.model == nil {
		return false
	}
	t.resetTimer()
	return true
}

// SetTimeout updates the idle unload timeout. Pass 0 to disable auto-unload.
// Safe to call from any goroutine. Takes effect immediately.
func (t *Transcriber) SetTimeout(d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.timeout = d
	if t.model != nil {
		t.resetTimer()
	} else if t.timer != nil {
		t.timer.Stop()
		t.seq++
		t.timer = nil
	}
}

// Swap replaces the loaded model with a new one at the given path.
// Safe to call when the model is currently unloaded.
func (t *Transcriber) Swap(modelPath string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Stop any pending unload timer and invalidate it.
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
	t.seq++

	newModel, err := whisper.New(modelPath)
	if err != nil {
		return fmt.Errorf("transcribe: swap model %q: %w", modelPath, err)
	}
	if t.model != nil {
		t.model.Close()
	}
	t.model = newModel
	t.modelPath = modelPath
	t.resetTimer()
	return nil
}

// Close releases the model from memory and stops the idle timer.
func (t *Transcriber) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.timer != nil {
		t.timer.Stop()
		t.seq++
		t.timer = nil
	}
	if t.model != nil {
		t.model.Close()
		t.model = nil
	}
}

// resetTimer (re)starts the idle unload countdown.
// Must be called with t.mu held.
func (t *Transcriber) resetTimer() {
	if t.timeout == 0 {
		if t.timer != nil {
			t.timer.Stop()
			t.timer = nil
		}
		return
	}
	if t.timer != nil {
		t.timer.Stop()
	}
	t.seq++
	seq := t.seq
	t.timer = time.AfterFunc(t.timeout, func() { t.unloadIfSeq(seq) })
}

// unloadIfSeq closes the model if seq still matches the current sequence number.
// Called from the AfterFunc goroutine — acquires the lock independently.
func (t *Transcriber) unloadIfSeq(seq int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.seq != seq || t.model == nil {
		return // timer was superseded or model already unloaded
	}
	t.model.Close()
	t.model = nil
	t.timer = nil
	log.Printf("transcribe: model unloaded after %s idle", t.timeout)
}

// isSilence reports whether Whisper returned a blank or non-speech marker.
func isSilence(s string) bool {
	switch strings.ToLower(s) {
	case "", "[blank_audio]", "(music)":
		return true
	}
	return false
}
