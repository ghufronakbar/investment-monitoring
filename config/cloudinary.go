// config/redis.go
package config

import (
	cld "github.com/cloudinary/cloudinary-go/v2"
)

func ConnectCloudinary(cfg *Config) (*cld.Cloudinary, error) {
	cloudinary, err := cld.NewFromParams(cfg.CLOUDINARY_CLOUD_NAME, cfg.CLOUDINARY_API_KEY, cfg.CLOUDINARY_API_SECRET)
	if err != nil {
		panic("CLOUDINARY_NOT_SET")
	}

	return cloudinary, nil
}
