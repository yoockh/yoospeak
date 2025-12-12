package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RealtimeBuffer struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SessionID  string             `bson:"session_id" json:"session_id"`
	ChunkIndex int64              `bson:"chunk_index" json:"chunk_index"`

	AudioURL    *string `bson:"audio_url,omitempty" json:"audio_url,omitempty"`
	AudioBase64 *string `bson:"audio_base64,omitempty" json:"audio_base64,omitempty"`

	RawText       string  `bson:"raw_text,omitempty" json:"raw_text,omitempty"`
	STTStatus     string  `bson:"stt_status" json:"stt_status"` // pending|processing|done|failed
	STTConfidence float64 `bson:"stt_confidence,omitempty" json:"stt_confidence,omitempty"`

	LLMStatus   string `bson:"llm_status" json:"llm_status"` // pending|processing|done|failed
	LLMResponse string `bson:"llm_response,omitempty" json:"llm_response,omitempty"`

	ProcessingTimeMS int64     `bson:"processing_time_ms,omitempty" json:"processing_time_ms,omitempty"`
	Timestamp        time.Time `bson:"timestamp" json:"timestamp"`

	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"` // for TTL index
}
