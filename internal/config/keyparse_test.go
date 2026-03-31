package config

import (
	"testing"

	"golang.design/x/hotkey"
)

func TestParseCombo_simpleKey(t *testing.T) {
	c, err := parseCombo("space")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Key != hotkey.KeySpace {
		t.Errorf("expected KeySpace, got %v", c.Key)
	}
	if len(c.Mods) != 0 {
		t.Errorf("expected no mods, got %v", c.Mods)
	}
}

func TestParseCombo_withModifiers(t *testing.T) {
	c, err := parseCombo("option+shift+k")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Key != hotkey.KeyK {
		t.Errorf("expected KeyK, got %v", c.Key)
	}
	if len(c.Mods) != 2 {
		t.Errorf("expected 2 mods, got %d", len(c.Mods))
	}
}

func TestParseCombo_singleModAndKey(t *testing.T) {
	c, err := parseCombo("option+space")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Key != hotkey.KeySpace {
		t.Errorf("expected KeySpace, got %v", c.Key)
	}
	if len(c.Mods) != 1 || c.Mods[0] != hotkey.ModOption {
		t.Errorf("expected [ModOption], got %v", c.Mods)
	}
}

func TestParseCombo_caseInsensitive(t *testing.T) {
	c, err := parseCombo("Option+Shift+K")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Key != hotkey.KeyK {
		t.Errorf("expected KeyK, got %v", c.Key)
	}
}

func TestParseCombo_escape(t *testing.T) {
	c, err := parseCombo("esc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Key != hotkey.KeyEscape {
		t.Errorf("expected KeyEscape, got %v", c.Key)
	}
}

func TestParseCombo_unknownToken(t *testing.T) {
	_, err := parseCombo("option+super+k")
	if err == nil {
		t.Fatal("expected error for unknown token 'super'")
	}
}

func TestParseCombo_noKey(t *testing.T) {
	_, err := parseCombo("option+shift")
	if err == nil {
		t.Fatal("expected error when no key token present")
	}
}

func TestParseCombo_functionKey(t *testing.T) {
	c, err := parseCombo("f5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Key != hotkey.KeyF5 {
		t.Errorf("expected KeyF5, got %v", c.Key)
	}
}

func TestParseCombo_digitKey(t *testing.T) {
	c, err := parseCombo("cmd+1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Key != hotkey.Key1 {
		t.Errorf("expected Key1, got %v", c.Key)
	}
	if len(c.Mods) != 1 || c.Mods[0] != hotkey.ModCmd {
		t.Errorf("expected [ModCmd], got %v", c.Mods)
	}
}

func TestComboEqual(t *testing.T) {
	a, _ := parseCombo("option+shift+k")
	b, _ := parseCombo("option+shift+k")
	if !comboEqual(a, b) {
		t.Error("expected equal combos to match")
	}
}

func TestComboEqual_differentKey(t *testing.T) {
	a, _ := parseCombo("option+shift+k")
	b, _ := parseCombo("option+shift+j")
	if comboEqual(a, b) {
		t.Error("expected combos with different keys to not match")
	}
}

func TestComboEqual_differentModCount(t *testing.T) {
	a, _ := parseCombo("option+k")
	b, _ := parseCombo("option+shift+k")
	if comboEqual(a, b) {
		t.Error("expected combos with different mod counts to not match")
	}
}

func TestComboZero(t *testing.T) {
	if !comboZero(Combo{}) {
		t.Error("zero Combo should be detected as zero")
	}
	c, _ := parseCombo("space")
	if comboZero(c) {
		t.Error("non-zero Combo should not be detected as zero")
	}
}
