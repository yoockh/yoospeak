package llm

import (
	"context"

	vertexgenai "cloud.google.com/go/vertexai/genai"
	"google.golang.org/api/iterator"
)

type VertexGemini struct {
	client *vertexgenai.Client
	model  *vertexgenai.GenerativeModel
}

func NewVertexGemini(ctx context.Context, projectID, location, modelName string) (*VertexGemini, error) {
	c, err := vertexgenai.NewClient(ctx, projectID, location)
	if err != nil {
		return nil, err
	}

	if modelName == "" {
		modelName = "gemini-1.5-flash"
	}

	m := c.GenerativeModel(modelName)
	return &VertexGemini{client: c, model: m}, nil
}

func (v *VertexGemini) Close() error { return v.client.Close() }

func (v *VertexGemini) StreamAnswer(ctx context.Context, prompt string) (<-chan string, <-chan error) {
	out := make(chan string, 32)
	errs := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errs)

		it := v.model.GenerateContentStream(ctx, vertexgenai.Text(prompt))
		for {
			resp, err := it.Next()
			if err == iterator.Done {
				return
			}
			if err != nil {
				errs <- err
				return
			}

			for _, cand := range resp.Candidates {
				if cand.Content == nil {
					continue
				}
				for _, part := range cand.Content.Parts {
					if t, ok := part.(vertexgenai.Text); ok && string(t) != "" {
						out <- string(t)
					}
				}
			}
		}
	}()

	return out, errs
}
