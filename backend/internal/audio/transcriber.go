// Package audio provides speech-to-text transcription and the audio format
// conversion it depends on. The Transcriber interface is provider-neutral:
// Yandex SpeechKit v3 is the v1 implementation, with a GigaChat fallback stub
// kept behind the same seam (ROADMAP 4.1, ARCHITECTURE §11.5). The package is
// self-contained; wiring it into the chat flow happens separately (4.2).
package audio

import (
	"context"
	"errors"
)

// MaxDurationMs caps a single voice message (SPEC.md F5: <= 60s). SpeechKit
// streaming recognition itself allows far longer sessions; this is the product
// limit, enforced by the Converter after it learns the source duration.
const MaxDurationMs int64 = 60_000

var (
	// ErrNotImplemented is returned by provider stubs that exist only to hold
	// the fallback seam open (GigaChat).
	ErrNotImplemented = errors.New("audio: transcriber not implemented")
	// ErrTooLong is returned by a Converter when the source exceeds MaxDurationMs.
	ErrTooLong = errors.New("audio: recording exceeds max duration")
	// ErrEmptyResult is returned by a Transcriber when no speech was recognized.
	ErrEmptyResult = errors.New("audio: no speech recognized")
)

// Result is the outcome of a transcription.
type Result struct {
	Text       string
	DurationMs int64
}

// Transcriber recognizes speech from already-converted OggOpus (16 kHz mono)
// audio. Loading the source object and converting it are the caller's job (see
// Converter); keeping those concerns out of the transcriber makes it provider-
// agnostic and trivially testable. lang is a BCP-47 code (e.g. "ru-RU"); ""
// lets the provider use its default.
type Transcriber interface {
	Transcribe(ctx context.Context, oggOpus []byte, lang string) (Result, error)
}

// Converter transcodes recorded audio (m4a/aac/mp3) into the OggOpus 16 kHz
// mono format SpeechKit expects and reports the source duration in ms.
type Converter interface {
	ToOggOpus(ctx context.Context, input []byte, contentType string) (oggOpus []byte, durationMs int64, err error)
}
