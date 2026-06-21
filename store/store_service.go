package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type ClickEvent struct {
	Timestamp time.Time `json:"timestamp"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer"`
}

type Analytics struct {
	ShortUrl     string       `json:"short_url"`
	TotalClicks  int64        `json:"total_clicks"`
	RecentClicks []ClickEvent `json:"recent_clicks"`
}

type UrlEntry struct {
	ShortUrl    string    `json:"short_url"`
	LongUrl     string    `json:"long_url"`
	UserId      string    `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	TotalClicks int64     `json:"total_clicks"`
}

type Store interface {
	SaveUrlMapping(shortUrl string, originalUrl string, userId string)
	RetrieveInitialUrl(shortUrl string) (string, error)
	DeleteUrlMapping(shortUrl string)
	RecordClick(shortUrl string, event ClickEvent)
	GetAnalytics(shortUrl string) (*Analytics, error)
	ListUrls(userId string) ([]*UrlEntry, error)
}

type RedisStore struct {
	redisClient *redis.Client
}

var (
	defaultStore Store
	ctx          = context.Background()
)

const (
	CacheDuration      = 6 * time.Hour
	analyticsListKey   = "analytics:visits:%s"
	analyticsCountKey  = "analytics:count:%s"
	urlIndexKey        = "yoorl:urls"
	urlMetaKey         = "yoorl:meta:%s"
	maxRecentVisits    = 100
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func SetStore(s Store) {
	defaultStore = s
}

func InitializeStore() error {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}

	store := &RedisStore{redisClient: redisClient}
	SetStore(store)
	return nil
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

func RecordClick(shortUrl string, event ClickEvent) {
	defaultStore.RecordClick(shortUrl, event)
}

func GetAnalytics(shortUrl string) (*Analytics, error) {
	return defaultStore.GetAnalytics(shortUrl)
}

func ListUrls(userId string) ([]*UrlEntry, error) {
	return defaultStore.ListUrls(userId)
}

func (s *RedisStore) SaveUrlMapping(shortUrl string, originalUrl string, userId string) {
	pipe := s.redisClient.Pipeline()

	pipe.Set(ctx, shortUrl, originalUrl, CacheDuration)
	pipe.SAdd(ctx, urlIndexKey, shortUrl)

	meta := map[string]string{
		"long_url": originalUrl,
		"user_id":  userId,
	}
	existing, _ := s.redisClient.HGetAll(ctx, fmt.Sprintf(urlMetaKey, shortUrl)).Result()
	if createdAt, ok := existing["created_at"]; ok {
		meta["created_at"] = createdAt
	} else {
		meta["created_at"] = time.Now().Format(time.RFC3339)
	}

	pipe.HSet(ctx, fmt.Sprintf(urlMetaKey, shortUrl), meta)

	_, err := pipe.Exec(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed saving url mapping | Error: %v - shortUrl: %s\n", err, shortUrl))
	}
}

func (s *RedisStore) RetrieveInitialUrl(shortUrl string) (string, error) {
	result, err := s.redisClient.Get(ctx, shortUrl).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			meta, metaErr := s.redisClient.HGetAll(ctx, fmt.Sprintf(urlMetaKey, shortUrl)).Result()
			if metaErr != nil || len(meta) == 0 {
				return "", fmt.Errorf("short URL %s not found", shortUrl)
			}
			return meta["long_url"], nil
		}
		return "", fmt.Errorf("failed to retrieve short URL %s: %w", shortUrl, err)
	}
	return result, nil
}

func (s *RedisStore) DeleteUrlMapping(shortUrl string) {
	pipe := s.redisClient.Pipeline()
	pipe.Del(ctx, shortUrl)
	pipe.Del(ctx, fmt.Sprintf(urlMetaKey, shortUrl))
	pipe.SRem(ctx, urlIndexKey, shortUrl)
	pipe.Del(ctx, fmt.Sprintf(analyticsCountKey, shortUrl))
	pipe.Del(ctx, fmt.Sprintf(analyticsListKey, shortUrl))
	_, err := pipe.Exec(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed deleting url mapping | Error: %v - shortUrl: %s\n", err, shortUrl))
	}
}

func (s *RedisStore) RecordClick(shortUrl string, event ClickEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	listKey := fmt.Sprintf(analyticsListKey, shortUrl)
	countKey := fmt.Sprintf(analyticsCountKey, shortUrl)

	pipe := s.redisClient.Pipeline()
	pipe.LPush(ctx, listKey, string(data))
	pipe.LTrim(ctx, listKey, 0, maxRecentVisits-1)
	pipe.Incr(ctx, countKey)
	pipe.Expire(ctx, listKey, CacheDuration)
	pipe.Expire(ctx, countKey, CacheDuration)
	_, _ = pipe.Exec(ctx)
}

func (s *RedisStore) GetAnalytics(shortUrl string) (*Analytics, error) {
	listKey := fmt.Sprintf(analyticsListKey, shortUrl)
	countKey := fmt.Sprintf(analyticsCountKey, shortUrl)

	total, err := s.redisClient.Get(ctx, countKey).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return &Analytics{ShortUrl: shortUrl, TotalClicks: 0, RecentClicks: []ClickEvent{}}, nil
		}
		return nil, fmt.Errorf("failed to get analytics count: %w", err)
	}

	rawList, err := s.redisClient.LRange(ctx, listKey, 0, maxRecentVisits-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics visits: %w", err)
	}

	clicks := make([]ClickEvent, 0, len(rawList))
	for _, raw := range rawList {
		var event ClickEvent
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			continue
		}
		clicks = append(clicks, event)
	}

	return &Analytics{
		ShortUrl:     shortUrl,
		TotalClicks:  total,
		RecentClicks: clicks,
	}, nil
}

func (s *RedisStore) ListUrls(userId string) ([]*UrlEntry, error) {
	shortUrls, err := s.redisClient.SMembers(ctx, urlIndexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list URLs: %w", err)
	}

	if len(shortUrls) == 0 {
		return []*UrlEntry{}, nil
	}

	pipe := s.redisClient.Pipeline()
	metaCmds := make([]*redis.StringStringMapCmd, len(shortUrls))
	countCmds := make([]*redis.StringCmd, len(shortUrls))
	for i, u := range shortUrls {
		metaCmds[i] = pipe.HGetAll(ctx, fmt.Sprintf(urlMetaKey, u))
		countCmds[i] = pipe.Get(ctx, fmt.Sprintf(analyticsCountKey, u))
	}
	_, _ = pipe.Exec(ctx)

	entries := make([]*UrlEntry, 0, len(shortUrls))
	for i, su := range shortUrls {
		meta := metaCmds[i].Val()
		if len(meta) == 0 {
			continue
		}

		if userId != "" && meta["user_id"] != userId {
			continue
		}

		createdAt, _ := time.Parse(time.RFC3339, meta["created_at"])
		totalClicks, _ := strconv.ParseInt(countCmds[i].Val(), 10, 64)

		entries = append(entries, &UrlEntry{
			ShortUrl:    su,
			LongUrl:     meta["long_url"],
			UserId:      meta["user_id"],
			CreatedAt:   createdAt,
			TotalClicks: totalClicks,
		})
	}

	return entries, nil
}
