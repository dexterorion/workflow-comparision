package redis

import (
	"github.com/go-redis/redis/v8"
)

type RedisConnection interface {
	GetConn() *redis.Client
	NoKeyError(err error) bool
}

type redisConnectionImpl struct {
	Conn *redis.Client
}

func NewRedisConnection() RedisConnection {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis-master:6379",
		Password: "redis", // no password set
		DB:       0,       // use default DB
	})

	return &redisConnectionImpl{
		Conn: rdb,
	}
}

func (r *redisConnectionImpl) GetConn() *redis.Client {
	return r.Conn
}

func (r *redisConnectionImpl) NoKeyError(err error) bool {
	if err == redis.Nil {
		return true
	}

	return false
}
