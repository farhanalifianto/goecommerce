package cache

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()
var Redis *redis.Client

func ConnectRedis() {
    Redis = redis.NewClient(&redis.Options{
        Addr:     "redis:6379",
        Password: "",
        DB:       0,
    })

    if _, err := Redis.Ping(Ctx).Result(); err != nil {
        log.Fatalf("Failed to connect Redis: %v", err)
    }

    log.Println("Redis connected (user-service)")
}
