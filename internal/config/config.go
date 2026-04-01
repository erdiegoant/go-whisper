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

	"github.com/erdiegoant/gowhisper/internal/mode"
)

// Combo is the parsed representation of a hotkey string (e.g. "option+shift+k").
type Combo struct {
	Mods []hotkey.Modifier
	Key  hotkey.Key
}

// raw is the YAML structure — field names match the config file keys.
type raw struct {
	Model                string     `yaml:"model"`
	Language             string     `yaml:"language"`
	ModelsDir            string     `yaml:"models_dir"`
	MaxRecordingSeconds  int        `yaml:"max_recording_seconds"`
	LogLevel             string     `yaml:"log_level"`
	SoundEnabled         *bool      `yaml:"sound_enabled"`
	NotificationsEnabled *bool      `yaml:"notifications_enabled"`
	Claude               claudeRaw  `yaml:"claude"`
	Ollama               ollamaRaw  `yaml:"ollama"`
	Hotkeys              hotkeysRaw `yaml:"hotkeys"`
	Modes                []modeRaw  `yaml:"modes"`
}

type claudeRaw struct {
	APIKey         string `yaml:"api_key"`
	Model          string `yaml:"model"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type ollamaRaw struct {
	Model          string `yaml:"model"`
	Host           string `yaml:"host"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type hotkeysRaw struct {
	ToggleRecording string `yaml:"toggle_recording"`
	CancelRecording string `yaml:"cancel_recording"`
	ChangeMode      string `yaml:"change_mode"`
}

type modeRaw struct {
	Name      string `yaml:"name"`
	Language  string `yaml:"language"`
	Translate bool   `yaml:"translate"`
	Prompt    string `yaml:"prompt"`
}

// Combos holds the parsed hotkey combos.
type Combos struct {
	Toggle Combo
	Cancel Combo
	Mode   Combo
}

// ClaudeConfig holds the resolved Claude API settings.
type ClaudeConfig struct {
	APIKey         string
	Model          string
	TimeoutSeconds int
}

// OllamaConfig holds the Ollama local LLM settings.
type OllamaConfig struct {
	Model          string
	Host           string
	TimeoutSeconds int
}

// ChangeEvent is passed to OnChange callbacks describing what changed on reload.
type ChangeEvent struct {
	Combos        Combos
	CombosChanged bool
	Model         string
	ModelChanged  bool
	Modes         []mode.Mode
	ModesChanged  bool
}

// Manager owns the parsed config and the fsnotify watcher.
type Manager struct {
	mu       sync.RWMutex
	cfg      raw
	combos   Combos
	dir      string
	onChange []func(ChangeEvent)
	watcher  *fsnotify.Watcher
	debounce *time.Timer
}

var defaults = raw{
	Model:               "small",
	Language:            "auto",
	ModelsDir:           "~/.config/gowhisper/models",
	MaxRecordingSeconds: 120,
	LogLevel:            "info",
	Claude: claudeRaw{
		Model:          "claude-haiku-4-5-20251001",
		TimeoutSeconds: 15,
	},
	Hotkeys: hotkeysRaw{
		ToggleRecording: "option+space",
		CancelRecording: "esc",
		ChangeMode:      "option+shift+k",
	},
}

// Load reads (or creates) the config file at path and starts a file watcher.
func Load(path string) (*Manager, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

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

// Language returns the configured language string.
func (m *Manager) Language() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.Language
}

// ClaudeConfig returns the resolved Claude API settings.
// api_key in config.yaml takes precedence; falls back to ANTHROPIC_API_KEY env var.
func (m *Manager) ClaudeConfig() ClaudeConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := m.cfg.Claude.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	return ClaudeConfig{
		APIKey:         key,
		Model:          m.cfg.Claude.Model,
		TimeoutSeconds: m.cfg.Claude.TimeoutSeconds,
	}
}

// OllamaConfig returns the Ollama local LLM settings.
// Model is empty string when Ollama is not configured.
func (m *Manager) OllamaConfig() OllamaConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return OllamaConfig{
		Model:          m.cfg.Ollama.Model,
		Host:           m.cfg.Ollama.Host,
		TimeoutSeconds: m.cfg.Ollama.TimeoutSeconds,
	}
}

// Modes converts the raw modes list to []mode.Mode.
// Returns mode.DefaultModes if no modes are defined in config.
func (m *Manager) Modes() []mode.Mode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return parseModes(m.cfg.Modes)
}

// LogLevel returns the configured log level string ("info" or "debug").
func (m *Manager) LogLevel() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.LogLevel
}

// MaxRecordingSeconds returns the hard cap on recording duration.
func (m *Manager) MaxRecordingSeconds() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.MaxRecordingSeconds
}

