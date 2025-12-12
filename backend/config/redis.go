package config

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() error {
	val := os.Getenv("REDIS_ADDR")
	if val == "" {
		val = os.Getenv("REDIS_URI")
	}
	if val == "" {
		val = os.Getenv("REDIS_URL")
	}
	if val == "" {
		return errors.New("REDIS_ADDR (or REDIS_URI/REDIS_URL) environment variable is not set")
	}

	if strings.HasPrefix(val, "redis://") || strings.HasPrefix(val, "rediss://") {
		opt, err := redis.ParseURL(val)
		if err != nil {
			return err
		}
		RedisClient = redis.NewClient(opt)
	} else {
		RedisClient = redis.NewClient(&redis.Options{Addr: val})
	}

	_, err := RedisClient.Ping(context.Background()).Result()
	return err
}
