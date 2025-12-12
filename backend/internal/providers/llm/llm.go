package llm

import "context"

type Provider interface {
	// StreamAnswer returns a stream of text chunks (incremental).
	StreamAnswer(ctx context.Context, prompt string) (chunks <-chan string, errs <-chan error)
	Close() error
}
