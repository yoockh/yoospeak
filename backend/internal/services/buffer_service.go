package services

import (
	"context"
	"time"

	"github.com/yoockh/yoospeak/internal/models"
	mongorepo "github.com/yoockh/yoospeak/internal/repositories/mongo"
	"github.com/yoockh/yoospeak/internal/utils"
)

type BufferService interface {
	InsertAudioChunk(ctx context.Context, sessionID string, chunkIndex int64, audioURL, audioBase64 *string) (*models.RealtimeBuffer, error)
	MarkSTT(ctx context.Context, sessionID string, chunkIndex int64, rawText string, confidence float64, status string) error
	MarkLLM(ctx context.Context, sessionID string, chunkIndex int64, response string, status string, processingMS int64) error
	ListBySession(ctx context.Context, sessionID string, limit int64) ([]models.RealtimeBuffer, error)
}

type bufferService struct {
	buffers mongorepo.BufferRepository
	ttl     time.Duration
}

func NewBufferService(buffers mongorepo.BufferRepository, ttl time.Duration) BufferService {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &bufferService{buffers: buffers, ttl: ttl}
}

func (s *bufferService) InsertAudioChunk(ctx context.Context, sessionID string, chunkIndex int64, audioURL, audioBase64 *string) (*models.RealtimeBuffer, error) {
	const op = "BufferService.InsertAudioChunk"

	if sessionID == "" || chunkIndex <= 0 {
		return nil, utils.E(utils.CodeInvalidArgument, op, "session_id is required and chunk_index must be > 0", nil)
	}

	now := time.Now().UTC()
	doc := &models.RealtimeBuffer{
		SessionID:   sessionID,
		ChunkIndex:  chunkIndex,
		AudioURL:    audioURL,
		AudioBase64: audioBase64,

		STTStatus: "pending",
		LLMStatus: "pending",

		Timestamp: now,
		ExpiresAt: now.Add(s.ttl),
	}

	if err := s.buffers.InsertChunk(ctx, doc); err != nil {
		return nil, utils.E(utils.CodeInternal, op, "failed to insert audio chunk", err)
	}
	return doc, nil
}

func (s *bufferService) MarkSTT(ctx context.Context, sessionID string, chunkIndex int64, rawText string, confidence float64, status string) error {
	const op = "BufferService.MarkSTT"

	if sessionID == "" || chunkIndex <= 0 || status == "" {
		return utils.E(utils.CodeInvalidArgument, op, "session_id, chunk_index (>0), and status are required", nil)
	}
	if err := s.buffers.UpdateSTT(ctx, sessionID, chunkIndex, rawText, confidence, status); err != nil {
		return utils.E(utils.CodeInternal, op, "failed to update stt fields", err)
	}
	return nil
}

func (s *bufferService) MarkLLM(ctx context.Context, sessionID string, chunkIndex int64, response string, status string, processingMS int64) error {
	const op = "BufferService.MarkLLM"

	if sessionID == "" || chunkIndex <= 0 || status == "" {
		return utils.E(utils.CodeInvalidArgument, op, "session_id, chunk_index (>0), and status are required", nil)
	}
	if err := s.buffers.UpdateLLM(ctx, sessionID, chunkIndex, response, status, processingMS); err != nil {
		return utils.E(utils.CodeInternal, op, "failed to update llm fields", err)
	}
	return nil
}

func (s *bufferService) ListBySession(ctx context.Context, sessionID string, limit int64) ([]models.RealtimeBuffer, error) {
	const op = "BufferService.ListBySession"

	if sessionID == "" {
		return nil, utils.E(utils.CodeInvalidArgument, op, "session_id is required", nil)
	}
	out, err := s.buffers.ListBySession(ctx, sessionID, limit)
	if err != nil {
		return nil, utils.E(utils.CodeInternal, op, "failed to list realtime buffer", err)
	}
	return out, nil
}
