package notify

import (
	"fmt"
	"os/exec"
)

// Show displays a macOS desktop notification via osascript.
// body is truncated to 100 characters. Non-blocking — errors are silently ignored.
func Show(title, body string) {
	if len(body) > 100 {
		body = body[:100]
	}
	script := fmt.Sprintf(`display notification %q with title %q`, body, title)
	go func() { _ = exec.Command("osascript", "-e", script).Run() }()
}
