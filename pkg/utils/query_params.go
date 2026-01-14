// pkg/utils/query_params.go
package utils

import (
	"investment_monitoring/pkg/errs"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

func GetQueryInt64(r *http.Request, key string) (int64, error) {
	val := r.URL.Query().Get(key)
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, errs.NewValidationFailed(map[string]string{key: "INVALID INT64"})
	}
	return i, nil
}

func GetQueryEnum(r *http.Request, key string, enumValues []string) (string, error) {
	val := r.URL.Query().Get(key)
	if !slices.Contains(enumValues, val) {
		return "", errs.NewValidationFailed(map[string]string{key: "INVALID ENUM", "enumValues": strings.Join(enumValues, ",")})
	}
	return val, nil
}