// SoundEnabled returns true unless sound_enabled is explicitly false in config.
func (m *Manager) SoundEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.SoundEnabled == nil || *m.cfg.SoundEnabled
}

// NotificationsEnabled returns true unless notifications_enabled is explicitly false in config.
func (m *Manager) NotificationsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.NotificationsEnabled == nil || *m.cfg.NotificationsEnabled
}

// Dir returns the config directory (also used for state.json).
func (m *Manager) Dir() string { return m.dir }

// OnChange registers a callback invoked when a reload produces any change.
// The callback runs on the watcher goroutine — it must not block.
func (m *Manager) OnChange(fn func(ChangeEvent)) {
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

// reloadAndNotify re-parses config and fires OnChange if anything changed.
func (m *Manager) reloadAndNotify(path string) {
	m.mu.Lock()
	oldCombos := m.combos
	oldModel := resolveModelPath(m.cfg)
	oldModes := m.cfg.Modes

	if err := m.reload(path); err != nil {
		log.Printf("config: reload error: %v — keeping previous config", err)
		m.mu.Unlock()
		return
	}

	evt := ChangeEvent{
		Combos:        m.combos,
		CombosChanged: !combosEqual(oldCombos, m.combos),
		Model:         resolveModelPath(m.cfg),
		ModelChanged:  resolveModelPath(m.cfg) != oldModel,
		Modes:         parseModes(m.cfg.Modes),
		ModesChanged:  !modesEqual(oldModes, m.cfg.Modes),
	}
	callbacks := make([]func(ChangeEvent), len(m.onChange))
	copy(callbacks, m.onChange)
	m.mu.Unlock()

	if evt.CombosChanged || evt.ModelChanged || evt.ModesChanged {
		for _, fn := range callbacks {
			fn(evt)
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

// parseModes converts []modeRaw to []mode.Mode, applying language default.
// Returns mode.DefaultModes if the list is empty.
func parseModes(raw []modeRaw) []mode.Mode {
	if len(raw) == 0 {
		return mode.DefaultModes
	}
	modes := make([]mode.Mode, len(raw))
	for i, r := range raw {
		lang := r.Language
		if lang == "" {
			lang = "auto"
		}
		modes[i] = mode.Mode{
			Name:      r.Name,
			Language:  lang,
			Translate: r.Translate,
			Prompt:    r.Prompt,
		}
	}
	return modes
}

// modesEqual reports whether two raw mode lists are identical.
func modesEqual(a, b []modeRaw) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// parseCombos parses all three hotkey strings.
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
	if c.SoundEnabled == nil {
		v := true
		c.SoundEnabled = &v
	}
	if c.NotificationsEnabled == nil {
		v := true
		c.NotificationsEnabled = &v
	}
	if c.Claude.Model == "" {
		c.Claude.Model = defaults.Claude.Model
	}
	if c.Claude.TimeoutSeconds == 0 {
		c.Claude.TimeoutSeconds = defaults.Claude.TimeoutSeconds
	}
	if c.Ollama.Host == "" {
		c.Ollama.Host = "http://localhost:11434"
	}
	if c.Ollama.TimeoutSeconds == 0 {
		c.Ollama.TimeoutSeconds = 30
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
	return filepath.Join(expandTilde(c.ModelsDir), "ggml-"+c.Model+".bin")
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

// comboZero reports whether c has no key set.
func comboZero(c Combo) bool { return c.Key == 0 && len(c.Mods) == 0 }

// comboEqual compares two Combo values field by field.
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
sound_enabled: true
notifications_enabled: true

claude:
  api_key: ""             # leave empty to use ANTHROPIC_API_KEY environment variable
  model: "claude-haiku-4-5-20251001"
  timeout_seconds: 15

hotkeys:
  toggle_recording: "option+space"
  cancel_recording: "esc"
  change_mode: "option+shift+k"

# Custom modes — uncomment and edit to add your own.
# Each mode can optionally override the cleanup system prompt sent to Claude.
# Omitting "prompt" uses the built-in cleanup prompt (removes filler words, fixes punctuation).
#
# modes:
#   - name: Standard
#     language: auto
#     translate: false
#
#   - name: Translate
#     language: es
#     translate: true
#
#   - name: Formal
#     language: auto
#     translate: false
#     prompt: "Rewrite this transcript in a formal professional tone. Preserve all technical terms and code identifiers. Return only the result."
#
#   - name: Bullets
#     language: auto
#     translate: false
#     prompt: "Convert this dictation into a concise bullet point list. Preserve technical terms. Return only the result."
`
	return os.WriteFile(path, []byte(template), 0o644)
}
