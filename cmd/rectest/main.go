// rectest records 5 seconds of audio and saves it as /tmp/rectest.wav.
// Run with: make rectest
// Optionally pass a device name substring: make rectest DEV="MacBook"
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/gen2brain/malgo"
)

func main() {
	deviceFilter := ""
	if len(os.Args) > 1 {
		deviceFilter = strings.ToLower(os.Args[1])
	}

	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		fmt.Printf("FAIL init context: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = ctx.Uninit(); ctx.Free() }()

	// List devices.
	devices, err := ctx.Devices(malgo.Capture)
	if err != nil {
		fmt.Printf("FAIL list devices: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Available capture devices (%d):\n", len(devices))
	for i, d := range devices {
		fmt.Printf("  [%d] %s\n", i, d.Name())
	}

	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatF32
	cfg.Capture.Channels = 1
	cfg.SampleRate = 16000

	// If a filter was provided, pick the first matching device.
	if deviceFilter != "" {
		for i := range devices {
			if strings.Contains(strings.ToLower(devices[i].Name()), deviceFilter) {
				id := devices[i].ID
				cfg.Capture.DeviceID = id.Pointer()
				fmt.Printf("Using device: %s\n", devices[i].Name())
				break
			}
		}
	} else {
		fmt.Println("Using system default device.")
	}

	var buf []float32
	dev, err := malgo.InitDevice(ctx.Context, cfg, malgo.DeviceCallbacks{
		Data: func(_, in []byte, frames uint32) {
			count := int(frames)
			for i := 0; i < count; i++ {
				off := i * 4
				bits := uint32(in[off]) | uint32(in[off+1])<<8 |
					uint32(in[off+2])<<16 | uint32(in[off+3])<<24
				buf = append(buf, math.Float32frombits(bits))
			}
		},
	})
	if err != nil {
		fmt.Printf("FAIL init device: %v\n", err)
		os.Exit(1)
	}
	defer dev.Uninit()

	fmt.Println("Recording 5 seconds — speak now...")
	if err := dev.Start(); err != nil {
		fmt.Printf("FAIL start device: %v\n", err)
		os.Exit(1)
	}
	time.Sleep(5 * time.Second)
	dev.Stop()

	fmt.Printf("Captured %d samples\n", len(buf))

	if len(buf) == 0 {
		fmt.Println("RESULT: 0 samples — device started but no data received.")
		os.Exit(1)
	}

	// Calculate RMS.
	var sum float64
	for _, s := range buf {
		sum += float64(s) * float64(s)
	}
	rms := math.Sqrt(sum / float64(len(buf)))
	fmt.Printf("RMS level: %.6f\n", rms)
	if rms < 0.0001 {
		fmt.Println("RESULT: near-silence — mic may be muted or wrong device.")
	} else {
		fmt.Printf("RESULT: audio captured (RMS=%.4f) — mic is working.\n", rms)
	}

	// Write WAV file so you can play it back.
	outPath := "/tmp/rectest.wav"
	if err := writeWAV(outPath, buf, 16000); err != nil {
		fmt.Printf("WARN: could not write WAV: %v\n", err)
	} else {
		fmt.Printf("Saved to %s — play with: afplay %s\n", outPath, outPath)
	}
}

func writeWAV(path string, samples []float32, sampleRate uint32) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	numSamples := uint32(len(samples))
	dataSize := numSamples * 2 // 16-bit PCM

	write := func(v any) { binary.Write(f, binary.LittleEndian, v) }

	// RIFF header
	f.WriteString("RIFF")
	write(uint32(36 + dataSize))
	f.WriteString("WAVE")
	// fmt chunk
	f.WriteString("fmt ")
	write(uint32(16))      // chunk size
	write(uint16(1))       // PCM
	write(uint16(1))       // mono
	write(sampleRate)      // sample rate
	write(sampleRate * 2)  // byte rate
	write(uint16(2))       // block align
	write(uint16(16))      // bits per sample
	// data chunk
	f.WriteString("data")
	write(dataSize)
	for _, s := range samples {
		v := s
		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		write(int16(v * 32767))
	}
	return nil
}
