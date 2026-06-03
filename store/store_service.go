package store

import (
	"context"
	"time"

	"github.com/go-redis/redis"
)

type StorageService struct {
	redisClient *redis.Client
}

var (
	storeService = &StorageService{}
	ctx          = context.Background()
)

const CacheDuration = 6 * time.Hour
