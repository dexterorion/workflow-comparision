package redis

import (
	"github.com/go-redis/redis/v8"
)

type RedisConnection struct {
	Conn *redis.Client
}

func NewRedisConnection() RedisConnection {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis-master:6379",
		Password: "redis", // no password set
		DB:       0,       // use default DB
	})

	return RedisConnection{
		Conn: rdb,
	}
}
