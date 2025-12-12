package postgres

import (
	"context"
	"errors"

	"github.com/yoockh/yoospeak/internal/models"
	"github.com/yoockh/yoospeak/internal/utils"
	"gorm.io/gorm"
)

type ConversationRepo interface {
	Insert(ctx context.Context, log *models.ConversationLog) error
	ListBySession(ctx context.Context, userID, sessionID string, limit int) ([]models.ConversationLog, error)
	LatestN(ctx context.Context, userID string, n int) ([]models.ConversationLog, error)
	GetByID(ctx context.Context, id string) (*models.ConversationLog, error)
}

type conversationRepo struct {
	db *gorm.DB
}

func NewConversationRepo(db *gorm.DB) ConversationRepo {
	return &conversationRepo{db: db}
}

func (r *conversationRepo) Insert(ctx context.Context, log *models.ConversationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *conversationRepo) ListBySession(ctx context.Context, userID, sessionID string, limit int) ([]models.ConversationLog, error) {
	if limit <= 0 {
		limit = 50
	}

	var rows []models.ConversationLog
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *conversationRepo) LatestN(ctx context.Context, userID string, n int) ([]models.ConversationLog, error) {
	if n <= 0 {
		n = 5
	}
	var rows []models.ConversationLog
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("timestamp DESC").
		Limit(n).
		Find(&rows).Error
	return rows, err
}

func (r *conversationRepo) GetByID(ctx context.Context, id string) (*models.ConversationLog, error) {
	var row models.ConversationLog
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, utils.ErrNotFound
	}
	return &row, err
}
