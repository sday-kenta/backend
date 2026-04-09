package config

import (
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type (
	Config struct {
		App       App
		Admin     AdminBootstrap
		Auth      Auth
		FCM       FCM
		HTTP      HTTP
		Log       Log
		PG        PG
		Metrics   Metrics
		Swagger   Swagger
		CDN       CDN
		Geo       Geo
		Nominatim Nominatim
	}

	App struct {
		Name    string `env:"APP_NAME,required"`
		Version string `env:"APP_VERSION,required"`
	}

	AdminBootstrap struct {
		Enabled    bool   `env:"ADMIN_BOOTSTRAP_ENABLED" envDefault:"false"`
		Login      string `env:"ADMIN_BOOTSTRAP_LOGIN" envDefault:"admin"`
		Email      string `env:"ADMIN_BOOTSTRAP_EMAIL" envDefault:""`
		Password   string `env:"ADMIN_BOOTSTRAP_PASSWORD" envDefault:""`
		Phone      string `env:"ADMIN_BOOTSTRAP_PHONE" envDefault:""`
		LastName   string `env:"ADMIN_BOOTSTRAP_LAST_NAME" envDefault:"Bootstrap"`
		FirstName  string `env:"ADMIN_BOOTSTRAP_FIRST_NAME" envDefault:"Admin"`
		MiddleName string `env:"ADMIN_BOOTSTRAP_MIDDLE_NAME" envDefault:""`
		City       string `env:"ADMIN_BOOTSTRAP_CITY" envDefault:"N/A"`
		Street     string `env:"ADMIN_BOOTSTRAP_STREET" envDefault:"N/A"`
		House      string `env:"ADMIN_BOOTSTRAP_HOUSE" envDefault:"N/A"`
		Apartment  string `env:"ADMIN_BOOTSTRAP_APARTMENT" envDefault:""`
	}

	Auth struct {
		JWTSecret string        `env:"JWT_SECRET" envDefault:"dev-secret-change-me"`
		JWTTTL    time.Duration `env:"JWT_TTL" envDefault:"24h"`
		JWTIssuer string        `env:"JWT_ISSUER" envDefault:"backend"`
	}

	FCM struct {
		Enabled         bool          `env:"FCM_ENABLED" envDefault:"false"`
		CredentialsFile string        `env:"FCM_CREDENTIALS_FILE" envDefault:"./firebase-service-account.json"`
		Timeout         time.Duration `env:"FCM_TIMEOUT" envDefault:"5s"`
	}

	HTTP struct {
		Port           string `env:"HTTP_PORT,required"`
		UsePreforkMode bool   `env:"HTTP_USE_PREFORK_MODE" envDefault:"false"`
	}

	Log struct {
		Level               string `env:"LOG_LEVEL,required"`
		Pretty              bool   `env:"LOG_PRETTY" envDefault:"false"`
		HTTPLogHeaders      bool   `env:"HTTP_LOG_HEADERS" envDefault:"false"`
		HTTPLogBody         bool   `env:"HTTP_LOG_BODY" envDefault:"false"`
		HTTPLogBodyMaxBytes int    `env:"HTTP_LOG_BODY_MAX_BYTES" envDefault:"4096"`
	}

	PG struct {
		PoolMax int    `env:"PG_POOL_MAX,required"`
		URL     string `env:"PG_URL,required"`
	}

	Metrics struct {
		Enabled bool `env:"METRICS_ENABLED" envDefault:"true"`
	}

	Swagger struct {
		Enabled  bool   `env:"SWAGGER_ENABLED" envDefault:"false"`
		Host     string `env:"SWAGGER_HOST" envDefault:""`
		BasePath string `env:"SWAGGER_BASE_PATH" envDefault:"/v1"`
		Schemes  string `env:"SWAGGER_SCHEMES" envDefault:""`
	}

	CDN struct {
		AvatarBaseURL        string `env:"AVATAR_BASE_URL" envDefault:""`
		IncidentMediaBaseURL string `env:"INCIDENT_MEDIA_BASE_URL" envDefault:""`
		CategoryMediaBaseURL string `env:"CATEGORY_MEDIA_BASE_URL" envDefault:""`
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
)

func NewConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}
	return cfg, nil
}
