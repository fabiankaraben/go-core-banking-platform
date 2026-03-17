package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// CorrelationID injects a unique X-Correlation-ID header into every request.
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Correlation-ID")
		if id == "" {
			id = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Correlation-ID", id)
		ctx := context.WithValue(r.Context(), correlationKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type correlationKey struct{}

// GetCorrelationID returns the correlation ID from the context.
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationKey{}).(string); ok {
		return id
	}
	return ""
}

// RateLimiter implements a sliding-window rate limiter per client IP using Redis.
type RateLimiter struct {
	client *redis.Client
	rpm    int
	logger *zap.Logger
}

// NewRateLimiter creates a new RateLimiter middleware.
func NewRateLimiter(redisAddr string, rpm int, logger *zap.Logger) *RateLimiter {
	client := redis.NewClient(&redis.Options{Addr: redisAddr})
	return &RateLimiter{client: client, rpm: rpm, logger: logger}
}

// Limit is the middleware handler.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		key := fmt.Sprintf("ratelimit:%s", ip)
		ctx := r.Context()

		pipe := rl.client.Pipeline()
		incr := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, time.Minute)
		if _, err := pipe.Exec(ctx); err != nil {
			rl.logger.Warn("redis rate limit error, allowing through", zap.Error(err))
			next.ServeHTTP(w, r)
			return
		}

		if incr.Val() > int64(rl.rpm) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error":"rate limit exceeded"}`)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Close shuts down the Redis client.
func (rl *RateLimiter) Close() error {
	return rl.client.Close()
}
