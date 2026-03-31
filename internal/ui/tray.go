package ui

import (
	"log"
	"os/exec"

	"fyne.io/systray"
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

// Done returns a channel that closes when the user clicks Quit.
func (t *Tray) Done() <-chan struct{} {
	return t.quitCh
}
