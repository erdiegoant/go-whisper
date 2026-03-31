package sound

import "os/exec"

// Kind identifies which sound to play.
type Kind int

const (
	Start  Kind = iota // recording started
	Stop               // recording stopped / paste completed
	Cancel             // recording cancelled
)

// Play fires afplay in the background. Non-blocking — errors are silently ignored.
func Play(k Kind) {
	path, ok := soundPath(k)
	if !ok {
		return
	}
	go func() { _ = exec.Command("afplay", path).Run() }()
}

// PlaySync plays a sound and blocks until afplay finishes.
// Use for the start cue so the mic doesn't capture the sound.
func PlaySync(k Kind) {
	path, ok := soundPath(k)
	if !ok {
		return
	}
	_ = exec.Command("afplay", path).Run()
}

func soundPath(k Kind) (string, bool) {
	switch k {
	case Start:
		return "/System/Library/Sounds/Submarine.aiff", true
	case Stop:
		return "/System/Library/Sounds/Bottle.aiff", true
	case Cancel:
		return "/System/Library/Sounds/Basso.aiff", true
	}
	return "", false
}
