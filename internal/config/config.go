package config

import (
	"flag"
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	GRPCAddr string `env:"GRPC_ADDR" env-default:":50051"`
	DSN      string `env:"DATABASE_DSN" env-default:"postgres://rates:rates@localhost:5432/rates?sslmode=disable"`

	Grinex GrinexConfig
}

// GrinexConfig holds exchange client settings.
type GrinexConfig struct {
	BaseURL string        `env:"GRINEX_BASE_URL" env-default:"https://grinex.io"`
	Market  string        `env:"GRINEX_MARKET" env-default:"usdta7a5"`
	Timeout time.Duration `env:"GRINEX_TIMEOUT" env-default:"5s"`
	UseStub bool          `env:"GRINEX_USE_STUB" env-default:"false"`
}

func Load(args []string) (Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}

	fs := flag.NewFlagSet("app", flag.ContinueOnError)
	fs.StringVar(&cfg.GRPCAddr, "port", cfg.GRPCAddr, "")
	fs.StringVar(&cfg.DSN, "dsn", cfg.DSN, "")
	fs.StringVar(&cfg.Grinex.BaseURL, "grinex-url", cfg.Grinex.BaseURL, "")
	fs.DurationVar(&cfg.Grinex.Timeout, "grinex-timeout", cfg.Grinex.Timeout, "")
	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}
