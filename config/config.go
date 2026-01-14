// config/config.go
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Config struct {
	// COMMON
	MODE string
	PORT string

	APP_NAME          string
	APP_LOGO          string
	APP_ADDRESS       string
	APP_CONTACT_EMAIL string

	// DOMAIN
	API_URL string
	WEB_URL string

	// DATABASE
	DATABASE_URL string

	// REDIS
	REDIS_HOST string
	REDIS_PORT string
	REDIS_PASS string
	REDIS_DB   int

	// JWT
	JWT_ACCESS_TOKEN_SECRET  string
	JWT_REFRESH_TOKEN_SECRET string

	// TIME
	JWT_ACCESS_TOKEN_EXPIRED  time.Duration // minutes
	JWT_REFRESH_TOKEN_EXPIRED time.Duration // days
	JWT_REFRESH_TOKEN_RENEWAL time.Duration // days

	// SMTP
	SMTP_HOST        string
	SMTP_PORT        int
	SMTP_SECURE      bool
	SMTP_USER        string
	SMTP_PASS        string
	SMTP_NAME        string
	SMTP_SERVER_NAME string
	SMTP_SENDER      string

	// CLOUDINARY
	CLOUDINARY_CLOUD_NAME string
	CLOUDINARY_API_KEY    string
	CLOUDINARY_API_SECRET string
}

func Load() *Config {
	// Load .env, abaikan error jika file tidak ada (misal di production pakai env asli)
	_ = godotenv.Load()

	smtpPort := getEnv("SMTP_PORT")
	numSmtpPort, err := strconv.Atoi(smtpPort)
	if err != nil {
		panic("ENV SMTP_PORT is required number")
	}

	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB"))

	jwtAccessTokenExpired, _ := strconv.Atoi(getEnv("JWT_ACCESS_TOKEN_EXPIRED"))
	jwtRefreshTokenExpired, _ := strconv.Atoi(getEnv("JWT_REFRESH_TOKEN_EXPIRED"))
	jwtRefreshTokenRenewal, _ := strconv.Atoi(getEnv("JWT_REFRESH_TOKEN_RENEWAL"))

	jwtAccessTokenExpiredDuration := time.Duration(jwtAccessTokenExpired) * time.Minute
	jwtRefreshTokenExpiredDuration := time.Duration(jwtRefreshTokenExpired) * time.Hour * 24
	jwtRefreshTokenRenewalDuration := time.Duration(jwtRefreshTokenRenewal) * time.Hour * 24

	return &Config{
		// COMMON
		MODE:              getEnv("MODE"),
		PORT:              getEnv("PORT"),
		APP_NAME:          getEnv("APP_NAME"),
		APP_LOGO:          getEnv("APP_LOGO"),
		APP_ADDRESS:       getEnv("APP_ADDRESS"),
		APP_CONTACT_EMAIL: getEnv("APP_CONTACT_EMAIL"),

		// DOMAIN
		API_URL: getEnv("API_URL"),
		WEB_URL: getEnv("WEB_URL"),

		// DATABASE
		DATABASE_URL: getEnv("DATABASE_URL"),

		// REDIS
		REDIS_HOST: getEnv("REDIS_HOST"),
		REDIS_PORT: getEnv("REDIS_PORT"),
		REDIS_PASS: getEnv("REDIS_PASS"),
		REDIS_DB:   redisDB,

		// JWT
		JWT_ACCESS_TOKEN_SECRET:  getEnv("JWT_ACCESS_TOKEN_SECRET"),
		JWT_REFRESH_TOKEN_SECRET: getEnv("JWT_REFRESH_TOKEN_SECRET"),

		// TIME
		JWT_ACCESS_TOKEN_EXPIRED:  jwtAccessTokenExpiredDuration,
		JWT_REFRESH_TOKEN_EXPIRED: jwtRefreshTokenExpiredDuration,
		JWT_REFRESH_TOKEN_RENEWAL: jwtRefreshTokenRenewalDuration,

		// SMTP
		SMTP_HOST:        getEnv("SMTP_HOST"),
		SMTP_PORT:        numSmtpPort,
		SMTP_SECURE:      numSmtpPort == 465 || numSmtpPort == 587,
		SMTP_USER:        getEnv("SMTP_USER"),
		SMTP_PASS:        getEnv("SMTP_PASS"),
		SMTP_NAME:        getEnv("SMTP_NAME"),
		SMTP_SERVER_NAME: getEnv("SMTP_SERVER_NAME"),
		SMTP_SENDER:      getEnv("SMTP_SENDER"),

		// CLOUDINARY
		CLOUDINARY_CLOUD_NAME: getEnv("CLOUDINARY_CLOUD_NAME"),
		CLOUDINARY_API_KEY:    getEnv("CLOUDINARY_API_KEY"),
		CLOUDINARY_API_SECRET: getEnv("CLOUDINARY_API_SECRET"),
	}
}

func getEnv(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	panic("ENV " + key + " is required")
}
