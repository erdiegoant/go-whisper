package hotkey

import (
	"fmt"
	"log"

	"golang.design/x/hotkey"
)

// Action represents what a hotkey press should trigger.
type Action int

const (
	ActionToggle Action = iota // start or stop recording
	ActionCancel               // discard active recording
	ActionMode                 // cycle to next mode
)

// Manager registers global hotkeys and emits Actions on a channel.
// Toggle (⌥Space) and Mode (⌥⇧K) are always registered.
// Cancel (Esc) is registered only while recording via EnableCancel/DisableCancel.
type Manager struct {
	ch       chan Action
	toggleHK *hotkey.Hotkey
	modeHK   *hotkey.Hotkey
	cancelHK *hotkey.Hotkey // nil when not recording
}

// New creates a Manager and registers the always-on hotkeys.
func New() (*Manager, error) {
	m := &Manager{
		ch: make(chan Action, 4),
	}

	toggle := hotkey.New([]hotkey.Modifier{hotkey.ModOption}, hotkey.KeySpace)
	if err := toggle.Register(); err != nil {
		return nil, fmt.Errorf("hotkey: register toggle: %w", err)
	}
	m.toggleHK = toggle
	go func() {
		for range toggle.Keydown() {
			m.ch <- ActionToggle
		}
	}()

	mode := hotkey.New([]hotkey.Modifier{hotkey.ModOption, hotkey.ModShift}, hotkey.KeyK)
	if err := mode.Register(); err != nil {
		m.Close()
		return nil, fmt.Errorf("hotkey: register mode: %w", err)
	}
	m.modeHK = mode
	go func() {
		for range mode.Keydown() {
			m.ch <- ActionMode
		}
	}()

	log.Println("hotkey: registered ⌥Space (toggle), ⌥⇧K (mode) — Esc active only while recording")
	return m, nil
}

// EnableCancel registers the Esc hotkey. Call when entering RECORDING state.
func (m *Manager) EnableCancel() {
	if m.cancelHK != nil {
		return
	}
	hk := hotkey.New([]hotkey.Modifier{}, hotkey.Key(0x35))
	if err := hk.Register(); err != nil {
		log.Printf("hotkey: register Esc: %v", err)
		return
	}
	m.cancelHK = hk
	go func() {
		for range hk.Keydown() {
			m.ch <- ActionCancel
		}
	}()
}

// DisableCancel unregisters the Esc hotkey. Call when leaving RECORDING state.
func (m *Manager) DisableCancel() {
	if m.cancelHK == nil {
		return
	}
	if err := m.cancelHK.Unregister(); err != nil {
		log.Printf("hotkey: unregister Esc: %v", err)
	}
	m.cancelHK = nil
}

// C returns the channel that emits Actions on each keypress.
func (m *Manager) C() <-chan Action {
	return m.ch
}

// Close unregisters all hotkeys.
func (m *Manager) Close() {
	m.DisableCancel()
	if m.toggleHK != nil {
		if err := m.toggleHK.Unregister(); err != nil {
			log.Printf("hotkey: unregister toggle: %v", err)
		}
		m.toggleHK = nil
	}
	if m.modeHK != nil {
		if err := m.modeHK.Unregister(); err != nil {
			log.Printf("hotkey: unregister mode: %v", err)
		}
		m.modeHK = nil
	}
}
