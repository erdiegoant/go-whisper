package ui

import (
	"fyne.io/systray"
)

const (
	iconIdle       = "⬜" // shown in menu title — actual icon is set via systray.SetTitle
	iconRecording  = "🔴"
	iconProcessing = "⏳"
)

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
		systray.SetTitle("⬜ GoWhisper")
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

// SetIdle updates the tray icon to the idle state.
func (t *Tray) SetIdle(mode string) {
	label := "⬜ " + mode
	systray.SetTitle(label)
	systray.SetTooltip("GoWhisper — idle")
}

// SetRecording updates the tray icon to the recording state.
func (t *Tray) SetRecording(mode string) {
	systray.SetTitle("🔴 " + mode)
	systray.SetTooltip("GoWhisper — recording")
}

// SetProcessing updates the tray icon to the processing state.
func (t *Tray) SetProcessing(mode string) {
	systray.SetTitle("⏳ " + mode)
	systray.SetTooltip("GoWhisper — transcribing")
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
