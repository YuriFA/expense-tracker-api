package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string `yaml:"env"          env:"ENV"          env-required:"true"`
	StoragePath string `yaml:"storage_path" env:"STORAGE_PATH" env-required:"true"`
	HTTPServer `       yaml:"http_server"`
}

type HTTPServer struct {
	Address      string        `yaml:"address"       env-default:"localhost:8080"`
	ReadTimeout  time.Duration `yaml:"read_timeout"  env-default:"5s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env-default:"30s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"  env-default:"60s"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")

	if configPath == "" {
		log.Fatal("CONFIG_PATH environment variable is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("Config file does not exist at path: %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	return &cfg
}
