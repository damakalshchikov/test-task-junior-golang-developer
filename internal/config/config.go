package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env  string `env:"ENV" env-default:"local"`
	HTTP HTTPConfig
	DB   DBConfig
}

type HTTPConfig struct {
	Port            string        `env:"HTTP_PORT" env-default:"8080"`
	ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"5s"`
	WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"10s"`
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"10s"`
}

type DBConfig struct {
	Host           string `env:"DB_HOST" env-default:"localhost"`
	Port           string `env:"DB_PORT" env-default:"5432"`
	User           string `env:"DB_USER" env-default:"postgres"`
	Password       string `env:"DB_PASSWORD" env-required:"true"`
	Name           string `env:"DB_NAME" env-default:"subscriptions"`
	SSLMode        string `env:"DB_SSLMODE" env-default:"disable"`
	MigrationsPath string `env:"MIGRATIONS_PATH" env-default:"migrations"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(".env", &cfg); err != nil {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	return &cfg, nil
}
