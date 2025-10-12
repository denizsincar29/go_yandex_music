package main

/*
// This is a wrapper for mp3 decoder that signals a channel message on eof.

import (
	"io"

	"github.com/hajimehoshi/go-mp3"
)

// DecoderWrapper wraps an io.Reader and signals a channel message on EOF.
type DecoderWrapper struct {
	reader  io.Reader
	decoder *mp3.Decoder
	eofChan chan struct{}
}

// NewDecoderWrapper creates a new DecoderWrapper.
func NewDecoderWrapper(r io.Reader) *DecoderWrapper {
	decoder := mp3.NewDecoder(r)
	return &DecoderWrapper{
		reader:  r,
		decoder: decoder,
		eofChan: make(chan struct{}),
	}
}
*/
