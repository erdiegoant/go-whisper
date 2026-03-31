package hotkey

import (
	"fmt"
	"log"
	"sync"

	"golang.design/x/hotkey"
)

// Action represents what a hotkey press should trigger.
type Action int

const (
	ActionToggle Action = iota // start or stop recording
	ActionCancel               // discard active recording
	ActionMode                 // cycle to next mode
)

// Combo is a parsed hotkey combination. config.Combo mirrors this type so the
// config package can produce values without the hotkey package importing config.
type Combo struct {
	Mods []hotkey.Modifier
	Key  hotkey.Key
}

// Manager registers global hotkeys and emits Actions on a channel.
// Toggle and Mode are always registered. Cancel (Esc by default) is registered
// only while recording via EnableCancel/DisableCancel.
type Manager struct {
	mu          sync.Mutex
	ch          chan Action
	toggleHK    *hotkey.Hotkey
	modeHK      *hotkey.Hotkey
	cancelHK    *hotkey.Hotkey // nil when not recording
	cancelCombo Combo          // stored so the next EnableCancel uses the current config
}

// New creates a Manager and registers the always-on hotkeys.
func New(toggle, mode Combo) (*Manager, error) {
	m := &Manager{
		ch:          make(chan Action, 4),
		cancelCombo: Combo{Key: hotkey.KeyEscape},
	}

	if err := m.registerToggle(toggle); err != nil {
		return nil, err
	}
	if err := m.registerMode(mode); err != nil {
		m.Close()
		return nil, err
	}

	log.Printf("hotkey: registered toggle=%v mode=%v — Esc active only while recording", toggle, mode)
	return m, nil
}

// Rebind atomically swaps the toggle and mode hotkeys to new combos.
// There is a brief (~1–2 ms) gap between unregister and register during which
// keypresses are not captured; this is an unavoidable limitation of the Carbon API.
// If called while recording, cancelHK is left untouched; cancelCombo is updated for
// the next EnableCancel call.
func (m *Manager) Rebind(toggle, mode, cancel Combo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Unregister and re-register toggle.
	if m.toggleHK != nil {
		if err := m.toggleHK.Unregister(); err != nil {
			log.Printf("hotkey: rebind: unregister toggle: %v", err)
		}
		m.toggleHK = nil
	}
	if err := m.registerToggleLocked(toggle); err != nil {
		log.Printf("hotkey: rebind: register toggle: %v", err)
	}

	// Unregister and re-register mode.
	if m.modeHK != nil {
		if err := m.modeHK.Unregister(); err != nil {
			log.Printf("hotkey: rebind: unregister mode: %v", err)
		}
		m.modeHK = nil
	}
	if err := m.registerModeLocked(mode); err != nil {
		log.Printf("hotkey: rebind: register mode: %v", err)
	}

	// Update stored cancel combo; active cancelHK (if recording) is left alone.
	m.cancelCombo = cancel
}

// EnableCancel registers the cancel hotkey. Call when entering RECORDING state.
func (m *Manager) EnableCancel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancelHK != nil {
		return
	}
	hk := hotkey.New(m.cancelCombo.Mods, m.cancelCombo.Key)
	if err := hk.Register(); err != nil {
		log.Printf("hotkey: register cancel: %v", err)
		return
	}
	m.cancelHK = hk
	go func() {
		for range hk.Keydown() {
			m.ch <- ActionCancel
		}
	}()
}

// DisableCancel unregisters the cancel hotkey. Call when leaving RECORDING state.
func (m *Manager) DisableCancel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancelHK == nil {
		return
	}
	if err := m.cancelHK.Unregister(); err != nil {
		log.Printf("hotkey: unregister cancel: %v", err)
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
	m.mu.Lock()
	defer m.mu.Unlock()
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

// --- internal helpers ---

func (m *Manager) registerToggle(c Combo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registerToggleLocked(c)
}

func (m *Manager) registerToggleLocked(c Combo) error {
	hk := hotkey.New(c.Mods, c.Key)
	if err := hk.Register(); err != nil {
		return fmt.Errorf("hotkey: register toggle: %w", err)
	}
	m.toggleHK = hk
	go func() {
		for range hk.Keydown() {
			m.ch <- ActionToggle
		}
	}()
	return nil
}

func (m *Manager) registerMode(c Combo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registerModeLocked(c)
}

func (m *Manager) registerModeLocked(c Combo) error {
	hk := hotkey.New(c.Mods, c.Key)
	if err := hk.Register(); err != nil {
		return fmt.Errorf("hotkey: register mode: %w", err)
	}
	m.modeHK = hk
	go func() {
		for range hk.Keydown() {
			m.ch <- ActionMode
		}
	}()
	return nil
}
