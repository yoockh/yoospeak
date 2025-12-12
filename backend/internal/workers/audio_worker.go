package workers

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yoockh/yoospeak/internal/services"
)

type AudioWorkerPool struct {
	Redis      *redis.Client
	Buffers    services.BufferService
	NumWorkers int

	Stream         string
	Group          string
	ConsumerPrefix string
}

func (p *AudioWorkerPool) Start(ctx context.Context) error {
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

	// Ensure group exists
	_ = p.Redis.XGroupCreateMkStream(ctx, p.Stream, p.Group, "0").Err()
	// ignore BUSYGROUP

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
			// redis.Nil when timeout -> continue
			if err == redis.Nil {
				continue
			}
			// backoff
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, stream := range res {
			for _, msg := range stream.Messages {
				p.handleMsg(ctx, msg)

				// ACK regardless (you can make this conditional later)
				_ = p.Redis.XAck(ctx, p.Stream, p.Group, msg.ID).Err()
			}
		}
	}
}

func (p *AudioWorkerPool) handleMsg(ctx context.Context, msg redis.XMessage) {
	getStr := func(k string) string {
		v, ok := msg.Values[k]
		if !ok || v == nil {
			return ""
		}
		switch t := v.(type) {
		case string:
			return t
		default:
			return ""
		}
	}

	sessionID := getStr("session_id")
	chunkIndexStr := getStr("chunk_index")
	if sessionID == "" || chunkIndexStr == "" {
		return
	}
	chunkIndex, _ := strconv.ParseInt(chunkIndexStr, 10, 64)

	respCh := "session:" + sessionID + ":response"
	statusCh := "session:" + sessionID + ":status"

	// mark STT processing -> done (dummy)
	_ = p.Buffers.MarkSTT(ctx, sessionID, chunkIndex, "", 0, "processing")
	_ = p.Redis.Publish(ctx, statusCh, `{"type":"status","status":"processing","message":"stt processing","chunk_index":`+strconv.FormatInt(chunkIndex, 10)+`}`).Err()

	// dummy "transcription"
	audioB64 := getStr("audio_base64")
	text := "..."
	if audioB64 != "" {
		text = "transcribed(" + strconv.Itoa(len(audioB64)) + "b64)"
	}
	text = strings.TrimSpace(text)

	_ = p.Buffers.MarkSTT(ctx, sessionID, chunkIndex, text, 0.9, "done")

	sttPayload, _ := json.Marshal(map[string]any{
		"type":        "stt_result",
		"chunk_index": chunkIndex,
		"text":        text,
		"confidence":  0.9,
		"is_final":    true,
	})
	_ = p.Redis.Publish(ctx, respCh, string(sttPayload)).Err()

	// mark LLM processing -> done (dummy)
	start := time.Now()
	_ = p.Buffers.MarkLLM(ctx, sessionID, chunkIndex, "", "processing", 0)
	_ = p.Redis.Publish(ctx, statusCh, `{"type":"status","status":"processing","message":"llm processing","chunk_index":`+strconv.FormatInt(chunkIndex, 10)+`}`).Err()

	answer := "Dummy answer for: " + text
	procMS := time.Since(start).Milliseconds()

	_ = p.Buffers.MarkLLM(ctx, sessionID, chunkIndex, answer, "done", procMS)

	llmPayload, _ := json.Marshal(map[string]any{
		"type":               "llm_complete",
		"full_response":      answer,
		"processing_time_ms": procMS,
	})
	_ = p.Redis.Publish(ctx, respCh, string(llmPayload)).Err()

	_ = p.Redis.Publish(ctx, statusCh, `{"type":"status","status":"done","message":"chunk processed","chunk_index":`+strconv.FormatInt(chunkIndex, 10)+`}`).Err()
}
