package chunk

import (
	"reflect"
	"testing"
)

func TestSplit_ShortBuffer(t *testing.T) {
	// A buffer shorter than one chunk must be returned as-is (single element).
	samples := make([]float32, 10*sampleRate) // 10 seconds
	got := Split(samples, 25, 5)
	if len(got) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(got))
	}
	if !reflect.DeepEqual(got[0], samples) {
		t.Error("single chunk should be the original slice")
	}
}

func TestSplit_ExactlyOneChunk(t *testing.T) {
	samples := make([]float32, 25*sampleRate) // exactly 25 seconds
	got := Split(samples, 25, 5)
	if len(got) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(got))
	}
}

func TestSplit_TwoChunks(t *testing.T) {
	// 30 seconds with 25s chunks / 5s overlap → step = 20s
	// chunk 0: [0, 25s), chunk 1: [20s, 30s)
	samples := make([]float32, 30*sampleRate)
	got := Split(samples, 25, 5)
	if len(got) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(got))
	}
	if len(got[0]) != 25*sampleRate {
		t.Errorf("chunk 0 len = %d, want %d", len(got[0]), 25*sampleRate)
	}
	if len(got[1]) != 10*sampleRate {
		t.Errorf("chunk 1 len = %d, want %d", len(got[1]), 10*sampleRate)
	}
}

func TestSplit_ThreeChunks(t *testing.T) {
	// 50 seconds: step = 20s → chunks at 0, 20, 40 seconds
	samples := make([]float32, 50*sampleRate)
	got := Split(samples, 25, 5)
	if len(got) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(got))
	}
}

func TestSplit_OverlapSharedSamples(t *testing.T) {
	// Verify that the overlap region is literally the same slice of the original.
	samples := make([]float32, 30*sampleRate)
	for i := range samples {
		samples[i] = float32(i)
	}
	got := Split(samples, 25, 5)
	// chunk 0 ends at index 25*sampleRate-1; chunk 1 starts at index 20*sampleRate
	overlap0 := got[0][20*sampleRate:] // last 5s of chunk 0
	overlap1 := got[1][:5*sampleRate]  // first 5s of chunk 1
	for i := range overlap0 {
		if overlap0[i] != overlap1[i] {
			t.Fatalf("overlap mismatch at index %d: %v vs %v", i, overlap0[i], overlap1[i])
		}
	}
}

// --- Stitch / joinDedup ---

func TestStitch_Empty(t *testing.T) {
	if got := Stitch(nil); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestStitch_Single(t *testing.T) {
	want := "hello world"
	if got := Stitch([]string{want}); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStitch_NoOverlap(t *testing.T) {
	parts := []string{"hello world", "foo bar"}
	got := Stitch(parts)
	want := "hello world foo bar"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStitch_ExactOverlap(t *testing.T) {
	// "foo bar baz" + "bar baz qux" → overlap "bar baz" (2 words)
	parts := []string{"foo bar baz", "bar baz qux"}
	got := Stitch(parts)
	want := "foo bar baz qux"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStitch_CaseInsensitiveOverlap(t *testing.T) {
	// Whisper may capitalise differently at chunk boundaries.
	parts := []string{"Foo Bar Baz", "bar baz qux"}
	got := Stitch(parts)
	want := "Foo Bar Baz qux"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStitch_ThreeParts(t *testing.T) {
	parts := []string{"one two three", "two three four", "three four five"}
	got := Stitch(parts)
	want := "one two three four five"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStitch_EmptyParts(t *testing.T) {
	// Empty parts in the list should not produce double-spaces.
	parts := []string{"hello", "", "world"}
	got := Stitch(parts)
	// "" has no overlap with "hello", so joinDedup("hello","") → "hello"
	// then joinDedup("hello","world") → "hello world"
	want := "hello world"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestJoinDedup_BothEmpty(t *testing.T) {
	if got := joinDedup("", ""); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestJoinDedup_NoMatch(t *testing.T) {
	got := joinDedup("alpha beta", "gamma delta")
	want := "alpha beta gamma delta"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
