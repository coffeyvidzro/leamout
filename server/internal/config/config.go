package config

import (
	"strings"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type QueueConfig struct {
	Enabled    bool `env:"ENABLED" envDefault:"true"`
	MaxWorkers int  `env:"MAX_WORKERS" envDefault:"10"`
}

type CronConfig struct {
	Enabled  bool   `env:"ENABLED" envDefault:"true"`
	Timezone string `env:"TIMEZONE" envDefault:"Africa/Accra"`
}

type OAuthConfig struct {
	ClientID     string `env:"CLIENT_ID"`
	ClientSecret string `env:"CLIENT_SECRET"`
}

type ProviderConfig struct {
	BaseURL string `env:"BASE_URL"`
	APIKey  string `env:"API_KEY"`
}

type Config struct {
	AppEnv            string `env:"APP_ENV" envDefault:"development"`
	HTTPPort          string `env:"HTTP_PORT" envDefault:"8080"`
	APIBaseURL        string `env:"API_BASE_URL" envDefault:"http://localhost:8080"`
	FrontendBaseURL   string `env:"FRONTEND_BASE_URL" envDefault:"http://localhost:3000"`
	ShortBaseURL      string `env:"SHORT_BASE_URL" envDefault:"http://localhost:3000"`
	DatabaseURL       string `env:"DATABASE_URL" envDefault:"postgres://postgres:postgres@localhost:5432/leamout?sslmode=disable"`
	RedisURL          string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`
	ArcjetKey         string `env:"ARCJET_KEY,required"`
	IPInfoToken       string `env:"IPINFO_TOKEN"`
	GeoIPDatabasePath string `env:"GEOIP_DATABASE_PATH" envDefault:"./assets/GeoLite2-City.mmdb"`

	CORSOrigins    []string `env:"CORS_ORIGINS" envSeparator:"," envDefault:"http://localhost:3000,http://127.0.0.1:3000"`
	TrustedProxies []string `env:"TRUSTED_PROXIES" envSeparator:","`

	Queue QueueConfig `envPrefix:"QUEUE_"`
	Cron  CronConfig  `envPrefix:"CRON_"`

	Google OAuthConfig `envPrefix:"GOOGLE_"`
	Github OAuthConfig `envPrefix:"GITHUB_"`

	Arkesel ProviderConfig `envPrefix:"ARKESEL_"`
	PawaPay ProviderConfig `envPrefix:"PAWAPAY_"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return strings.EqualFold(c.AppEnv, "development")
}

func (c *Config) PaymentWebhookURL(provider string) string {
	return strings.TrimRight(c.APIBaseURL, "/") + "/webhooks/payments/" + provider
}
