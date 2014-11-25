package redis

import (
	"time"

	libredis "github.com/therealbill/libredis/client"
)

var (
	defaultRedis *RedisClient
)

func UseRedis(ip string) {
	defaultRedis = NewRedisClient(ip)
}

func Redis() *RedisClient {
	return defaultRedis
}

type RedisClient struct {
	*libredis.Redis
}

func NewRedisClient(ip string) *RedisClient {
	config := DefaultRedisConfig(ip)
	client, err := libredis.DialWithConfig(config)
	if err != nil {
		panic(err)
	}

	return &RedisClient{client}
}

func DefaultRedisConfig(ip string) *libredis.DialConfig {
	return &libredis.DialConfig{
		Network:  "tcp",
		Address:  ip + ":6379",
		Database: 0,
		Password: "",
		Timeout:  2 * time.Second,
		MaxIdle:  10,
	}
}
