package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const baseURL = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main"

// known SHA256 checksums for the standard GGML model files.
var checksums = map[string]string{
	"ggml-tiny.bin":   "be07e048e1e599ad46341c8d2a135645097a538221678b7acdd1b1919c6e1b21",
	"ggml-small.bin":  "1be3a9b2063867b937e64e2ec7483364a79917e157fa98c5d94b5c1fffea987b",
	"ggml-medium.bin": "fd9727b6807d35349f9d8cedcf90d189816a243432dbcb65ac9b148c10030278",
}

// Download fetches model size (tiny|small|medium) into dir, showing progress.
// Verifies SHA256 checksum after download. Safe to call if the file already exists.
func Download(size, dir string) error {
	size = strings.ToLower(strings.TrimSpace(size))
	filename := "ggml-" + size + ".bin"
	expected, ok := checksums[filename]
	if !ok {
		return fmt.Errorf("models: unknown size %q — valid options: tiny, small, medium", size)
	}

	dest := filepath.Join(dir, filename)

	// Skip download if file already exists and checksum matches.
	if _, err := os.Stat(dest); err == nil {
		fmt.Printf("Verifying existing %s... ", filename)
		if sum, err := sha256File(dest); err == nil && sum == expected {
			fmt.Println("already up to date.")
			return nil
		}
		fmt.Println("checksum mismatch — re-downloading.")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("models: create dir: %w", err)
	}

	url := baseURL + "/" + filename
	fmt.Printf("Downloading %s from Hugging Face...\n", filename)

	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return fmt.Errorf("models: download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("models: server returned %d", resp.StatusCode)
	}

	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("models: create temp file: %w", err)
	}
	defer os.Remove(tmp)

	total := resp.ContentLength
	if err := copyWithProgress(f, resp.Body, total); err != nil {
		f.Close()
		return fmt.Errorf("models: write: %w", err)
	}
	f.Close()

	fmt.Printf("\nVerifying checksum... ")
	sum, err := sha256File(tmp)
	if err != nil {
		return fmt.Errorf("models: checksum: %w", err)
	}
	if sum != expected {
		return fmt.Errorf("models: checksum mismatch (got %s, want %s)", sum, expected)
	}
	fmt.Println("ok.")

	if err := os.Rename(tmp, dest); err != nil {
		return fmt.Errorf("models: finalize: %w", err)
	}
	fmt.Printf("Saved to %s\n", dest)
	return nil
}

func copyWithProgress(dst io.Writer, src io.Reader, total int64) error {
	buf := make([]byte, 32*1024)
	var written int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			written += int64(n)
			if total > 0 {
				pct := float64(written) / float64(total) * 100
				fmt.Printf("\r  %.1f%% (%.0f / %.0f MB)", pct,
					float64(written)/1e6, float64(total)/1e6)
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
