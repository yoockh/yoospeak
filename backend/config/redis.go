package config

import (
	"context"
	"errors"
	"os"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() error {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return errors.New("REDIS_ADDR environment variable is not set")
	}
	RedisClient = redis.NewClient(&redis.Options{
		Addr: addr,
	})
	_, err := RedisClient.Ping(context.Background()).Result()
	return err
}
