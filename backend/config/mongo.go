package config

import (
	"context"
	"crypto/tls"
	"errors"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client

// InitMongo initializes MongoDB Atlas connection
// Returns error if connection fails or environment variables are missing
func InitMongo() error {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		return errors.New("MONGO_URI environment variable is not set")
	}

	// Context with timeout for initial connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// MongoDB client options - let driver handle TLS for Atlas automatically
	clientOpts := options.Client().ApplyURI(uri).
		SetServerSelectionTimeout(20 * time.Second).
		SetConnectTimeout(15 * time.Second).
		SetMaxPoolSize(10).
		SetMinPoolSize(1)

	// Configure TLS for Go 1.24+ compatibility with MongoDB Atlas
	// Go 1.24 has stricter TLS requirements that may conflict with Atlas
	if os.Getenv("MONGO_FORCE_TLS_CONFIG") == "true" || os.Getenv("GO_ENV") == "development" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: os.Getenv("MONGO_INSECURE_TLS") == "true",
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS12, // Force TLS 1.2 for Atlas compatibility
		}
		clientOpts = clientOpts.SetTLSConfig(tlsConfig)
	}

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return err
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return err
	}

	MongoClient = client
	return nil
}
