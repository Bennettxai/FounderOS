package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimitConfig struct {
	GetLimit   int
	PostLimit  int
	SyncLimit  int
	WindowSecs int
}

func DefaultMobileRateLimits() RateLimitConfig {
	return RateLimitConfig{
		GetLimit:   100,
		PostLimit:  30,
		SyncLimit:  10,
		WindowSecs: 60,
	}
}

type rateBucket struct {
	mu        sync.Mutex
	count     int
	resetTime time.Time
}

type RateLimiter struct {
	config  RateLimitConfig
	buckets sync.Map // key -> *rateBucket
}

func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{config: config}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		now := time.Now()
		rl.buckets.Range(func(key, value interface{}) bool {
			if bucket, ok := value.(*rateBucket); ok {
				if now.After(bucket.resetTime) {
					rl.buckets.Delete(key)
				}
			}
			return true
		})
	}
}

func (rl *RateLimiter) check(key string, limit int) (bool, int, time.Time) {
	now := time.Now()
	window := time.Duration(rl.config.WindowSecs) * time.Second

	val, _ := rl.buckets.LoadOrStore(key, &rateBucket{
		count:     0,
		resetTime: now.Add(window),
	})
	bucket := val.(*rateBucket)

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	if now.After(bucket.resetTime) {
		bucket.count = 0
		bucket.resetTime = now.Add(window)
	}

	bucket.count++
	remaining := limit - bucket.count
	if remaining < 0 {
		remaining = 0
	}

	return bucket.count <= limit, remaining, bucket.resetTime
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetCurrentUser(c)
		deviceID := GetDeviceID(c)

		var userKey string
		if user != nil {
			userKey = user.ID
		} else {
			userKey = "anon"
		}
		baseKey := userKey + ":" + deviceID

		var limit int
		var bucketType string

		switch c.Request.Method {
		case http.MethodGet:
			if c.FullPath() == "/api/mobile/v1/sync" {
				limit = rl.config.SyncLimit
				bucketType = "sync"
			} else {
				limit = rl.config.GetLimit
				bucketType = "get"
			}
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			limit = rl.config.PostLimit
			bucketType = "post"
		default:
			limit = rl.config.GetLimit
			bucketType = "get"
		}

		key := baseKey + ":" + bucketType
		allowed, remaining, resetTime := rl.check(key, limit)

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", resetTime.Format(time.RFC3339))

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":        "RATE_LIMITED",
					"message":     "Too many requests",
					"retry_after": int(time.Until(resetTime).Seconds()),
				},
			})
			return
		}

		c.Next()
	}
}
