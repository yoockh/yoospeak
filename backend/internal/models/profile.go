package models

import (
	"time"

	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
)

type Profile struct {
	UserID      string `gorm:"column:user_id;type:uuid;primaryKey" json:"user_id"`
	FullName    string `gorm:"column:full_name;type:text" json:"full_name"`
	PhoneNumber string `gorm:"column:phone_number;type:text" json:"phone_number"`
	CVText      string `gorm:"column:cv_text;type:text" json:"cv_text"`

	Skills pq.StringArray `gorm:"column:skills;type:text[]" json:"skills"`

	// JSONB (save as raw JSON, structure fleksibel)
	Experience  datatypes.JSON `gorm:"column:experience;type:jsonb" json:"experience"`
	Education   datatypes.JSON `gorm:"column:education;type:jsonb" json:"education"`
	Preferences datatypes.JSON `gorm:"column:preferences;type:jsonb" json:"preferences"`

	// pgvector
	CVEmbedding pgvector.Vector `gorm:"column:cv_embedding;type:vector(768)" json:"cv_embedding"`

	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamptz" json:"updated_at"`
}

func (Profile) TableName() string { return "profiles" }
