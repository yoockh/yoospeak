package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Session struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SessionID string             `bson:"session_id" json:"session_id"` // uuid v4
	UserID    string             `bson:"user_id" json:"user_id"`       // uuid from Supabase Auth

	Type     string          `bson:"type" json:"type"`         // interview|casual
	Language string          `bson:"language" json:"language"` // id|en
	Status   string          `bson:"status" json:"status"`     // active|ended|paused
	Metadata SessionMetadata `bson:"metadata,omitempty" json:"metadata,omitempty"`

	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	EndedAt   *time.Time `bson:"ended_at,omitempty" json:"ended_at,omitempty"`

	DurationSeconds int64 `bson:"duration_seconds" json:"duration_seconds"`
}

type SessionMetadata struct {
	InterviewType string `bson:"interview_type,omitempty" json:"interview_type,omitempty"`
	CompanyName   string `bson:"company_name,omitempty" json:"company_name,omitempty"`
	Position      string `bson:"position,omitempty" json:"position,omitempty"`
}
