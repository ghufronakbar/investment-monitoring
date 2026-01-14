// internal/http/internal_middleware/req_filter.go
package internal_middleware

import (
	"context"
	"investment_monitoring/pkg/filter"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

type contextReqFilterKey string

const ReqFilterContextKey contextReqFilterKey = "reqFilter"

var allowedSort = []string{"asc", "desc"}
var allowedSortByDefault = []string{"createdAt", "updatedAt", "name"}

func ReqFilterMiddleware(next http.Handler, allowedSortBy []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		search := strings.TrimSpace(q.Get("search"))
		sortBy := q.Get("sortBy")
		sort := q.Get("sort")
		category := q.Get("category")

		page, err := strconv.Atoi(q.Get("page"))
		if err != nil || page < 1 {
			page = 1
		}

		limit, err := strconv.Atoi(q.Get("limit"))
		if err != nil || limit < 1 {
			limit = 10
		}
		if limit > 50 {
			limit = 50
		}

		if !slices.Contains(allowedSort, sort) {
			sort = "desc"
		}

		allSortBy := append([]string{}, allowedSortByDefault...)
		allSortBy = append(allSortBy, allowedSortBy...)
		if !slices.Contains(allSortBy, sortBy) {
			sortBy = "id"
		}

		// ✅ DateStart/DateEnd: hanya set kalau valid (YYYY-MM-DD)
		dateStart := filter.ParseDatePtr(strings.TrimSpace(q.Get("dateStart")))
		dateEnd := filter.ParseDatePtr(strings.TrimSpace(q.Get("dateEnd")))

		if dateStart != nil && dateEnd != nil && *dateStart > *dateEnd {
			// swap atau null-kan keduanya
			// swap:
			dateStart, dateEnd = nil, nil
		}

		ctx := context.WithValue(r.Context(), ReqFilterContextKey, filter.ReqFilter{
			Search:    search,
			Page:      page,
			Limit:     limit,
			Sort:      sort,
			SortBy:    sortBy,
			Category:  category,
			DateStart: dateStart,
			DateEnd:   dateEnd,
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetFilterFromContext(ctx context.Context) filter.ReqFilter {
	fil, ok := ctx.Value(ReqFilterContextKey).(filter.ReqFilter)
	if !ok {
		return filter.ReqFilter{
			Search:    "",
			SortBy:    "id",
			Page:      1,
			Limit:     10,
			Sort:      "desc",
			Category:  "",
			DateStart: nil,
			DateEnd:   nil,
		}
	}
	return fil
}
