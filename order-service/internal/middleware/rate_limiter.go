package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb      *redis.Client
	limit    int
	window   time.Duration
	keySpace string
}

func NewRateLimiter(rdb *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		rdb:      rdb,
		limit:    limit,
		window:   window,
		keySpace: "rate-limit",
	}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		bucket := time.Now().Unix() / int64(rl.window.Seconds())
		key := fmt.Sprintf("%s:%s:%d", rl.keySpace, ip, bucket)

		count, err := rl.rdb.Incr(c.Request.Context(), key).Result()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "rate limiter error"})
			return
		}

		if count == 1 {
			_ = rl.rdb.Expire(c.Request.Context(), key, rl.window+time.Second).Err()
		}

		if count > int64(rl.limit) {
			c.Header("Retry-After", strconv.Itoa(int(rl.window.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests",
			})
			return
		}

		c.Next()
	}
}
