package internal_middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthRateLimiterKeyUsesLowercaseEmailAndRestoresBody(t *testing.T) {
	limiter := NewAuthRateLimiter(nil, "test")
	req := httptest.NewRequest(http.MethodPost, "/iam/auth/login", strings.NewReader(`{"email":"USER@Example.COM"}`))
	req.RemoteAddr = "198.51.100.1:1234"

	keyA, err := limiter.key(req, "login")
	if err != nil {
		t.Fatalf("key returned error: %v", err)
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	if !strings.Contains(string(body), "USER@Example.COM") {
		t.Fatalf("restored body = %s, want original body", string(body))
	}

	req = httptest.NewRequest(http.MethodPost, "/iam/auth/login", strings.NewReader(`{"email":"user@example.com"}`))
	req.RemoteAddr = "198.51.100.1:1234"
	keyB, err := limiter.key(req, "login")
	if err != nil {
		t.Fatalf("key returned error: %v", err)
	}
	if keyA != keyB {
		t.Fatalf("keys differ for email case: %s vs %s", keyA, keyB)
	}
}

func TestAuthRateLimiterMemoryWindowResets(t *testing.T) {
	limiter := NewAuthRateLimiter(nil, "test")
	limiter.maxRequests = 1
	limiter.window = time.Minute
	key := "key"
	now := time.Now()

	if ok, _ := limiter.allowMemory(key, now); !ok {
		t.Fatal("first request should be allowed")
	}
	if ok, _ := limiter.allowMemory(key, now.Add(time.Second)); ok {
		t.Fatal("second request inside window should be rejected")
	}
	if ok, _ := limiter.allowMemory(key, now.Add(2*time.Minute)); !ok {
		t.Fatal("request after window should be allowed")
	}
}

func TestClientIPForRateLimitPriority(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("CF-Connecting-IP", "203.0.113.1")
	req.Header.Set("X-Forwarded-For", "203.0.113.2, 10.0.0.2")
	req.Header.Set("X-Real-IP", "203.0.113.3")

	if got := clientIPForRateLimit(req); got != "203.0.113.1" {
		t.Fatalf("ip = %s, want CF-Connecting-IP", got)
	}
}
