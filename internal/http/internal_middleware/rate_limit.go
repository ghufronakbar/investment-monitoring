package internal_middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/response"

	"github.com/redis/go-redis/v9"
)

const (
	defaultAuthRateLimitMaxRequests = 5
	defaultAuthRateLimitWindow      = 5 * time.Minute
	maxRateLimitBodyBytes           = 1 << 20
)

type AuthRateLimiter struct {
	rdb         *redis.Client
	prefix      string
	maxRequests int
	window      time.Duration
	mu          sync.Mutex
	memory      map[string]memoryRateLimitCounter
}

type memoryRateLimitCounter struct {
	Count     int
	ExpiresAt time.Time
}

func NewAuthRateLimiter(rdb *redis.Client, prefix string) *AuthRateLimiter {
	if prefix == "" {
		prefix = "invmonitor"
	}
	return &AuthRateLimiter{
		rdb:         rdb,
		prefix:      prefix,
		maxRequests: defaultAuthRateLimitMaxRequests,
		window:      defaultAuthRateLimitWindow,
		memory:      make(map[string]memoryRateLimitCounter),
	}
}

func (l *AuthRateLimiter) Limit(action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, err := l.key(r, action)
			if err != nil {
				response.Error(w, r, errs.NewInternalServerError(err), nil)
				return
			}

			allowed, retryAfter := l.allow(r.Context(), key)
			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
				response.Error(w, r, errs.NewTooManyRequests("RATE_LIMIT_EXCEEDED"), nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (l *AuthRateLimiter) allow(ctx context.Context, key string) (bool, time.Duration) {
	if l.rdb != nil {
		allowed, retryAfter, err := l.allowRedis(ctx, key)
		if err == nil {
			return allowed, retryAfter
		}
	}
	return l.allowMemory(key, time.Now())
}

func (l *AuthRateLimiter) allowRedis(ctx context.Context, key string) (bool, time.Duration, error) {
	count, err := l.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}
	if count == 1 {
		if err := l.rdb.Expire(ctx, key, l.window).Err(); err != nil {
			return false, 0, err
		}
	}

	if count <= int64(l.maxRequests) {
		return true, 0, nil
	}

	ttl, err := l.rdb.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		ttl = l.window
	}
	return false, ttl, err
}

func (l *AuthRateLimiter) allowMemory(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	counter, ok := l.memory[key]
	if !ok || now.After(counter.ExpiresAt) {
		l.memory[key] = memoryRateLimitCounter{
			Count:     1,
			ExpiresAt: now.Add(l.window),
		}
		return true, 0
	}

	counter.Count++
	l.memory[key] = counter
	if counter.Count <= l.maxRequests {
		return true, 0
	}
	return false, time.Until(counter.ExpiresAt)
}

func (l *AuthRateLimiter) key(r *http.Request, action string) (string, error) {
	email, err := emailFromBody(r)
	if err != nil {
		return "", err
	}

	identity := clientIPForRateLimit(r)
	if email != "" {
		identity = identity + ":" + strings.ToLower(email)
	}

	hash := sha256.Sum256([]byte(identity))
	return fmt.Sprintf("%s:rate-limit:auth:%s:%s", l.prefix, action, hex.EncodeToString(hash[:])), nil
}

func emailFromBody(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", nil
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxRateLimitBodyBytes))
	if err != nil {
		return "", err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	var payload struct {
		Email string `json:"email"`
	}
	if len(body) == 0 {
		return "", nil
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil
	}
	return payload.Email, nil
}

func clientIPForRateLimit(r *http.Request) string {
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return cfIP
	}
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
