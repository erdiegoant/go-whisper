package ui

import (
	"fmt"
	"log"
	"os/exec"
	"sync"

	"fyne.io/systray"

	"github.com/erdiegoant/gowhisper/internal/mode"
	"github.com/erdiegoant/gowhisper/internal/models"
)

const (
	iconIdle       = "⚫"
	iconRecording  = "🔴"
	iconProcessing = "⏳"
)

// ModeItem describes a single entry in the Mode submenu.
type ModeItem struct {
	Name    string // display name
	Tooltip string // shown on hover (prompt preview or description)
}

// Tray manages the menubar status item.
type Tray struct {
	quitCh chan struct{}
}

// New creates a Tray. Call Run from the main goroutine to start the event loop.
func New() *Tray {
	return &Tray{quitCh: make(chan struct{})}
}

// Run starts the systray event loop. Must be called from the main goroutine on
// macOS; it blocks until the user clicks Quit. onReady is called once the tray
// is initialised and the run loop is live.
func (t *Tray) Run(onReady func()) {
	systray.Run(func() {
		systray.SetTitle("⚫ GoWhisper")
		systray.SetTooltip("GoWhisper — idle")

		mQuit := systray.AddMenuItem("Quit", "Quit GoWhisper")
		go func() {
			<-mQuit.ClickedCh
			systray.Quit()
			close(t.quitCh)
		}()

		if onReady != nil {
			onReady()
		}
	}, func() {})
}

// SetIdle updates the tray title to the idle state.
func (t *Tray) SetIdle(modeName string) {
	systray.SetTitle("⚫ " + modeName)
	systray.SetTooltip("GoWhisper — idle")
}

// SetRecording updates the tray title to the recording state.
func (t *Tray) SetRecording(modeName string) {
	systray.SetTitle("🔴 " + modeName)
	systray.SetTooltip("GoWhisper — recording")
}

// SetProcessing updates the tray title to the processing state.
func (t *Tray) SetProcessing(modeName string) {
	systray.SetTitle("⏳ " + modeName)
	systray.SetTooltip("GoWhisper — transcribing")
}

// AddOpenConfigItem adds an "Open Config" menu item that opens path in the
// default application for .yaml files.
// Must be called after Run's onReady fires.
func (t *Tray) AddOpenConfigItem(path string) {
	item := systray.AddMenuItem("Open Config", "Edit ~/.config/gowhisper/config.yaml")
	go func() {
		for range item.ClickedCh {
			if err := exec.Command("open", path).Start(); err != nil {
				log.Printf("tray: open config: %v", err)
			}
		}
	}()
}

// ModeItems converts a []mode.Mode slice to []ModeItem, building tooltips.
func ModeItems(modes []mode.Mode) []ModeItem {
	items := make([]ModeItem, len(modes))
	for i, m := range modes {
		items[i] = ModeItem{Name: m.Name, Tooltip: modeTooltip(m)}
	}
	return items
}

// modeTooltip returns a short description used as the tray submenu tooltip.
func modeTooltip(m mode.Mode) string {
	if m.Prompt != "" {
		if len(m.Prompt) > 60 {
			return m.Prompt[:60] + "…"
		}
		return m.Prompt
	}
	if m.Translate {
		return m.Name + " — ES→EN (Whisper native)"
	}
	return m.Name + " — auto transcription"
}

// AddModeMenu adds a "Mode" submenu listing the given modes.
// The active mode shows a checkmark. Clicking a mode calls onSelect with the mode name.
// Returns an update function — call it with the active mode name whenever the active
// mode changes (hotkey cycle, config reload, or tray click) to refresh checkmarks.
// Must be called after Run's onReady fires.
func (t *Tray) AddModeMenu(modes []ModeItem, onSelect func(name string)) func(active string) {
	parent := systray.AddMenuItem("Mode", "Select transcription mode")
	items := make([]*systray.MenuItem, len(modes))
	for i, m := range modes {
		item := parent.AddSubMenuItem(m.Name, m.Tooltip)
		items[i] = item
		go func(item *systray.MenuItem, name string) {
			for range item.ClickedCh {
				onSelect(name)
			}
		}(item, m.Name)
	}

	update := func(active string) {
		for i, item := range items {
			if modes[i].Name == active {
				item.Check()
			} else {
				item.Uncheck()
			}
		}
	}
	return update
}

