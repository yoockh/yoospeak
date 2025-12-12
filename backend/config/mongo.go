package config

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"strconv"
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

	baseOpts := options.Client().ApplyURI(uri).
		SetServerSelectionTimeout(20 * time.Second).
		SetConnectTimeout(15 * time.Second).
		SetMaxPoolSize(10).
		SetMinPoolSize(1)

	// Try default connect first (let driver choose TLS params)
	if client, err := tryConnect(ctx, baseOpts); err == nil {
		MongoClient = client
		return nil
	} else {
		// If TLS-related and user explicitly asked for insecure debug, retry with InsecureSkipVerify
		insecure := false
		if v := os.Getenv("MONGO_INSECURE_TLS"); v != "" {
			if b, err := strconv.ParseBool(v); err == nil && b {
				insecure = b
			}
		}
		if insecure {
			tlsConfig := &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS12,
			}
			fmt.Println("InitMongo: retrying with InsecureSkipVerify=true (debug only)")
			clientOpts := baseOpts.SetTLSConfig(tlsConfig)
			if client, err := tryConnect(ctx, clientOpts); err == nil {
				MongoClient = client
				return nil
			} else {
				return fmt.Errorf("InitMongo: retry with insecure tls failed: %w", err)
			}
		}
		return fmt.Errorf("InitMongo: initial connect failed: %w", err)
	}
}

func tryConnect(ctx context.Context, clientOpts *options.ClientOptions) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}
	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	return client, nil
}
