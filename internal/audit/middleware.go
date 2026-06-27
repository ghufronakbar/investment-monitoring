package audit

import (
	"context"
	"net"
	"net/http"
	"strings"

	chiMw "github.com/go-chi/chi/v5/middleware"
)

type requestMetadataContextKey struct{}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metadata := RequestMetadata{
			RequestID: chiMw.GetReqID(r.Context()),
			IPAddress: clientIP(r),
			UserAgent: r.UserAgent(),
		}

		ctx := context.WithValue(r.Context(), requestMetadataContextKey{}, metadata)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func MetadataFromContext(ctx context.Context) RequestMetadata {
	metadata, ok := ctx.Value(requestMetadataContextKey{}).(RequestMetadata)
	if !ok {
		return RequestMetadata{}
	}
	return metadata
}

func clientIP(r *http.Request) string {
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
