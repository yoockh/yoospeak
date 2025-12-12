package models

import (
	"time"

	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
)

type ConversationLog struct {
	ID        string          `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	UserID    string          `gorm:"column:user_id;type:uuid;index" json:"user_id"`
	SessionID string          `gorm:"column:session_id;type:uuid;index" json:"session_id"`
	Role      string          `gorm:"column:role;type:text" json:"role"` // "user" | "assistant"
	Content   string          `gorm:"column:content;type:text" json:"content"`
	Embedding pgvector.Vector `gorm:"column:embedding;type:vector(768)" json:"embedding"`
	Timestamp time.Time       `gorm:"column:timestamp;type:timestamptz;index" json:"timestamp"`
	Metadata  datatypes.JSON  `gorm:"column:metadata;type:jsonb" json:"metadata"`
}

func (ConversationLog) TableName() string { return "conversation_logs" }
