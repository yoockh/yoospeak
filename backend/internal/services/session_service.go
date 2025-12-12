package services

import (
	"context"
	"errors"
	"time"

	"github.com/yoockh/yoospeak/internal/models"
	mongorepo "github.com/yoockh/yoospeak/internal/repositories/mongo"
	"github.com/yoockh/yoospeak/internal/utils"

	"github.com/google/uuid"
)

type SessionService interface {
	Start(ctx context.Context, userID, typ, language string, md models.SessionMetadata) (*models.Session, error)
	Get(ctx context.Context, sessionID string) (*models.Session, error)
	End(ctx context.Context, sessionID string) (*models.Session, error)
	SetStatus(ctx context.Context, sessionID, status string) error
}

type sessionService struct {
	sessions mongorepo.SessionRepository
}

func NewSessionService(sessions mongorepo.SessionRepository) SessionService {
	return &sessionService{sessions: sessions}
}

func (s *sessionService) Start(ctx context.Context, userID, typ, language string, md models.SessionMetadata) (*models.Session, error) {
	const op = "SessionService.Start"

	if userID == "" || typ == "" || language == "" {
		return nil, utils.E(utils.CodeInvalidArgument, op, "user_id, type, and language are required", nil)
	}

	now := time.Now().UTC()
	session := &models.Session{
		SessionID:       uuid.NewString(),
		UserID:          userID,
		Type:            typ,
		Language:        language,
		Status:          "active",
		Metadata:        md,
		CreatedAt:       now,
		DurationSeconds: 0,
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, utils.E(utils.CodeInternal, op, "failed to create session", err)
	}
	return session, nil
}

func (s *sessionService) Get(ctx context.Context, sessionID string) (*models.Session, error) {
	const op = "SessionService.Get"

	if sessionID == "" {
		return nil, utils.E(utils.CodeInvalidArgument, op, "session_id is required", nil)
	}

	out, err := s.sessions.GetBySessionID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, utils.ErrNotFound) {
			return nil, utils.E(utils.CodeNotFound, op, "session not found", err)
		}
		return nil, utils.E(utils.CodeInternal, op, "failed to get session", err)
	}
	return out, nil
}

func (s *sessionService) End(ctx context.Context, sessionID string) (*models.Session, error) {
	const op = "SessionService.End"

	if sessionID == "" {
		return nil, utils.E(utils.CodeInvalidArgument, op, "session_id is required", nil)
	}

	ss, err := s.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	dur := int64(now.Sub(ss.CreatedAt).Seconds())
	if dur < 0 {
		dur = 0
	}

	if err := s.sessions.End(ctx, sessionID, now, dur); err != nil {
		return nil, utils.E(utils.CodeInternal, op, "failed to end session", err)
	}

	ss.Status = "ended"
	ss.EndedAt = &now
	ss.DurationSeconds = dur
	return ss, nil
}

func (s *sessionService) SetStatus(ctx context.Context, sessionID, status string) error {
	const op = "SessionService.SetStatus"

	if sessionID == "" || status == "" {
		return utils.E(utils.CodeInvalidArgument, op, "session_id and status are required", nil)
	}
	if err := s.sessions.SetStatus(ctx, sessionID, status); err != nil {
		return utils.E(utils.CodeInternal, op, "failed to set status", err)
	}
	return nil
}
