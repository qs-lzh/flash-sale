package config

import (
	"os"

	"github.com/qs-lzh/flash-sale/internal/util"
)

type Config struct {
	DatabaseDSN string
	Addr        string
	CacheURL    string
	MQURL       string
}

func LoadConfig() (*Config, error) {
	if err := util.LoadEnv(); err != nil {
		return nil, err
	}
	databaseDSN := os.Getenv("DATABASE_DSN")
	addr := os.Getenv("ADDR")
	cacheURL := os.Getenv("CACHE_URL")
	mqURL := os.Getenv("RABBIT_MQ_URL")
	return &Config{
		DatabaseDSN: databaseDSN,
		Addr:        addr,
		CacheURL:    cacheURL,
		MQURL:       mqURL,
	}, nil
}
