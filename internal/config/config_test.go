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

// --- applyDefaults: sound_enabled / notifications_enabled ---

func TestApplyDefaults_soundAndNotificationsDefaultTrue(t *testing.T) {
	c := raw{} // nil *bool fields
	applyDefaults(&c)
	if c.SoundEnabled == nil || !*c.SoundEnabled {
		t.Error("expected SoundEnabled to default to true")
	}
	if c.NotificationsEnabled == nil || !*c.NotificationsEnabled {
		t.Error("expected NotificationsEnabled to default to true")
	}
}

func TestApplyDefaults_soundExplicitFalsePreserved(t *testing.T) {
	f := false
	c := raw{SoundEnabled: &f}
	applyDefaults(&c)
	if c.SoundEnabled == nil || *c.SoundEnabled {
		t.Error("expected explicit false to be preserved for SoundEnabled")
	}
}

func TestApplyDefaults_notificationsExplicitFalsePreserved(t *testing.T) {
	f := false
	c := raw{NotificationsEnabled: &f}
	applyDefaults(&c)
	if c.NotificationsEnabled == nil || *c.NotificationsEnabled {
		t.Error("expected explicit false to be preserved for NotificationsEnabled")
	}
}

func TestApplyDefaults_soundExplicitTruePreserved(t *testing.T) {
	v := true
	c := raw{SoundEnabled: &v}
	applyDefaults(&c)
	if c.SoundEnabled == nil || !*c.SoundEnabled {
		t.Error("expected explicit true to be preserved for SoundEnabled")
	}
}

// --- parseModes: vocabulary ---

func TestParseModes_propagatesVocabulary(t *testing.T) {
	vocab := []string{"Kubernetes", "gRPC"}
	raw := []modeRaw{{Name: "Dev", Language: "en", Vocabulary: vocab}}
	modes := parseModes(raw)
	if len(modes[0].Vocabulary) != 2 {
		t.Fatalf("expected 2 vocabulary entries, got %d", len(modes[0].Vocabulary))
	}
	if modes[0].Vocabulary[0] != "Kubernetes" || modes[0].Vocabulary[1] != "gRPC" {
		t.Errorf("unexpected vocabulary: %v", modes[0].Vocabulary)
	}
}

func TestParseModes_nilVocabularyPassedThrough(t *testing.T) {
	raw := []modeRaw{{Name: "Standard"}}
	modes := parseModes(raw)
	if modes[0].Vocabulary != nil {
		t.Errorf("expected nil vocabulary, got %v", modes[0].Vocabulary)
	}
}

// --- modesEqual: vocabulary ---

func TestModesEqual_sameVocabulary(t *testing.T) {
	a := []modeRaw{{Name: "A", Vocabulary: []string{"foo", "bar"}}}
	b := []modeRaw{{Name: "A", Vocabulary: []string{"foo", "bar"}}}
	if !modesEqual(a, b) {
		t.Error("expected modes with identical vocabulary to be equal")
	}
}

func TestModesEqual_differentVocabulary(t *testing.T) {
	a := []modeRaw{{Name: "A", Vocabulary: []string{"foo"}}}
	b := []modeRaw{{Name: "A", Vocabulary: []string{"bar"}}}
	if modesEqual(a, b) {
		t.Error("expected modes with different vocabulary to not be equal")
	}
}

func TestModesEqual_oneVocabularyNil(t *testing.T) {
	a := []modeRaw{{Name: "A", Vocabulary: []string{"foo"}}}
	b := []modeRaw{{Name: "A"}}
	if modesEqual(a, b) {
		t.Error("expected modes where one has vocabulary and other does not to not be equal")
	}
}

// --- stringSliceEqual ---

func TestStringSliceEqual_equal(t *testing.T) {
	if !stringSliceEqual([]string{"a", "b"}, []string{"a", "b"}) {
		t.Error("expected equal slices to match")
	}
}

func TestStringSliceEqual_bothNil(t *testing.T) {
	if !stringSliceEqual(nil, nil) {
		t.Error("expected nil == nil")
	}
}

func TestStringSliceEqual_differentLength(t *testing.T) {
	if stringSliceEqual([]string{"a"}, []string{"a", "b"}) {
		t.Error("expected different-length slices to not match")
	}
}

func TestStringSliceEqual_differentContent(t *testing.T) {
	if stringSliceEqual([]string{"a", "b"}, []string{"a", "c"}) {
		t.Error("expected slices with different content to not match")
	}
}

func TestStringSliceEqual_nilAndEmpty(t *testing.T) {
	if !stringSliceEqual(nil, []string{}) {
		t.Error("expected nil and empty slice to be equal")
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
