package stt

import "context"

type Provider interface {
	Transcribe(ctx context.Context, audio []byte, language string) (text string, confidence float64, err error)
	Close() error
}
