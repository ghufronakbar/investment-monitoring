package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareInjectsRequestMetadata(t *testing.T) {
	var got RequestMetadata
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = MetadataFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/audit", nil)
	req.RemoteAddr = "10.0.0.1:3210"
	req.Header.Set("User-Agent", "go-test")

	rr := httptest.NewRecorder()
	Middleware(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if got.RequestID != "" {
		t.Fatalf("request id = %q, want empty when request id middleware is absent", got.RequestID)
	}
	if got.IPAddress != "10.0.0.1" {
		t.Fatalf("ip address = %q, want 10.0.0.1", got.IPAddress)
	}
	if got.UserAgent != "go-test" {
		t.Fatalf("user agent = %q, want go-test", got.UserAgent)
	}
}

func TestClientIPPrefersForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1111"
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.2")

	if got := clientIP(req); got != "203.0.113.10" {
		t.Fatalf("client ip = %q, want first forwarded ip", got)
	}

	req.Header.Del("X-Forwarded-For")
	req.Header.Set("X-Real-IP", "203.0.113.20")
	if got := clientIP(req); got != "203.0.113.20" {
		t.Fatalf("client ip = %q, want x-real-ip", got)
	}
}
