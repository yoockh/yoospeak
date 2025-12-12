package mongo

import (
	"context"
	"time"

	"github.com/yoockh/yoospeak/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BufferRepository interface {
	InsertChunk(ctx context.Context, b *models.RealtimeBuffer) error
	UpdateSTT(ctx context.Context, sessionID string, chunkIndex int64, rawText string, confidence float64, status string) error
	UpdateLLM(ctx context.Context, sessionID string, chunkIndex int64, response string, status string, processingMS int64) error
	ListBySession(ctx context.Context, sessionID string, limit int64) ([]models.RealtimeBuffer, error)
}

type bufferRepo struct {
	col *mongo.Collection
}

func NewBufferRepo(db *mongo.Database) BufferRepository {
	return &bufferRepo{col: db.Collection("realtime_buffer")}
}

func (r *bufferRepo) InsertChunk(ctx context.Context, b *models.RealtimeBuffer) error {
	if b.Timestamp.IsZero() {
		b.Timestamp = time.Now().UTC()
	}
	_, err := r.col.InsertOne(ctx, b)
	return err
}

func (r *bufferRepo) UpdateSTT(ctx context.Context, sessionID string, chunkIndex int64, rawText string, confidence float64, status string) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"session_id": sessionID, "chunk_index": chunkIndex},
		bson.M{"$set": bson.M{
			"raw_text":       rawText,
			"stt_confidence": confidence,
			"stt_status":     status,
		}},
	)
	return err
}

func (r *bufferRepo) UpdateLLM(ctx context.Context, sessionID string, chunkIndex int64, response string, status string, processingMS int64) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"session_id": sessionID, "chunk_index": chunkIndex},
		bson.M{"$set": bson.M{
			"llm_response":       response,
			"llm_status":         status,
			"processing_time_ms": processingMS,
		}},
	)
	return err
}

func (r *bufferRepo) ListBySession(ctx context.Context, sessionID string, limit int64) ([]models.RealtimeBuffer, error) {
	if limit <= 0 {
		limit = 200
	}

	cur, err := r.col.Find(ctx,
		bson.M{"session_id": sessionID},
		options.Find().
			SetSort(bson.D{{Key: "chunk_index", Value: 1}}).
			SetLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []models.RealtimeBuffer
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}
