package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client

// InitMongo initializes MongoDB Atlas connection
func InitMongo() error {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		return errors.New("missing MONGO_URI environment variable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	clientOpts := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(20 * time.Second).
		SetConnectTimeout(15 * time.Second).
		SetMaxPoolSize(10).
		SetMinPoolSize(1)

	// Try normal secure TLS first
	client, err := tryConnect(ctx, clientOpts)
	if err == nil {
		MongoClient = client
		return nil
	}

	return fmt.Errorf("MongoDB connection failed: %w", err)
}

func tryConnect(ctx context.Context, opts *options.ClientOptions) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return nil, err
	}

	return client, nil
}
