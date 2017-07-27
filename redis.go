package main

import (
	"github.com/go-redis/redis"
)

var (
	rds *redis.Client
)

func NewRedisClient(cfg *Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: "", // no password set
		DB:       0,  // use yubari
	})

	_, err := client.Ping().Result()
	// Output: PONG <nil>
	return client, err
}
