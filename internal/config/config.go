package config

import "os"

type Config struct {
	Database DatabaseConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func Load() Config {
	return Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "3307"),
			User:     getEnv("DB_USER", "trading"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "trading"),
		},
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
