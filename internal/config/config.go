package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"golang.design/x/hotkey"
	"gopkg.in/yaml.v3"
)

// Combo is the parsed representation of a hotkey string (e.g. "option+shift+k").
// It mirrors hotkey.Modifier / hotkey.Key so callers can pass it directly to the
// hotkey package without importing config.
type Combo struct {
	Mods []hotkey.Modifier
	Key  hotkey.Key
}

// raw is the YAML structure — field names match the config file keys.
type raw struct {
	Model               string      `yaml:"model"`
	Language            string      `yaml:"language"`
	ModelsDir           string      `yaml:"models_dir"`
	MaxRecordingSeconds int         `yaml:"max_recording_seconds"`
	LogLevel            string      `yaml:"log_level"`
	Ollama              ollamaRaw   `yaml:"ollama"`
	Hotkeys             hotkeysRaw  `yaml:"hotkeys"`
}

type ollamaRaw struct {
	Enabled        bool   `yaml:"enabled"`
	Endpoint       string `yaml:"endpoint"`
	Model          string `yaml:"model"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type hotkeysRaw struct {
	ToggleRecording string `yaml:"toggle_recording"`
	CancelRecording string `yaml:"cancel_recording"`
	ChangeMode      string `yaml:"change_mode"`
}

// Combos holds the parsed hotkey combos derived from HotkeysRaw.
type Combos struct {
	Toggle Combo
	Cancel Combo
	Mode   Combo
}

// Manager owns the parsed config and the fsnotify watcher.
type Manager struct {
	mu       sync.RWMutex
	cfg      raw
	combos   Combos
	dir      string // directory containing config.yaml (also used for state.json)
	onChange []func(newCombos Combos, combosChanged bool, newModel string, modelChanged bool)
	watcher  *fsnotify.Watcher
	debounce *time.Timer
}

var defaults = raw{
	Model:               "small",
	Language:            "auto",
	ModelsDir:           "~/.config/gowhisper/models",
	MaxRecordingSeconds: 120,
	LogLevel:            "info",
	Ollama: ollamaRaw{
		Endpoint:       "http://localhost:11434",
		Model:          "llama3.2:3b",
		TimeoutSeconds: 10,
	},
	Hotkeys: hotkeysRaw{
		ToggleRecording: "option+space",
		CancelRecording: "esc",
		ChangeMode:      "option+shift+k",
	},
}

// Load reads (or creates) the config file at path and starts a file watcher.
// Call Close when done.
func Load(path string) (*Manager, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	// Write defaults if the file doesn't exist yet.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if writeErr := writeDefaults(path); writeErr != nil {
			log.Printf("config: could not write default config: %v", writeErr)
		}
	}

	m := &Manager{dir: dir}
	m.cfg = defaults

	if err := m.reload(path); err != nil {
		log.Printf("config: parse error on load: %v — using defaults", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	// Watch only the specific file, not the whole directory, so state.json
	// changes don't trigger a config reload.
	if err := watcher.Add(path); err != nil {
		_ = watcher.Close()
		return nil, err
	}
	m.watcher = watcher

	go m.watchLoop(path)
	return m, nil
}

// Combos returns the current parsed hotkey combos (safe for concurrent use).
func (m *Manager) Combos() Combos {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.combos
}

// ModelPath returns the absolute path to the model binary.
func (m *Manager) ModelPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return resolveModelPath(m.cfg)
}

// Language returns the configured language string (e.g. "auto", "es").
func (m *Manager) Language() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.Language
}

// Dir returns the config directory (also used for state.json).
func (m *Manager) Dir() string { return m.dir }

// OnChange registers a callback invoked when a reload changes combos or the model.
// combosChanged and modelChanged indicate which parts differ from the previous config.
// The callback runs on the watcher goroutine — it must not block; use go func if needed.
func (m *Manager) OnChange(fn func(newCombos Combos, combosChanged bool, newModel string, modelChanged bool)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = append(m.onChange, fn)
}

// Close stops the file watcher.
func (m *Manager) Close() {
	if m.watcher != nil {
		_ = m.watcher.Close()
	}
}

// watchLoop processes fsnotify events with 200 ms debounce.
func (m *Manager) watchLoop(path string) {
	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				m.mu.Lock()
				if m.debounce != nil {
					m.debounce.Stop()
				}
				m.debounce = time.AfterFunc(200*time.Millisecond, func() {
					m.reloadAndNotify(path)
				})
				m.mu.Unlock()
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("config: watcher error: %v", err)
		}
	}
}

// reloadAndNotify re-parses the config file and fires OnChange callbacks for any
// differences in combos or model.
func (m *Manager) reloadAndNotify(path string) {
	m.mu.Lock()
	oldCombos := m.combos
	oldModel := resolveModelPath(m.cfg)

	if err := m.reload(path); err != nil {
		log.Printf("config: reload error: %v — keeping previous config", err)
		m.mu.Unlock()
		return
	}

	newCombos := m.combos
	newModel := resolveModelPath(m.cfg)
	callbacks := make([]func(Combos, bool, string, bool), len(m.onChange))
	copy(callbacks, m.onChange)
	m.mu.Unlock()

	combosChanged := !combosEqual(oldCombos, newCombos)
	modelChanged := oldModel != newModel
	if combosChanged || modelChanged {
		for _, fn := range callbacks {
			fn(newCombos, combosChanged, newModel, modelChanged)
		}
	}
}

// reload parses path into m.cfg and m.combos. Must be called with m.mu held or during init.
func (m *Manager) reload(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	next := defaults
	if err := yaml.Unmarshal(data, &next); err != nil {
		return err
	}
	applyDefaults(&next)

	combos, errs := parseCombos(next.Hotkeys)
	for _, e := range errs {
		log.Printf("config: hotkey parse error: %v — keeping previous binding", e)
	}
	// For fields that failed to parse, fall back to the current combo.
	if errs != nil {
		prev := m.combos
		if comboZero(combos.Toggle) {
			combos.Toggle = prev.Toggle
		}
		if comboZero(combos.Cancel) {
			combos.Cancel = prev.Cancel
		}
		if comboZero(combos.Mode) {
			combos.Mode = prev.Mode
		}
	}

	warnConflicts(combos)

	m.cfg = next
	m.combos = combos
	return nil
}

// parseCombos parses all three hotkey strings. Returns any errors per-field.
func parseCombos(h hotkeysRaw) (Combos, []error) {
	var c Combos
	var errs []error

	if t, err := parseCombo(h.ToggleRecording); err != nil {
		errs = append(errs, err)
	} else {
		c.Toggle = t
	}
	if cancel, err := parseCombo(h.CancelRecording); err != nil {
		errs = append(errs, err)
	} else {
		c.Cancel = cancel
	}
	if mode, err := parseCombo(h.ChangeMode); err != nil {
		errs = append(errs, err)
	} else {
		c.Mode = mode
	}
	return c, errs
}

// warnConflicts logs a warning if any two actions share the same combo.
func warnConflicts(c Combos) {
	type named struct {
		name  string
		combo Combo
	}
	pairs := []named{{"toggle", c.Toggle}, {"cancel", c.Cancel}, {"mode", c.Mode}}
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if comboEqual(pairs[i].combo, pairs[j].combo) {
				log.Printf("config: hotkey conflict — %s and %s share the same combo; %s wins",
					pairs[i].name, pairs[j].name, pairs[j].name)
			}
		}
	}
}

// applyDefaults fills zero-value fields with defaults.
func applyDefaults(c *raw) {
	if c.Model == "" {
		c.Model = defaults.Model
	}
	if c.Language == "" {
		c.Language = defaults.Language
	}
	if c.ModelsDir == "" {
		c.ModelsDir = defaults.ModelsDir
	}
	if c.MaxRecordingSeconds == 0 {
		c.MaxRecordingSeconds = defaults.MaxRecordingSeconds
	}
	if c.LogLevel == "" {
		c.LogLevel = defaults.LogLevel
	}
	if c.Ollama.Endpoint == "" {
		c.Ollama.Endpoint = defaults.Ollama.Endpoint
	}
	if c.Ollama.Model == "" {
		c.Ollama.Model = defaults.Ollama.Model
	}
	if c.Ollama.TimeoutSeconds == 0 {
		c.Ollama.TimeoutSeconds = defaults.Ollama.TimeoutSeconds
	}
	if c.Hotkeys.ToggleRecording == "" {
		c.Hotkeys.ToggleRecording = defaults.Hotkeys.ToggleRecording
	}
	if c.Hotkeys.CancelRecording == "" {
		c.Hotkeys.CancelRecording = defaults.Hotkeys.CancelRecording
	}
	if c.Hotkeys.ChangeMode == "" {
		c.Hotkeys.ChangeMode = defaults.Hotkeys.ChangeMode
	}
}

// resolveModelPath expands ~ and builds the full path to the model binary.
func resolveModelPath(c raw) string {
	dir := expandTilde(c.ModelsDir)
	return filepath.Join(dir, "ggml-"+c.Model+".bin")
}

// expandTilde replaces a leading ~ with the user's home directory.
func expandTilde(p string) string {
	if !strings.HasPrefix(p, "~") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, p[1:])
}

// comboZero reports whether c has no key set (used to detect a failed parse).
func comboZero(c Combo) bool { return c.Key == 0 && len(c.Mods) == 0 }

// comboEqual compares two Combos field by field (slices are not comparable with ==).
func comboEqual(a, b Combo) bool {
	if a.Key != b.Key || len(a.Mods) != len(b.Mods) {
		return false
	}
	for i := range a.Mods {
		if a.Mods[i] != b.Mods[i] {
			return false
		}
	}
	return true
}

// combosEqual reports whether two Combos structs are identical.
func combosEqual(a, b Combos) bool {
	return comboEqual(a.Toggle, b.Toggle) &&
		comboEqual(a.Cancel, b.Cancel) &&
		comboEqual(a.Mode, b.Mode)
}

// writeDefaults writes a default config.yaml to path.
func writeDefaults(path string) error {
	const template = `# GoWhisper configuration
# All hotkeys are configurable. Restart not required — changes are applied on save.

model: small              # tiny | small | medium
language: auto            # auto | es | en
models_dir: "~/.config/gowhisper/models"
max_recording_seconds: 120
log_level: info

ollama:
  enabled: false
  endpoint: "http://localhost:11434"
  model: "llama3.2:3b"
  timeout_seconds: 10

hotkeys:
  toggle_recording: "option+space"
  cancel_recording: "esc"
  change_mode: "option+shift+k"
`
	return os.WriteFile(path, []byte(template), 0o644)
}
