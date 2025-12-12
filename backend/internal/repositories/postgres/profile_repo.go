package postgres

import (
	"context"
	"errors"

	"github.com/yoockh/yoospeak/internal/models"
	"github.com/yoockh/yoospeak/internal/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProfileRepository interface {
	GetByUserID(ctx context.Context, userID string) (*models.Profile, error)
	Upsert(ctx context.Context, p *models.Profile) error
}

type profileRepo struct {
	db *gorm.DB
}

func NewProfileRepo(db *gorm.DB) ProfileRepository {
	return &profileRepo{db: db}
}

func (r *profileRepo) GetByUserID(ctx context.Context, userID string) (*models.Profile, error) {
	var p models.Profile
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Take(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, utils.ErrNotFound
	}
	return &p, err
}

func (r *profileRepo) Upsert(ctx context.Context, p *models.Profile) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"full_name", "phone_number", "cv_text", "skills", "experience", "education", "cv_embedding", "preferences", "updated_at"}),
		}).
		Create(p).Error
}
