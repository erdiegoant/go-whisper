package config

import (
	"testing"

	"github.com/erdiegoant/gowhisper/internal/mode"
)

// --- parseModes ---

func TestParseModes_empty(t *testing.T) {
	modes := parseModes(nil)
	if len(modes) != len(mode.DefaultModes) {
		t.Fatalf("expected %d default modes, got %d", len(mode.DefaultModes), len(modes))
	}
	if modes[0].Name != mode.DefaultModes[0].Name {
		t.Errorf("expected first default mode %q, got %q", mode.DefaultModes[0].Name, modes[0].Name)
	}
}

func TestParseModes_setsLanguageDefault(t *testing.T) {
	raw := []modeRaw{{Name: "NoLang"}}
	modes := parseModes(raw)
	if modes[0].Language != "auto" {
		t.Errorf("expected language 'auto', got %q", modes[0].Language)
	}
}

func TestParseModes_preservesFields(t *testing.T) {
	raw := []modeRaw{
		{Name: "Formal", Language: "en", Translate: false, Prompt: "Be formal."},
		{Name: "Translate", Language: "es", Translate: true},
	}
	modes := parseModes(raw)
	if len(modes) != 2 {
		t.Fatalf("expected 2 modes, got %d", len(modes))
	}
	if modes[0].Prompt != "Be formal." {
		t.Errorf("expected prompt preserved, got %q", modes[0].Prompt)
	}
	if !modes[1].Translate {
		t.Error("expected Translate=true for second mode")
	}
}

// --- modesEqual ---

func TestModesEqual_equal(t *testing.T) {
	a := []modeRaw{{Name: "A", Language: "auto"}, {Name: "B", Language: "es", Translate: true}}
	b := []modeRaw{{Name: "A", Language: "auto"}, {Name: "B", Language: "es", Translate: true}}
	if !modesEqual(a, b) {
		t.Error("expected equal slices to match")
	}
}

func TestModesEqual_differentLength(t *testing.T) {
	a := []modeRaw{{Name: "A"}}
	b := []modeRaw{{Name: "A"}, {Name: "B"}}
	if modesEqual(a, b) {
		t.Error("expected different-length slices to not match")
	}
}

func TestModesEqual_differentField(t *testing.T) {
	a := []modeRaw{{Name: "A", Language: "auto"}}
	b := []modeRaw{{Name: "A", Language: "es"}}
	if modesEqual(a, b) {
		t.Error("expected slices with different Language to not match")
	}
}

func TestModesEqual_bothNil(t *testing.T) {
	if !modesEqual(nil, nil) {
		t.Error("expected nil == nil")
	}
}

// --- applyDefaults ---

func TestApplyDefaults_fillsEmpty(t *testing.T) {
	c := raw{}
	applyDefaults(&c)
	if c.Model != defaults.Model {
		t.Errorf("expected model %q, got %q", defaults.Model, c.Model)
	}
	if c.Language != defaults.Language {
		t.Errorf("expected language %q, got %q", defaults.Language, c.Language)
	}
	if c.MaxRecordingSeconds != defaults.MaxRecordingSeconds {
		t.Errorf("expected max_recording_seconds %d, got %d", defaults.MaxRecordingSeconds, c.MaxRecordingSeconds)
	}
	if c.Claude.Model != defaults.Claude.Model {
		t.Errorf("expected claude model %q, got %q", defaults.Claude.Model, c.Claude.Model)
	}
	if c.Claude.TimeoutSeconds != defaults.Claude.TimeoutSeconds {
		t.Errorf("expected timeout %d, got %d", defaults.Claude.TimeoutSeconds, c.Claude.TimeoutSeconds)
	}
}

func TestApplyDefaults_doesNotOverrideSet(t *testing.T) {
	c := raw{Model: "medium", Language: "es", MaxRecordingSeconds: 60}
	applyDefaults(&c)
	if c.Model != "medium" {
		t.Errorf("expected model 'medium' to be preserved, got %q", c.Model)
	}
	if c.Language != "es" {
		t.Errorf("expected language 'es' to be preserved, got %q", c.Language)
	}
	if c.MaxRecordingSeconds != 60 {
		t.Errorf("expected 60s to be preserved, got %d", c.MaxRecordingSeconds)
	}
}

// --- combosEqual ---

func TestCombosEqual(t *testing.T) {
	toggle, _ := parseCombo("option+space")
	cancel, _ := parseCombo("esc")
	modeC, _ := parseCombo("option+shift+k")

	a := Combos{Toggle: toggle, Cancel: cancel, Mode: modeC}
	b := Combos{Toggle: toggle, Cancel: cancel, Mode: modeC}
	if !combosEqual(a, b) {
		t.Error("expected equal Combos to match")
	}
}

func TestCombosEqual_different(t *testing.T) {
	toggle, _ := parseCombo("option+space")
	cancel, _ := parseCombo("esc")
	modeA, _ := parseCombo("option+shift+k")
	modeB, _ := parseCombo("option+shift+j")

	a := Combos{Toggle: toggle, Cancel: cancel, Mode: modeA}
	b := Combos{Toggle: toggle, Cancel: cancel, Mode: modeB}
	if combosEqual(a, b) {
		t.Error("expected Combos with different Mode to not match")
	}
}
