package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type bucket struct {
	tokens     float64
	lastCheck  time.Time
	lastAccess time.Time
}

type TokenBucket struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64
	capacity float64
	done     chan struct{}
}

func NewTokenBucket(rate, capacity int) *TokenBucket {
	return &TokenBucket{
		buckets:  make(map[string]*bucket),
		rate:     float64(rate),
		capacity: float64(capacity),
		done:     make(chan struct{}),
	}
}

func (tb *TokenBucket) Start() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				tb.cleanup()
			case <-tb.done:
				return
			}
		}
	}()
}

func (tb *TokenBucket) Stop() {
	close(tb.done)
}

func (tb *TokenBucket) cleanup() {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	threshold := time.Now().Add(-30 * time.Minute)
	for key, b := range tb.buckets {
		if b.lastAccess.Before(threshold) {
			delete(tb.buckets, key)
		}
	}
}

func (tb *TokenBucket) Allow(key string) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	b, ok := tb.buckets[key]
	if !ok {
		b = &bucket{tokens: tb.capacity, lastCheck: now, lastAccess: now}
		tb.buckets[key] = b
	}

	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens += elapsed * tb.rate
	if b.tokens > tb.capacity {
		b.tokens = tb.capacity
	}
	b.lastCheck = now
	b.lastAccess = now

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}

func RateLimit(limiter *TokenBucket) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}
		c.Next()
	}
}
