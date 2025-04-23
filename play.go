package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

const (
	defaultSampleRate  = 44100
	defaultNumChannels = 2
	defaultAudioFormat = oto.FormatSignedInt16LE // Standard CD quality format
	defaultBufferSize  = 8192                    // Default buffer size in bytes
)

// StreamPlayer handles streaming and playback of MP3 audio from a URL.
type StreamPlayer struct {
	stream  io.ReadCloser // The audio stream (HTTP response body)
	decoder *mp3.Decoder  // MP3 Decoder
	player  *oto.Player   // Audio Player
	context *oto.Context  // Audio Context

	mu      sync.Mutex // Protects concurrent access to shared fields
	playing bool       // Flag indicating if playback is active (Play called, Stop not called)
	paused  bool       // Flag indicating if playback is paused
	url     string     // Store the URL for potential reuse (e.g., restarting after stop)

	// Store config values needed to recreate the player correctly
	sampleRate  int
	numChannels int
	format      oto.Format
}

// Config holds configuration for the audio context.
type Config struct {
	SampleRate  int        // Sample rate of the audio. Defaults to 44100 if zero.
	NumChannels int        // Number of channels. Defaults to 2 if zero.
	Format      oto.Format // Audio format (e.g., oto.FormatSignedInt16LE). Defaults if zero.
	BufferSize  int        // Buffer size in bytes. Defaults if zero.
}

// NewStreamPlayer creates a new YAMusic instance and initializes the audio context.
// It does *not* immediately open the URL stream; call OpenURL or Play separately.
func NewStreamPlayer(cfg *Config) (*StreamPlayer, error) {
	if cfg == nil {
		cfg = &Config{} // Use default config values
	}

	sampleRate := cfg.SampleRate
	if sampleRate == 0 {
		sampleRate = defaultSampleRate
	}

	numChannels := cfg.NumChannels
	if numChannels == 0 {
		numChannels = defaultNumChannels
	}

	format := cfg.Format
	if format == 0 {
		format = defaultAudioFormat
	}

	bufferSize := cfg.BufferSize
	if bufferSize == 0 {
		bufferSize = defaultBufferSize
	}

	// Initialize the audio context
	// The readyChan indicates when the context is ready, but for synchronous init,
	// we can usually proceed directly after NewContext returns without error.
	octx, readyChan, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: numChannels,
		Format:       format,
		// BufferSize field is not present in oto/v3 NewContextOptions
		// The buffer size is managed internally by oto based on its driver.
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create oto context: %w", err)
	}
	// Wait for the context to be ready (optional, but good practice)
	<-readyChan

	ym := &StreamPlayer{
		context:     octx,
		sampleRate:  sampleRate, // Store for potential future use if needed
		numChannels: numChannels,
		format:      format,
	}

	return ym, nil
}

// OpenURL fetches the audio stream from the URL and initializes the MP3 decoder.
// It closes any existing stream before opening the new one.
func (ym *StreamPlayer) OpenURL(url string) error {
	ym.mu.Lock()
	defer ym.mu.Unlock()

	// Close existing stream first to prevent resource leaks
	if err := ym.closeStreamInternal(); err != nil {
		// Log or handle potential error during close, but proceed to open new URL
		fmt.Printf("Warning: error closing previous stream: %v\n", err)
	}

	// Store the url *before* potentially failing the HTTP GET
	ym.url = url
	ym.playing = false // Reset state when opening a new URL
	ym.paused = false

	resp, err := http.Get(url)
	if err != nil {
		ym.url = "" // Clear URL if GET failed
		return fmt.Errorf("http get error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close() // Important: Close the body even on non-200 status
		ym.url = ""       // Clear URL on non-OK status
		return &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status}
	}

	ym.stream = resp.Body // Assign the response body to the stream

	// Create the MP3 decoder
	decoder, err := mp3.NewDecoder(ym.stream)
	if err != nil {
		ym.stream.Close() // Close the stream if decoder creation fails
		ym.stream = nil
		ym.url = ""
		return fmt.Errorf("failed to create mp3 decoder: %w", err)
	}

	// Validate sample rate and channel count if possible (optional)
	// Note: go-mp3 doesn't expose channel count directly in the Decoder struct.
	// We are relying on the config passed to NewYAMusicFromURL.
	// if decoder.SampleRate() != ym.sampleRate {
	//     // Handle mismatch if necessary - oto context was created with ym.sampleRate
	// }

	ym.decoder = decoder

	return nil
}

// Play starts or resumes playing the audio stream.
// If stopped or never started, it creates a new player.
// If paused, it resumes the existing player.
// It requires OpenURL to have been successfully called first.
func (ym *StreamPlayer) Play() error {
	ym.mu.Lock()
	defer ym.mu.Unlock()

	if ym.decoder == nil {
		return &PlayerError{"No decoder initialized. Call OpenURL first."}
	}
	if ym.context == nil {
		return &PlayerError{"Audio context not initialized."}
	}

	// If already playing and not paused, do nothing
	if ym.playing && !ym.paused {
		// Optionally return an error or just ignore
		// return &PlayerError{"Already playing"}
		return nil
	}

	// If paused, just resume
	if ym.paused {
		if ym.player != nil {
			ym.player.Play() // Oto uses Play() to resume
			ym.paused = false
			// ym.playing remains true
			return nil
		}
		// If paused but player is somehow nil, fall through to create a new player
		ym.paused = false // Reset paused state anyway
	}

	// If stopped (player is nil) or never started, create and start a new player
	if ym.player == nil {
		ym.player = ym.context.NewPlayer(ym.decoder)
		// Player volume can be set here if needed: ym.player.SetVolume(1.0)
	}

	ym.player.Play()
	ym.playing = true
	ym.paused = false // Ensure paused is false when starting fresh

	// Note: Play starts playback asynchronously. The stream will be read
	// by the player in a separate goroutine. This function returns immediately.
	// You might want a way to detect when playback finishes (e.g., io.EOF).

	return nil
}

