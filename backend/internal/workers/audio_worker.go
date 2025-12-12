package workers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/yoockh/yoospeak/internal/providers/llm"
	"github.com/yoockh/yoospeak/internal/providers/stt"
	"github.com/yoockh/yoospeak/internal/services"
)

type AudioWorkerPool struct {
	Redis      *redis.Client
	Buffers    services.BufferService
	NumWorkers int

	STT stt.Provider
	LLM llm.Provider

	Logger *logrus.Logger

	Stream         string
	Group          string
	ConsumerPrefix string
}

func (p *AudioWorkerPool) Start(ctx context.Context) error {
	if p.Redis == nil || p.Buffers == nil || p.STT == nil || p.LLM == nil {
		return errors.New("AudioWorkerPool missing dependency: Redis/Buffers/STT/LLM must be set")
	}
	if p.Stream == "" {
		p.Stream = "audio:stream"
	}
	if p.Group == "" {
		p.Group = "audio-workers"
	}
	if p.ConsumerPrefix == "" {
		p.ConsumerPrefix = "c"
	}
	if p.NumWorkers <= 0 {
		p.NumWorkers = 5
	}
	if p.Logger == nil {
		p.Logger = logrus.New()
	}

	_ = p.Redis.XGroupCreateMkStream(ctx, p.Stream, p.Group, "0").Err() // ignore BUSYGROUP

	for i := 0; i < p.NumWorkers; i++ {
		consumer := p.ConsumerPrefix + "-" + strconv.Itoa(i+1)
		go p.runConsumer(ctx, consumer)
	}
	return nil
}

func (p *AudioWorkerPool) runConsumer(ctx context.Context, consumer string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		res, err := p.Redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    p.Group,
			Consumer: consumer,
			Streams:  []string{p.Stream, ">"},
			Count:    10,
			Block:    5 * time.Second,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, stream := range res {
			for _, msg := range stream.Messages {
				p.handleMsg(ctx, msg)
				_ = p.Redis.XAck(ctx, p.Stream, p.Group, msg.ID).Err()
			}
		}
	}
}

func normalizeLanguage(v string) string {
	v = strings.TrimSpace(v)
	switch v {
	case "id", "id-ID":
		return "id-ID"
	case "en", "en-US":
		return "en-US"
	default:
		if v == "" {
			return "en-US"
		}
		return v
	}
}

