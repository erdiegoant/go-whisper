package mode

// Mode describes a transcription mode.
type Mode struct {
	Name      string
	Language  string // "auto", "es", etc.
	Translate bool   // true = Whisper native ES→EN translation
}

// All is the ordered list of modes cycled by ⌥⇧K.
var All = []Mode{
	{Name: "Standard", Language: "auto", Translate: false},
	{Name: "Translate", Language: "es", Translate: true},
}

// Manager cycles through All.
type Manager struct {
	idx int
}

// Current returns the active mode.
func (m *Manager) Current() Mode { return All[m.idx] }

// Next advances to the next mode and returns it.
func (m *Manager) Next() Mode {
	m.idx = (m.idx + 1) % len(All)
	return All[m.idx]
}

// SetByName sets the active mode to the one with the given name.
// Returns false if no matching mode is found (index is unchanged).
func (m *Manager) SetByName(name string) bool {
	for i, mode := range All {
		if mode.Name == name {
			m.idx = i
			return true
		}
	}
	return false
}
