package config

import (
	"context"
	"errors"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func EnsureMongoIndexes() error {
	if MongoClient == nil {
		return errors.New("MongoClient is nil; call InitMongo() first")
	}

	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "yoospeak"
	}
	db := MongoClient.Database(dbName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// realtime_buffer indexes
	realtime := db.Collection("realtime_buffer")
	_, err := realtime.Indexes().CreateMany(ctx, []mongo.IndexModel{
		// 1) TTL index: expire at ExpiresAt (must be Date)
		{
			Keys: bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().
				SetName("ttl_expires_at").
				SetExpireAfterSeconds(0),
		},
		// 2) Ensure no duplicate chunk per session
		{
			Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "chunk_index", Value: 1}},
			Options: options.Index().
				SetName("uniq_session_chunk").
				SetUnique(true),
		},
		// 3) Query helper
		{
			Keys:    bson.D{{Key: "session_id", Value: 1}, {Key: "timestamp", Value: -1}},
			Options: options.Index().SetName("by_session_ts"),
		},
	})
	if err != nil {
		return err
	}

	// sessions indexes (recommended)
	sessions := db.Collection("sessions")
	_, err = sessions.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "session_id", Value: 1}},
			Options: options.Index().
				SetName("uniq_session_id").
				SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("by_user_created"),
		},
	})
	return err
}
