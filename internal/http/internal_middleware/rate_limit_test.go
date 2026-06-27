package internal_middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthRateLimiterMemoryFallbackLimitsRequests(t *testing.T) {
	limiter := NewAuthRateLimiter(nil, "test")
	hitCount := 0

	handler := limiter.Limit("login")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("handler could not read restored body: %v", err)
		}
		if payload.Email != "user@example.com" {
			t.Fatalf("expected restored email user@example.com, got %s", payload.Email)
		}
		hitCount++
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < defaultAuthRateLimitMaxRequests; i++ {
		req := httptest.NewRequest(http.MethodPost, "/iam/auth/login", strings.NewReader(`{"email":"user@example.com"}`))
		req.RemoteAddr = "192.0.2.10:1234"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d expected status 200, got %d", i+1, rr.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/iam/auth/login", strings.NewReader(`{"email":"user@example.com"}`))
	req.RemoteAddr = "192.0.2.10:1234"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429 after limit exceeded, got %d", rr.Code)
	}
	if hitCount != defaultAuthRateLimitMaxRequests {
		t.Fatalf("expected handler hit count %d, got %d", defaultAuthRateLimitMaxRequests, hitCount)
	}
}
