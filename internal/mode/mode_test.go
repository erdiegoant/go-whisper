package mode

import "testing"

func TestNewManager_defaults(t *testing.T) {
	m := NewManager(nil)
	if m.Current().Name != "Standard" {
		t.Errorf("expected Standard, got %s", m.Current().Name)
	}
}

func TestNewManager_custom(t *testing.T) {
	modes := []Mode{
		{Name: "A", Language: "auto"},
		{Name: "B", Language: "es", Translate: true},
	}
	m := NewManager(modes)
	if m.Current().Name != "A" {
		t.Errorf("expected A, got %s", m.Current().Name)
	}
}

func TestNext_cycles(t *testing.T) {
	modes := []Mode{{Name: "A"}, {Name: "B"}, {Name: "C"}}
	m := NewManager(modes)
	if got := m.Next().Name; got != "B" {
		t.Errorf("expected B, got %s", got)
	}
	if got := m.Next().Name; got != "C" {
		t.Errorf("expected C, got %s", got)
	}
	if got := m.Next().Name; got != "A" {
		t.Errorf("expected wrap to A, got %s", got)
	}
}

func TestSetByName_found(t *testing.T) {
	modes := []Mode{{Name: "A"}, {Name: "B"}, {Name: "C"}}
	m := NewManager(modes)
	if !m.SetByName("C") {
		t.Fatal("SetByName should return true for known name")
	}
	if m.Current().Name != "C" {
		t.Errorf("expected C, got %s", m.Current().Name)
	}
}

func TestSetByName_notFound(t *testing.T) {
	modes := []Mode{{Name: "A"}, {Name: "B"}}
	m := NewManager(modes)
	if m.SetByName("Z") {
		t.Fatal("SetByName should return false for unknown name")
	}
	if m.Current().Name != "A" {
		t.Errorf("active mode should be unchanged, got %s", m.Current().Name)
	}
}

func TestReload_preservesActiveName(t *testing.T) {
	m := NewManager([]Mode{{Name: "A"}, {Name: "B"}})
	m.SetByName("B")
	m.Reload([]Mode{{Name: "X"}, {Name: "B"}, {Name: "Y"}})
	if m.Current().Name != "B" {
		t.Errorf("expected B to stay active after reload, got %s", m.Current().Name)
	}
}

func TestReload_fallbackWhenMissing(t *testing.T) {
	m := NewManager([]Mode{{Name: "A"}, {Name: "B"}})
	m.SetByName("B")
	m.Reload([]Mode{{Name: "X"}, {Name: "Y"}})
	if m.Current().Name != "X" {
		t.Errorf("expected fallback to first mode X, got %s", m.Current().Name)
	}
}

func TestReload_emptyFallsBackToDefaults(t *testing.T) {
	m := NewManager([]Mode{{Name: "A"}})
	m.Reload(nil)
	if m.Current().Name != DefaultModes[0].Name {
		t.Errorf("expected default mode %s, got %s", DefaultModes[0].Name, m.Current().Name)
	}
}

func TestAll_returnsCurrentList(t *testing.T) {
	modes := []Mode{{Name: "A"}, {Name: "B"}}
	m := NewManager(modes)
	all := m.All()
	if len(all) != 2 {
		t.Errorf("expected 2 modes, got %d", len(all))
	}
}

func TestPromptField(t *testing.T) {
	modes := []Mode{
		{Name: "Custom", Language: "auto", Prompt: "Do something special."},
	}
	m := NewManager(modes)
	if m.Current().Prompt != "Do something special." {
		t.Errorf("unexpected prompt: %q", m.Current().Prompt)
	}
}
