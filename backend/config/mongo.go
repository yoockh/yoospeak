package config

import (
	"context"
	"errors"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client

func InitMongo() error {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		return errors.New("MONGO_URI environment variable is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(20*time.Second).
		SetConnectTimeout(15*time.Second),
	)
	if err != nil {
		return err
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return err
	}

	MongoClient = client
	return nil
}
