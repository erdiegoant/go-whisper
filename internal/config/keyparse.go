package config

import (
	"fmt"
	"strings"

	"golang.design/x/hotkey"
)

var modMap = map[string]hotkey.Modifier{
	"option": hotkey.ModOption,
	"shift":  hotkey.ModShift,
	"ctrl":   hotkey.ModCtrl,
	"cmd":    hotkey.ModCmd,
}

var keyMap = map[string]hotkey.Key{
	"space":  hotkey.KeySpace,
	"esc":    hotkey.KeyEscape,
	"return": hotkey.KeyReturn,
	"delete": hotkey.KeyDelete,
	"tab":    hotkey.KeyTab,
	"left":   hotkey.KeyLeft,
	"right":  hotkey.KeyRight,
	"up":     hotkey.KeyUp,
	"down":   hotkey.KeyDown,
	"a": hotkey.KeyA, "b": hotkey.KeyB, "c": hotkey.KeyC, "d": hotkey.KeyD,
	"e": hotkey.KeyE, "f": hotkey.KeyF, "g": hotkey.KeyG, "h": hotkey.KeyH,
	"i": hotkey.KeyI, "j": hotkey.KeyJ, "k": hotkey.KeyK, "l": hotkey.KeyL,
	"m": hotkey.KeyM, "n": hotkey.KeyN, "o": hotkey.KeyO, "p": hotkey.KeyP,
	"q": hotkey.KeyQ, "r": hotkey.KeyR, "s": hotkey.KeyS, "t": hotkey.KeyT,
	"u": hotkey.KeyU, "v": hotkey.KeyV, "w": hotkey.KeyW, "x": hotkey.KeyX,
	"y": hotkey.KeyY, "z": hotkey.KeyZ,
	"0": hotkey.Key0, "1": hotkey.Key1, "2": hotkey.Key2, "3": hotkey.Key3,
	"4": hotkey.Key4, "5": hotkey.Key5, "6": hotkey.Key6, "7": hotkey.Key7,
	"8": hotkey.Key8, "9": hotkey.Key9,
	"f1": hotkey.KeyF1, "f2": hotkey.KeyF2, "f3": hotkey.KeyF3, "f4": hotkey.KeyF4,
	"f5": hotkey.KeyF5, "f6": hotkey.KeyF6, "f7": hotkey.KeyF7, "f8": hotkey.KeyF8,
	"f9": hotkey.KeyF9, "f10": hotkey.KeyF10, "f11": hotkey.KeyF11, "f12": hotkey.KeyF12,
}

// parseCombo parses a string like "option+shift+k" into a Combo.
// Tokens are split on "+", lowercased, and matched against known modifiers and keys.
// The last token that is not a modifier is treated as the key.
func parseCombo(s string) (Combo, error) {
	tokens := strings.Split(strings.ToLower(strings.TrimSpace(s)), "+")
	var mods []hotkey.Modifier
	var key hotkey.Key
	hasKey := false

	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if mod, ok := modMap[tok]; ok {
			mods = append(mods, mod)
			continue
		}
		if k, ok := keyMap[tok]; ok {
			key = k
			hasKey = true
			continue
		}
		return Combo{}, fmt.Errorf("config: unknown key token %q in %q", tok, s)
	}

	if !hasKey {
		return Combo{}, fmt.Errorf("config: no key found in %q", s)
	}
	return Combo{Mods: mods, Key: key}, nil
}
