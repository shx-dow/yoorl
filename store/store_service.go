package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

type Store interface {
	SaveUrlMapping(shortUrl string, originalUrl string, userId string)
	RetrieveInitialUrl(shortUrl string) (string, error)
	DeleteUrlMapping(shortUrl string)
}

type RedisStore struct {
	redisClient *redis.Client
}

var (
	defaultStore Store
	ctx          = context.Background()
)

const CacheDuration = 6 * time.Hour

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func SetStore(s Store) {
	defaultStore = s
}

func InitializeStore() Store {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	pong, err := redisClient.Ping(ctx).Result()

	if err != nil {
		panic(fmt.Sprintf("Error init Redis: %v", err))
	}

	fmt.Printf("\nRedis started successfully: pong message = {%s}", pong)

	store := &RedisStore{redisClient: redisClient}
	SetStore(store)
	return store
}

func SaveUrlMapping(shortUrl string, originalUrl string, userId string) {
	defaultStore.SaveUrlMapping(shortUrl, originalUrl, userId)
}

func RetrieveInitialUrl(shortUrl string) (string, error) {
	return defaultStore.RetrieveInitialUrl(shortUrl)
}

func DeleteUrlMapping(shortUrl string) {
	defaultStore.DeleteUrlMapping(shortUrl)
}

func (s *RedisStore) SaveUrlMapping(shortUrl string, originalUrl string, userId string) {
	err := s.redisClient.Set(ctx, shortUrl, originalUrl, CacheDuration).Err()
	if err != nil {
		panic(fmt.Sprintf("Failed saving key url | Error: %v - shortUrl: %s - originalUrl: %s\n", err, shortUrl, originalUrl))
	}
}

func (s *RedisStore) RetrieveInitialUrl(shortUrl string) (string, error) {
	result, err := s.redisClient.Get(ctx, shortUrl).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", fmt.Errorf("short URL %s not found", shortUrl)
		}
		return "", fmt.Errorf("failed to retrieve short URL %s: %w", shortUrl, err)
	}
	return result, nil
}

func (s *RedisStore) DeleteUrlMapping(shortUrl string) {
	err := s.redisClient.Del(ctx, shortUrl).Err()
	if err != nil {
		panic(fmt.Sprintf("Failed deleting key url | Error: %v - shortUrl: %s\n", err, shortUrl))
	}
}
