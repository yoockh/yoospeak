package postgres

import (
	"context"

	"github.com/yoockh/yoospeak/internal/models"
	"gorm.io/gorm"
)

type UserRepository interface {
	HasProfile(ctx context.Context, userID string) (bool, error)
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) HasProfile(ctx context.Context, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Profile{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count > 0, err
}
