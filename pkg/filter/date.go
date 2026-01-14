// pkg/filter/date.go
package filter

import (
	"strings"
	"time"
)

// ParseDatePtr menerima string date (YYYY-MM-DD).
// Return nil kalau empty / invalid.
func ParseDatePtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// Validasi ketat format date
	// time.Parse akan error jika ada extra char, mis. "2025-12-20sss"
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return nil
	}

	// pointer yang stabil
	v := s
	return &v
}