// Pause pauses the audio playback.
func (ym *StreamPlayer) Pause() {
	ym.mu.Lock()
	defer ym.mu.Unlock()

	// Can only pause if currently playing and not already paused
	if !ym.playing || ym.paused {
		return
	}

	if ym.player != nil {
		ym.player.Pause()
	}
	ym.paused = true
}

// Resume resumes paused audio playback.
func (ym *StreamPlayer) Resume() {
	ym.mu.Lock()
	defer ym.mu.Unlock()

	// Can only resume if currently playing and paused
	if !ym.playing || !ym.paused {
		return
	}

	if ym.player != nil {
		ym.player.Play() // Oto uses Play() to resume
	}
	ym.paused = false
}

// Stop stops the audio playback, closes the player, and closes the stream.
// Playback position is lost. Call OpenURL and Play again to restart.
func (ym *StreamPlayer) Stop() error {
	ym.mu.Lock()
	defer ym.mu.Unlock()

	// Close the player first
	var playerErr error
	if ym.player != nil {
		playerErr = ym.player.Close() // Close releases audio resources
		ym.player = nil
	}

	// Close the stream and decoder
	streamErr := ym.closeStreamInternal()

	// Reset state flags
	ym.playing = false
	ym.paused = false
	// Keep ym.url so the user knows what *was* playing

	// Return combined errors if any occurred
	if playerErr != nil || streamErr != nil {
		return fmt.Errorf("error during stop: player_err=%v, stream_err=%v", playerErr, streamErr)
	}
	return nil
}

// PlayAnotherURL stops any current playback, opens the new URL, and starts playing it.
func (ym *StreamPlayer) PlayAnotherURL(url string) error {
	// Stop existing playback cleanly. Lock is acquired within Stop.
	if err := ym.Stop(); err != nil {
		// Log or return the error from stopping the previous track
		fmt.Printf("Warning: error stopping previous track: %v\n", err)
	}

	// Open the new URL. Lock is acquired within OpenURL.
	if err := ym.OpenURL(url); err != nil {
		return fmt.Errorf("failed to open new URL: %w", err)
	}

	// Start playback. Lock is acquired within Play.
	if err := ym.Play(); err != nil {
		return fmt.Errorf("failed to play new URL: %w", err)
	}

	return nil
}

// IsPlaying returns true if the audio is intended to be playing (Play called, Stop not called).
// Note: This reflects the *intended* state. The actual player might stop if the stream ends (EOF).
// You could check `ym.player != nil && ym.player.IsPlaying()` for the hardware state.
func (ym *StreamPlayer) IsPlaying() bool {
	ym.mu.Lock()
	defer ym.mu.Unlock()
	return ym.playing
}

// IsPaused returns true if the audio is currently paused.
func (ym *StreamPlayer) IsPaused() bool {
	ym.mu.Lock()
	defer ym.mu.Unlock()
	return ym.paused
}

// Close stops playback and releases all resources (player, stream).
// The YAMusic object should not be used after calling Close.
func (ym *StreamPlayer) Close() error {
	// Stop already handles closing the player and stream and acquires the lock.
	// The context doesn't need explicit closing in oto/v3.
	err := ym.Stop()
	// Optionally, nullify the context reference if desired, though GC handles it.
	// ym.context = nil
	return err
}

// closeStreamInternal closes the network stream and decoder. Internal use, assumes lock is held or not needed.
// Returns error from stream closing.
func (ym *StreamPlayer) closeStreamInternal() error {
	var err error
	if ym.stream != nil {
		// Decoder wraps the stream, closing the stream should be sufficient.
		err = ym.stream.Close()
		ym.stream = nil
	}
	ym.decoder = nil // Clear decoder reference
	return err
}

// Length returns the total length of the decoded audio stream.
// This requires the decoder to be initialized (OpenURL succeeded).
// Note: Length calculation might require decoding the entire stream for VBR MP3s,
// which might not be efficient for streaming. The go-mp3 library calculation is usually efficient.
func (ym *StreamPlayer) Length() time.Duration {
	ym.mu.Lock()
	defer ym.mu.Unlock()

	if ym.decoder == nil {
		return 0
	}
	bytesPerSample := 2
	switch ym.format {
	case oto.FormatSignedInt16LE:
		bytesPerSample = 2 // 16-bit signed int (2 bytes)
	case oto.FormatFloat32LE:
		bytesPerSample = 4 // 32-bit float (4 bytes)
	case oto.FormatUnsignedInt8:
		bytesPerSample = 1 // 8-bit unsigned int (1 byte)
	default:
		// Handle other formats if needed, or default to 2 bytes
		bytesPerSample = 2 // Fallback to 16-bit signed int
	}
	samples := ym.decoder.Length() / int64(bytesPerSample) / int64(ym.numChannels)
	return time.Duration(samples) * time.Second / time.Duration(ym.decoder.SampleRate())
}

// PlayerError represents an error related to player state.
type PlayerError struct {
	s string
}

func (e *PlayerError) Error() string {
	return e.s
}

// HTTPError represents an error during the HTTP request.
type HTTPError struct {
	StatusCode int
	Status     string
}

func (e *HTTPError) Error() string {
	return "HTTP Error: " + e.Status
}
