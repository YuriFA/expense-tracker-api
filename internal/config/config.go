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
	HTTPServer  `       yaml:"http_server"`
}

type HTTPServer struct {
	Address       string        `yaml:"address"       env-default:"localhost:8080"`
	ReadTimeout   time.Duration `yaml:"read_timeout"  env-default:"5s"`
	WriteTimeout  time.Duration `yaml:"write_timeout" env-default:"30s"`
	IdleTimeout   time.Duration `yaml:"idle_timeout"  env-default:"60s"`
	CorsConfig    CORSConfig    `yaml:"cors"`
	SessionConfig SessionConfig `yaml:"session"`
}

type SessionConfig struct {
	TTL        time.Duration `yaml:"ttl"         env:"SESSION_TTL"         env-default:"24h"`
	CookieName string        `yaml:"cookie_name" env:"SESSION_COOKIE_NAME" env-default:"session_id"`
	Secure     bool          `yaml:"secure"      env:"SESSION_SECURE"      env-default:"true"`
	SameSite   string        `yaml:"same_site"   env:"SESSION_SAME_SITE"   env-default:"Lax"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS" env-default:"*"                           env-separator:","`
	AllowedMethods []string `yaml:"allowed_methods" env:"CORS_ALLOWED_METHODS" env-default:"GET,POST,PUT,DELETE,OPTIONS" env-separator:","`
	AllowedHeaders []string `yaml:"allowed_headers" env:"CORS_ALLOWED_HEADERS" env-default:"Content-Type,Authorization"  env-separator:","`
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
