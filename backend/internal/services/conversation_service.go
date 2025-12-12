package services

import (
	"context"
	"time"

	"github.com/yoockh/yoospeak/internal/models"
	pgrepo "github.com/yoockh/yoospeak/internal/repositories/postgres"
	"github.com/yoockh/yoospeak/internal/utils"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/datatypes"
)

type ConversationService interface {
	Append(ctx context.Context, userID, sessionID, role, content string, embedding []float32, metadataJSON []byte) (*models.ConversationLog, error)
	ListBySession(ctx context.Context, userID, sessionID string, limit int) ([]models.ConversationLog, error)
}

type conversationService struct {
	convos pgrepo.ConversationRepo
}

func NewConversationService(convos pgrepo.ConversationRepo) ConversationService {
	return &conversationService{convos: convos}
}

func (s *conversationService) Append(ctx context.Context, userID, sessionID, role, content string, embedding []float32, metadataJSON []byte) (*models.ConversationLog, error) {
	const op = "ConversationService.Append"

	if userID == "" || sessionID == "" || role == "" || content == "" {
		return nil, utils.E(utils.CodeInvalidArgument, op, "user_id, session_id, role, and content are required", nil)
	}

	row := &models.ConversationLog{
		ID:        uuid.NewString(),
		UserID:    userID,
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Timestamp: time.Now().UTC(),
		Metadata:  datatypes.JSON(metadataJSON),
	}

	if len(embedding) > 0 {
		row.Embedding = pgvector.NewVector(embedding)
	}

	if err := s.convos.Insert(ctx, row); err != nil {
		return nil, utils.E(utils.CodeInternal, op, "failed to insert conversation log", err)
	}
	return row, nil
}

func (s *conversationService) ListBySession(ctx context.Context, userID, sessionID string, limit int) ([]models.ConversationLog, error) {
	const op = "ConversationService.ListBySession"

	if userID == "" || sessionID == "" {
		return nil, utils.E(utils.CodeInvalidArgument, op, "user_id and session_id are required", nil)
	}

	rows, err := s.convos.ListBySession(ctx, userID, sessionID, limit)
	if err != nil {
		return nil, utils.E(utils.CodeInternal, op, "failed to list conversations", err)
	}
	return rows, nil
}