func (p *AudioWorkerPool) handleMsg(ctx context.Context, msg redis.XMessage) {
	getStr := func(k string) string {
		v, ok := msg.Values[k]
		if !ok || v == nil {
			return ""
		}
		s, _ := v.(string)
		return s
	}

	sessionID := getStr("session_id")
	chunkIndexStr := getStr("chunk_index")
	if sessionID == "" || chunkIndexStr == "" {
		return
	}
	chunkIndex, _ := strconv.ParseInt(chunkIndexStr, 10, 64)

	log := p.Logger.WithFields(logrus.Fields{
		"redis_id":    msg.ID,
		"session_id":  sessionID,
		"chunk_index": chunkIndex,
	})

	respCh := "session:" + sessionID + ":response"
	statusCh := "session:" + sessionID + ":status"

	language := normalizeLanguage(getStr("language"))

	// Fetch audio
	var audioBytes []byte
	if b64 := getStr("audio_base64"); b64 != "" {
		raw := b64
		if i := strings.Index(raw, ","); i >= 0 {
			raw = raw[i+1:] // strip data:...;base64,
		}
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			log.WithError(err).Warn("base64 decode failed")
			_ = p.Redis.Publish(ctx, statusCh, `{"type":"status","status":"failed","message":"invalid audio_base64","chunk_index":`+strconv.FormatInt(chunkIndex, 10)+`}`).Err()
			return
		}
		audioBytes = decoded
	} else if url := getStr("audio_url"); url != "" {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.WithError(err).Warn("audio_url fetch failed")
			_ = p.Redis.Publish(ctx, statusCh, `{"type":"status","status":"failed","message":"failed to fetch audio_url","chunk_index":`+strconv.FormatInt(chunkIndex, 10)+`}`).Err()
			return
		}
		defer resp.Body.Close()

		const maxBytes = 10 << 20
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
		if len(body) == 0 {
			_ = p.Redis.Publish(ctx, statusCh, `{"type":"status","status":"failed","message":"empty audio","chunk_index":`+strconv.FormatInt(chunkIndex, 10)+`}`).Err()
			return
		}
		audioBytes = body
	} else {
		return
	}

	// STT
	_ = p.Buffers.MarkSTT(ctx, sessionID, chunkIndex, "", 0, "processing")
	_ = p.Redis.Publish(ctx, statusCh, `{"type":"status","status":"processing","message":"stt processing","chunk_index":`+strconv.FormatInt(chunkIndex, 10)+`}`).Err()

	text, conf, err := p.STT.Transcribe(ctx, audioBytes, language)
	if err != nil {
		log.WithError(err).Error("stt failed")
		_ = p.Buffers.MarkSTT(ctx, sessionID, chunkIndex, "", 0, "failed")
		_ = p.Redis.Publish(ctx, statusCh, `{"type":"status","status":"failed","message":"stt failed","chunk_index":`+strconv.FormatInt(chunkIndex, 10)+`}`).Err()
		return
	}

	_ = p.Buffers.MarkSTT(ctx, sessionID, chunkIndex, text, conf, "done")
	sttPayload, _ := json.Marshal(map[string]any{
		"type":        "stt_result",
		"chunk_index": chunkIndex,
		"text":        text,
		"confidence":  conf,
		"is_final":    true,
	})
	_ = p.Redis.Publish(ctx, respCh, string(sttPayload)).Err()

	// LLM streaming
	start := time.Now()
	_ = p.Buffers.MarkLLM(ctx, sessionID, chunkIndex, "", "processing", 0)
	_ = p.Redis.Publish(ctx, statusCh, `{"type":"status","status":"processing","message":"llm processing","chunk_index":`+strconv.FormatInt(chunkIndex, 10)+`}`).Err()

	prompt := "You are an interview speaking coach. Reply concisely.\n\nUser said:\n" + text

	chunks, errs := p.LLM.StreamAnswer(ctx, prompt)

	full := strings.Builder{}
	seq := int64(0)

	for chunk := range chunks {
		seq++
		full.WriteString(chunk)

		chPayload, _ := json.Marshal(map[string]any{
			"type":        "llm_chunk",
			"chunk_index": chunkIndex,
			"seq":         seq,
			"chunk":       chunk,
		})
		_ = p.Redis.Publish(ctx, respCh, string(chPayload)).Err()
	}

	var streamErr error
	select {
	case streamErr = <-errs:
	default:
	}
	if streamErr != nil {
		log.WithError(streamErr).Error("llm stream failed")
		_ = p.Buffers.MarkLLM(ctx, sessionID, chunkIndex, "", "failed", time.Since(start).Milliseconds())
		_ = p.Redis.Publish(ctx, statusCh, `{"type":"status","status":"failed","message":"llm failed","chunk_index":`+strconv.FormatInt(chunkIndex, 10)+`}`).Err()
		return
	}

	answer := full.String()
	procMS := time.Since(start).Milliseconds()
	_ = p.Buffers.MarkLLM(ctx, sessionID, chunkIndex, answer, "done", procMS)

	donePayload, _ := json.Marshal(map[string]any{
		"type":               "llm_complete",
		"chunk_index":        chunkIndex,
		"full_response":      answer,
		"processing_time_ms": procMS,
	})
	_ = p.Redis.Publish(ctx, respCh, string(donePayload)).Err()
	_ = p.Redis.Publish(ctx, statusCh, `{"type":"status","status":"done","message":"chunk processed","chunk_index":`+strconv.FormatInt(chunkIndex, 10)+`}`).Err()
}
