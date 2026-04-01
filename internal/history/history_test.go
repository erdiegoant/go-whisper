package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func openTemp(t *testing.T) *Log {
	t.Helper()
	dir := t.TempDir()
	l, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { l.Close() })
	return l
}

func TestOpen_createsSchema(t *testing.T) {
	dir := t.TempDir()
	l, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer l.Close()

	if _, err := os.Stat(filepath.Join(dir, "history.db")); err != nil {
		t.Errorf("history.db not created: %v", err)
	}
}

func TestAdd_andRecent(t *testing.T) {
	l := openTemp(t)
	now := time.Now().UTC().Truncate(time.Second)

	e := Entry{
		Timestamp:     now,
		ModeName:      "Standard",
		PromptUsed:    "clean it up",
		RawText:       "um hello world",
		ProcessedText: "Hello world.",
		DurationMs:    1500,
		Language:      "en",
	}
	if err := l.Add(e); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entries, err := l.Recent(10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.ModeName != e.ModeName {
		t.Errorf("ModeName: want %q, got %q", e.ModeName, got.ModeName)
	}
	if got.RawText != e.RawText {
		t.Errorf("RawText: want %q, got %q", e.RawText, got.RawText)
	}
	if got.ProcessedText != e.ProcessedText {
		t.Errorf("ProcessedText: want %q, got %q", e.ProcessedText, got.ProcessedText)
	}
	if got.DurationMs != e.DurationMs {
		t.Errorf("DurationMs: want %d, got %d", e.DurationMs, got.DurationMs)
	}
	if !got.Timestamp.Equal(now) {
		t.Errorf("Timestamp: want %v, got %v", now, got.Timestamp)
	}
}

func TestRecent_newestFirst(t *testing.T) {
	l := openTemp(t)
	base := time.Now().UTC().Truncate(time.Second)

	for i := 0; i < 3; i++ {
		if err := l.Add(Entry{
			Timestamp: base.Add(time.Duration(i) * time.Second),
			RawText:   string(rune('a' + i)),
		}); err != nil {
			t.Fatalf("Add %d: %v", i, err)
		}
	}

	entries, err := l.Recent(10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("want 3 entries, got %d", len(entries))
	}
	// newest first: c, b, a
	if entries[0].RawText != "c" || entries[1].RawText != "b" || entries[2].RawText != "a" {
		t.Errorf("wrong order: %v", []string{entries[0].RawText, entries[1].RawText, entries[2].RawText})
	}
}

func TestRecent_respectsLimit(t *testing.T) {
	l := openTemp(t)

	for i := 0; i < 5; i++ {
		if err := l.Add(Entry{RawText: "x", Timestamp: time.Now().UTC()}); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	entries, err := l.Recent(2)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("want 2 entries, got %d", len(entries))
	}
}

func TestRecent_empty(t *testing.T) {
	l := openTemp(t)
	entries, err := l.Recent(10)
	if err != nil {
		t.Fatalf("Recent on empty db: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("want 0 entries, got %d", len(entries))
	}
}
