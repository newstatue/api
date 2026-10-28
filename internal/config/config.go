package config

import (
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	PostgresURL   string        `env:"POSTGRES_URL,required,notEmpty"`
	ValkeyURL     string        `env:"VALKEY_URL,required,notEmpty"`
	Port          int           `env:"PORT" envDefault:"8080"`
	Environment   string        `env:"ENV" envDefault:"prod"`
	SessionSecure bool          `env:"SESSION_SECURE" envDefault:"true"`
	SessionTTL    time.Duration `env:"SESSION_TTL" envDefault:"168h"`
}

func Load() (Config, error) {
	_ = godotenv.Load()

	return env.ParseAs[Config]()
}