// AddDeviceMenu adds a "Microphone" submenu listing the given device names.
// onSelect is called with the chosen name; "Default" means use the system default.
// Must be called after Run's onReady fires.
func (t *Tray) AddDeviceMenu(devices []string, onSelect func(name string)) {
	micItem := systray.AddMenuItem("Microphone", "Select input device")
	items := make([]*systray.MenuItem, len(devices))
	for i, name := range devices {
		item := micItem.AddSubMenuItem(name, "")
		items[i] = item
		go func(item *systray.MenuItem, name string) {
			for range item.ClickedCh {
				for _, it := range items {
					it.Uncheck()
				}
				item.Check()
				onSelect(name)
			}
		}(item, name)
	}
}

// AddCleanupToggle adds a "Cleanup" menu item that toggles LLM post-processing.
// enabled is the initial state. onToggle is called with the new enabled value on each click.
// Returns an update function — call it with the current enabled state to refresh the checkmark.
// Must be called after Run's onReady fires.
func (t *Tray) AddCleanupToggle(enabled bool, onToggle func(bool)) func(bool) {
	item := systray.AddMenuItem("Cleanup", "Toggle Claude transcript cleanup")
	if enabled {
		item.Check()
	}
	current := enabled
	go func() {
		for range item.ClickedCh {
			current = !current
			if current {
				item.Check()
			} else {
				item.Uncheck()
			}
			onToggle(current)
		}
	}()
	return func(v bool) {
		if v {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
}

// Done returns a channel that closes when the user clicks Quit.
func (t *Tray) Done() <-chan struct{} {
	return t.quitCh
}

// ModelMenu manages the "Models" submenu in the tray.
type ModelMenu struct {
	mu     sync.Mutex
	parent *systray.MenuItem
	items  map[string]*systray.MenuItem // size → item
}

// AddModelMenu adds a "Models" submenu listing tiny/small/medium models.
// statuses describes the initial local state; currentModel is the active size.
// onSelect is called with the model size string when the user clicks an item.
// Must be called after Run's onReady fires.
func (t *Tray) AddModelMenu(statuses []models.ModelStatus, currentModel string, onSelect func(size string)) *ModelMenu {
	parent := systray.AddMenuItem("Models", "Switch or download Whisper models")
	mm := &ModelMenu{
		parent: parent,
		items:  make(map[string]*systray.MenuItem),
	}

	for _, s := range statuses {
		title := modelItemTitle(s, currentModel)
		item := parent.AddSubMenuItem(title, "")
		mm.items[s.Size] = item
		if s.Size == currentModel {
			item.Check()
		}
		go func(item *systray.MenuItem, size string) {
			for range item.ClickedCh {
				onSelect(size)
			}
		}(item, s.Size)
	}

	return mm
}

// Update refreshes all item titles and checkmarks based on new statuses.
// Safe to call from any goroutine.
func (mm *ModelMenu) Update(statuses []models.ModelStatus, currentModel string) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	for _, s := range statuses {
		item, ok := mm.items[s.Size]
		if !ok {
			continue
		}
		item.Enable()
		item.SetTitle(modelItemTitle(s, currentModel))
		if s.Size == currentModel {
			item.Check()
		} else {
			item.Uncheck()
		}
	}
}

// SetDownloadProgress updates the title of the given model item to show
// download progress. pct is in [0, 1]. Safe to call from any goroutine.
func (mm *ModelMenu) SetDownloadProgress(size string, pct float64) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	item, ok := mm.items[size]
	if !ok {
		return
	}
	item.SetTitle(fmt.Sprintf("⏳ %s %.0f%%", size, pct*100))
	item.Disable()
}

// SetHasUpdates toggles the update badge on the parent "Models" item title.
// Safe to call from any goroutine.
func (mm *ModelMenu) SetHasUpdates(v bool) {
	if v {
		mm.parent.SetTitle("Models ●")
	} else {
		mm.parent.SetTitle("Models")
	}
}

// modelItemTitle returns the display title for a model menu item.
func modelItemTitle(s models.ModelStatus, currentModel string) string {
	switch {
	case s.HasUpdate && s.Size == currentModel:
		return "⬆ " + s.Size + " (update)"
	case s.HasUpdate:
		return "⬆ " + s.Size
	case !s.Installed:
		return "⬇ " + s.Size
	default:
		return "  " + s.Size
	}
}
