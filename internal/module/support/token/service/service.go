// internal/module/headless/token/token_service/service.go
package token_service

import (
	"investment_monitoring/config"
	"time"
)

type TokenMaker struct {
	// ACCESS
	accessSecret []byte
	accessTTL    time.Duration
	// REFRESH
	refreshSecret []byte
	refreshTTL    time.Duration
}

func NewService(cfg *config.Config) *TokenMaker {
	return &TokenMaker{
		accessSecret:  []byte(cfg.JWT_ACCESS_TOKEN_SECRET),
		refreshSecret: []byte(cfg.JWT_REFRESH_TOKEN_SECRET),
		accessTTL:     cfg.JWT_ACCESS_TOKEN_EXPIRED,
		refreshTTL:    cfg.JWT_REFRESH_TOKEN_EXPIRED,
	}
}
