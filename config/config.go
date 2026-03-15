package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type (
	Config struct {
		App       App
		HTTP      HTTP
		Log       Log
		PG        PG
		Geo       Geo
		Nominatim Nominatim
		Swagger   Swagger
	}

	App struct {
		Name    string `env:"APP_NAME,required"`
		Version string `env:"APP_VERSION,required"`
	}

	HTTP struct {
		Port           string `env:"HTTP_PORT,required"`
		UsePreforkMode bool   `env:"HTTP_USE_PREFORK_MODE" envDefault:"false"`
	}

	Log struct {
		Level string `env:"LOG_LEVEL,required"`
	}

	PG struct {
		PoolMax int    `env:"PG_POOL_MAX,required"`
		URL     string `env:"PG_URL,required"`
	}

	Geo struct {
		CacheRadiusMeters int `env:"GEO_CACHE_RADIUS_METERS" envDefault:"20"`
		MaxCityAttempts   int `env:"GEO_MAX_CITY_ATTEMPTS" envDefault:"4"`
	}

	Nominatim struct {
		BaseURL        string        `env:"NOMINATIM_BASE_URL" envDefault:"https://nominatim.openstreetmap.org"`
		UserAgent      string        `env:"NOMINATIM_USER_AGENT" envDefault:"sday-kenta/1.0"`
		Email          string        `env:"NOMINATIM_EMAIL"`
		AcceptLanguage string        `env:"NOMINATIM_ACCEPT_LANGUAGE" envDefault:"ru"`
		CountryCodes   string        `env:"NOMINATIM_COUNTRY_CODES" envDefault:"ru"`
		SearchLimit    int           `env:"NOMINATIM_SEARCH_LIMIT" envDefault:"5"`
		ReverseZoom    int           `env:"NOMINATIM_REVERSE_ZOOM" envDefault:"18"`
		Timeout        time.Duration `env:"NOMINATIM_TIMEOUT" envDefault:"5s"`
	}

	Swagger struct {
		Enabled bool `env:"SWAGGER_ENABLED" envDefault:"false"`
	}
)

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}
	return cfg, nil
}
