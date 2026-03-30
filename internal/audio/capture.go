package audio

import (
	"fmt"
	"math"
	"sync"

	"github.com/gen2brain/malgo"
)

const (
	sampleRate = uint32(16000)
	channels   = uint32(1)
	sampleSize = 4 // float32 = 4 bytes
)

// State represents the recording state machine.
type State int

const (
	StateIdle       State = iota
	StateRecording
	StateProcessing
)

// Capturer manages microphone capture and the recording state machine.
type Capturer struct {
	mu     sync.Mutex
	state  State
	buf    []float32

	ctx    *malgo.AllocatedContext
	device *malgo.Device
}

// New initializes a Capturer and the underlying miniaudio context.
func New() (*Capturer, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("audio: init context: %w", err)
	}
	return &Capturer{ctx: ctx}, nil
}

// CurrentState returns the current recording state.
func (c *Capturer) CurrentState() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// Start begins mic capture. Transitions IDLE → RECORDING.
// Returns an error if not currently IDLE.
func (c *Capturer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state != StateIdle {
		return fmt.Errorf("audio: cannot start, current state: %d", c.state)
	}

	c.buf = c.buf[:0] // reset buffer, reuse backing array

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = channels
	deviceConfig.SampleRate = sampleRate

	callbacks := malgo.DeviceCallbacks{
		Data: c.onData,
	}

	device, err := malgo.InitDevice(c.ctx.Context, deviceConfig, callbacks)
	if err != nil {
		return fmt.Errorf("audio: init device: %w", err)
	}

	if err := device.Start(); err != nil {
		device.Uninit()
		return fmt.Errorf("audio: start device: %w", err)
	}

	c.device = device
	c.state = StateRecording
	return nil
}

// Stop ends capture and transitions RECORDING → PROCESSING.
// Returns a copy of the captured sample buffer.
// Call SetIdle() after transcription completes.
func (c *Capturer) Stop() ([]float32, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state != StateRecording {
		return nil, fmt.Errorf("audio: cannot stop, current state: %d", c.state)
	}

	c.stopDevice()
	c.state = StateProcessing

	out := make([]float32, len(c.buf))
	copy(out, c.buf)
	return out, nil
}

// Cancel discards the active recording and returns to IDLE.
// Safe to call in any state — no-op if not recording.
func (c *Capturer) Cancel() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state != StateRecording {
		return
	}
	c.stopDevice()
	c.buf = c.buf[:0]
	c.state = StateIdle
}

// SetIdle transitions PROCESSING → IDLE. Call after transcription completes.
func (c *Capturer) SetIdle() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = StateIdle
}

// Close releases all miniaudio resources.
func (c *Capturer) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.device != nil {
		c.stopDevice()
	}
	_ = c.ctx.Uninit()
	c.ctx.Free()
}

// onData is called by miniaudio on the audio thread with incoming PCM bytes.
// It decodes float32 samples from the raw byte slice and appends them to the buffer.
func (c *Capturer) onData(_, pInputSample []byte, frameCount uint32) {
	count := int(frameCount) * int(channels)
	samples := make([]float32, count)
	for i := 0; i < count; i++ {
		offset := i * sampleSize
		bits := uint32(pInputSample[offset]) |
			uint32(pInputSample[offset+1])<<8 |
			uint32(pInputSample[offset+2])<<16 |
			uint32(pInputSample[offset+3])<<24
		samples[i] = math.Float32frombits(bits)
	}

	c.mu.Lock()
	if c.state == StateRecording {
		c.buf = append(c.buf, samples...)
	}
	c.mu.Unlock()
}

// stopDevice uninitializes the device. Caller must hold c.mu.
func (c *Capturer) stopDevice() {
	if c.device == nil {
		return
	}
	c.device.Stop()
	c.device.Uninit()
	c.device = nil
}
