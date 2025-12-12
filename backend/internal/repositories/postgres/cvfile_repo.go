package postgres

import (
	"context"

	"github.com/yoockh/yoospeak/internal/models"
	"gorm.io/gorm"
)

type CVFileRepository interface {
	Insert(ctx context.Context, f *models.CVFile) error
	LatestByUser(ctx context.Context, userID string) (*models.CVFile, error)
}

type cvFileRepo struct {
	db *gorm.DB
}

func NewCVFileRepo(db *gorm.DB) CVFileRepository {
	return &cvFileRepo{db: db}
}

func (r *cvFileRepo) Insert(ctx context.Context, f *models.CVFile) error {
	return r.db.WithContext(ctx).Create(f).Error
}

func (r *cvFileRepo) LatestByUser(ctx context.Context, userID string) (*models.CVFile, error) {
	var row models.CVFile
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("upload_at DESC").
		Take(&row).Error
	return &row, err
}
