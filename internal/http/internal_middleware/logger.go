// internal/http/internal_middleware/logger.go
package internal_middleware

import (
	"net/http"
	"time"

	chiMw "github.com/go-chi/chi/v5/middleware"

	"investment_monitoring/pkg/logger"
)

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1) Request ID (pakai chi middleware RequestID atau generate sendiri)
		reqID := chiMw.GetReqID(r.Context())
		if reqID != "" {
			w.Header().Set("X-Request-ID", reqID)
		}

		// 2) Wrap response writer untuk ambil status/bytes
		ww := chiMw.NewWrapResponseWriter(w, r.ProtoMajor)

		start := time.Now()

		// 3) Buat request-scoped logger
		l := logger.L().With(
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
		)

		// 4) inject ke context
		ctx := logger.With(r.Context(), l)

		// 5) jalankan handler
		next.ServeHTTP(ww, r.WithContext(ctx))

		// 6) access log (selesai)
		l.Info("request completed",
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"latency_ms", time.Since(start).Milliseconds(),
		)
	})
}
