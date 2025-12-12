package models

import "time"

type CVFile struct {
	ID       string `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	UserID   string `gorm:"column:user_id;type:uuid;index" json:"user_id"`
	FileName string `gorm:"column:file_name;type:text" json:"file_name"`
	FilePath string `gorm:"column:file_path;type:text" json:"file_path"`

	FileSize int    `gorm:"column:file_size;type:integer" json:"file_size"`
	MimeType string `gorm:"column:mime_type;type:text" json:"mime_type"`

	UploadAt time.Time `gorm:"column:upload_at;type:timestamptz" json:"upload_at"`
}

func (CVFile) TableName() string { return "cv_files" }
