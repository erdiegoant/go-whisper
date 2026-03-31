package mode

import "log"

// Mode describes a transcription mode.
type Mode struct {
	Name      string
	Language  string // "auto", "es", etc.
	Translate bool   // true = Whisper native ES→EN translation
	Prompt    string // overrides llm.CleanupPrompt when non-empty; empty = use default
}

// DefaultModes is the fallback used when config.yaml has no modes block.
var DefaultModes = []Mode{
	{Name: "Standard", Language: "auto", Translate: false},
	{Name: "Translate", Language: "es", Translate: true},
}

// Manager cycles through a list of modes, tracking the active mode by name.
type Manager struct {
	modes []Mode
	name  string // name of the active mode
}

// NewManager creates a Manager from the given list.
// Falls back to DefaultModes if modes is empty.
func NewManager(modes []Mode) *Manager {
	if len(modes) == 0 {
		modes = DefaultModes
	}
	return &Manager{modes: modes, name: modes[0].Name}
}

// Current returns the active mode.
// If the stored name is no longer in the list (e.g. after a reload), returns the first mode.
func (m *Manager) Current() Mode {
	for _, mode := range m.modes {
		if mode.Name == m.name {
			return mode
		}
	}
	return m.modes[0]
}

// Next advances to the next mode and returns it.
func (m *Manager) Next() Mode {
	idx := 0
	for i, mode := range m.modes {
		if mode.Name == m.name {
			idx = i
			break
		}
	}
	next := m.modes[(idx+1)%len(m.modes)]
	m.name = next.Name
	return next
}

// SetByName sets the active mode by name.
// Returns false if no matching mode is found (active mode unchanged).
func (m *Manager) SetByName(name string) bool {
	for _, mode := range m.modes {
		if mode.Name == name {
			m.name = name
			return true
		}
	}
	return false
}

// Reload replaces the modes list. If the current mode name exists in the new
// list it stays active; otherwise falls back to the first mode.
func (m *Manager) Reload(modes []Mode) {
	if len(modes) == 0 {
		modes = DefaultModes
	}
	m.modes = modes
	for _, mode := range modes {
		if mode.Name == m.name {
			return
		}
	}
	log.Printf("mode: %q not found after config reload — switching to %s", m.name, modes[0].Name)
	m.name = modes[0].Name
}

// All returns the full list of available modes.
func (m *Manager) All() []Mode { return m.modes }
