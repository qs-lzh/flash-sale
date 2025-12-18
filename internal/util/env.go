package util

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadEnv searches parent directories for a .env file up to the root and loads it if found.
func LoadEnv() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			return godotenv.Load(envPath)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil
}
