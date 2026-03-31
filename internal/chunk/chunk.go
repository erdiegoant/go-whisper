// Package chunk splits large audio sample buffers into overlapping segments
// and stitches the resulting transcripts back together.
package chunk

import "strings"

const sampleRate = 16000 // Hz — matches Whisper's expected input rate

// Split divides samples into overlapping chunks of chunkSecs seconds,
// with overlapSecs seconds of shared audio at each boundary.
// Returns the original slice as a single-element result if it fits in one chunk.
func Split(samples []float32, chunkSecs, overlapSecs int) [][]float32 {
	chunkSize := chunkSecs * sampleRate
	step := (chunkSecs - overlapSecs) * sampleRate

	if len(samples) <= chunkSize {
		return [][]float32{samples}
	}

	var out [][]float32
	for start := 0; start < len(samples); start += step {
		end := start + chunkSize
		if end > len(samples) {
			end = len(samples)
		}
		out = append(out, samples[start:end])
		if end == len(samples) {
			break
		}
	}
	return out
}

// Stitch joins transcript parts, deduplicating the word-level overlap
// at each boundary between consecutive parts.
func Stitch(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result = joinDedup(result, parts[i])
	}
	return strings.TrimSpace(result)
}

// joinDedup finds the longest word-level suffix of a that case-insensitively
// matches a prefix of b, then concatenates without repeating it.
func joinDedup(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}

	aWords := strings.Fields(a)
	bWords := strings.Fields(b)

	maxOverlap := len(aWords)
	if len(bWords) < maxOverlap {
		maxOverlap = len(bWords)
	}
	if maxOverlap > 20 {
		maxOverlap = 20
	}

	best := 0
	for n := maxOverlap; n >= 1; n-- {
		aSuffix := strings.Join(aWords[len(aWords)-n:], " ")
		bPrefix := strings.Join(bWords[:n], " ")
		if strings.EqualFold(aSuffix, bPrefix) {
			best = n
			break
		}
	}

	if best == 0 {
		return a + " " + b
	}
	result := make([]string, 0, len(aWords)+len(bWords)-best)
	result = append(result, aWords...)
	result = append(result, bWords[best:]...)
	return strings.Join(result, " ")
}
