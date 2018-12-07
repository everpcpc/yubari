package main

import (
	"github.com/go-redis/redis"
)

var (
	redisClient *redis.Client
)

// NewRedisClient ...
func NewRedisClient(cfg *RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	_, err := client.Ping().Result()
	// Output: PONG <nil>
	return client, err
}
