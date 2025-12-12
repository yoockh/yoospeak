package mongo

import (
	"context"
	"errors"
	"time"

	"github.com/yoockh/yoospeak/internal/models"
	"github.com/yoockh/yoospeak/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type SessionRepository interface {
	Create(ctx context.Context, s *models.Session) error
	GetBySessionID(ctx context.Context, sessionID string) (*models.Session, error)
	End(ctx context.Context, sessionID string, endedAt time.Time, durationSeconds int64) error
	SetStatus(ctx context.Context, sessionID, status string) error
}

type sessionRepo struct {
	col *mongo.Collection
}

func NewSessionRepo(db *mongo.Database) SessionRepository {
	return &sessionRepo{col: db.Collection("sessions")}
}

func (r *sessionRepo) Create(ctx context.Context, s *models.Session) error {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	_, err := r.col.InsertOne(ctx, s)
	return err
}

func (r *sessionRepo) GetBySessionID(ctx context.Context, sessionID string) (*models.Session, error) {
	var s models.Session
	err := r.col.FindOne(ctx, bson.M{"session_id": sessionID}).Decode(&s)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, utils.ErrNotFound
	}
	return &s, err
}

func (r *sessionRepo) End(ctx context.Context, sessionID string, endedAt time.Time, durationSeconds int64) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"session_id": sessionID},
		bson.M{"$set": bson.M{
			"status":           "ended",
			"ended_at":         endedAt.UTC(),
			"duration_seconds": durationSeconds,
		}},
	)
	return err
}

func (r *sessionRepo) SetStatus(ctx context.Context, sessionID, status string) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"session_id": sessionID},
		bson.M{"$set": bson.M{"status": status}},
	)
	return err
}
