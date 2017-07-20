package base

import (
	"github.com/go-redis/redis"
	"time"
)

type Redis struct {
	client     *redis.Client
	expiration time.Duration
}

func (r *Redis) Get(key string) (string, error) {
	cmd := r.client.Get(key)
	if cmd.Err() == nil {
		return cmd.Result()
	}
	return "", cmd.Err()
}

func (r *Redis) Set(key, value string) error {
	return r.client.Set(key, value, r.expiration).Err()
}

func NewRedis(address, password string) (*Redis, error) {
	return &Redis{client: redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       0,
	}), expiration: time.Hour * 24 * 30}, nil
}
