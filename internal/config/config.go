package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	PostgresURL string `env:"POSTGRES_URL" envDefault:"postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"`
	ValkeyURL   string `env:"VALKEY_URL" envDefault:"redis://localhost:6379/0"`
	Port        int    `env:"PORT" envDefault:"8080"`
	Environment string `env:"APP_ENV" envDefault:"dev"`
}

func Load() (Config, error) {
	_ = godotenv.Load()

	return env.ParseAs[Config]()
}
