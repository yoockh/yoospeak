package stt

import (
	"context"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "cloud.google.com/go/speech/apiv1/speechpb"
)

type GoogleSpeech struct {
	c *speech.Client

	Encoding     speechpb.RecognitionConfig_AudioEncoding
	SampleRateHz int32
}

func NewGoogleSpeech(ctx context.Context) (*GoogleSpeech, error) {
	c, err := speech.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GoogleSpeech{
		c:            c,
		Encoding:     speechpb.RecognitionConfig_LINEAR16,
		SampleRateHz: 16000,
	}, nil
}

func (g *GoogleSpeech) Close() error { return g.c.Close() }

// language example: "en-US", "id-ID"
func (g *GoogleSpeech) Transcribe(ctx context.Context, audio []byte, language string) (string, float64, error) {
	if language == "" {
		language = "en-US"
	}

	resp, err := g.c.Recognize(ctx, &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:                   g.Encoding,
			SampleRateHertz:            g.SampleRateHz,
			LanguageCode:               language,
			EnableAutomaticPunctuation: true,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: audio},
		},
	})
	if err != nil {
		return "", 0, err
	}

	var bestText string
	var bestConf float64
	for _, r := range resp.Results {
		for _, alt := range r.Alternatives {
			if alt.Transcript != "" && float64(alt.Confidence) >= bestConf {
				bestText = alt.Transcript
				bestConf = float64(alt.Confidence)
			}
		}
	}

	return bestText, bestConf, nil
}
