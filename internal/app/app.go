package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sday-kenta/backend/config"
	"github.com/sday-kenta/backend/internal/controller/restapi"
	"github.com/sday-kenta/backend/internal/repo/persistent"
	"github.com/sday-kenta/backend/internal/repo/webapi"
	"github.com/sday-kenta/backend/internal/usecase/category"
	geoUseCase "github.com/sday-kenta/backend/internal/usecase/geo"
	incidentUseCase "github.com/sday-kenta/backend/internal/usecase/incident"
	pushUseCase "github.com/sday-kenta/backend/internal/usecase/push"
	userUseCase "github.com/sday-kenta/backend/internal/usecase/user"
	"github.com/sday-kenta/backend/pkg/httpserver"
	"github.com/sday-kenta/backend/pkg/logger"
	"github.com/sday-kenta/backend/pkg/postgres"
	"github.com/sday-kenta/backend/pkg/pushclient"
)

func Run(cfg *config.Config) { //nolint: gocyclo,cyclop,funlen,gocritic,nolintlint
	l := logger.New(cfg.Log.Level, cfg.Log.Pretty)

	pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.PoolMax))
	if err != nil {
		l.Fatal(fmt.Errorf("app - Run - postgres.New: %w", err))
	}
	defer pg.Close()

	categoryUC := category.New(persistent.NewCategoryRepo(pg))
	geoRepo := persistent.NewGeoRepo(pg, cfg.Geo.CacheRadiusMeters)
	nominatimRepo := webapi.NewNominatimRepo(webapi.Config{
		BaseURL:        cfg.Nominatim.BaseURL,
		UserAgent:      cfg.Nominatim.UserAgent,
		Email:          cfg.Nominatim.Email,
		AcceptLanguage: cfg.Nominatim.AcceptLanguage,
		CountryCodes:   cfg.Nominatim.CountryCodes,
		SearchLimit:    cfg.Nominatim.SearchLimit,
		ReverseZoom:    cfg.Nominatim.ReverseZoom,
		Timeout:        cfg.Nominatim.Timeout,
	})
	geoUC := geoUseCase.New(geoRepo, nominatimRepo, cfg.Geo.MaxCityAttempts)
	if err = geoUC.ReloadCities(context.Background()); err != nil {
		l.Fatal(fmt.Errorf("app - Run - geoUC.ReloadCities: %w", err))
	}

	userRepo := persistent.NewUserRepo(pg)
	userUC := userUseCase.New(userRepo)
	if _, err = userUC.EnsureBootstrapAdmin(context.Background(), cfg.Admin); err != nil {
		l.Fatal(fmt.Errorf("app - Run - userUC.EnsureBootstrapAdmin: %w", err))
	}
	if cfg.Admin.Enabled {
		l.Info("app - Run - bootstrap admin ensured for %s", cfg.Admin.Email)
	}
	pushSender := pushclient.NewNoopSender()
	if cfg.FCM.Enabled {
		pushSender, err = pushclient.NewFCMSender(context.Background(), pushclient.Config{
			CredentialsFile: cfg.FCM.CredentialsFile,
			Timeout:         cfg.FCM.Timeout,
		})
		if err != nil {
			l.Fatal(fmt.Errorf("app - Run - pushclient.NewFCMSender: %w", err))
		}
	}
	pushUC := pushUseCase.New(persistent.NewPushDeviceRepo(pg), pushSender)
	incidentUC := incidentUseCase.New(persistent.NewIncidentRepo(pg), userRepo, persistent.NewCategoryRepo(pg), geoRepo)

	httpServer := httpserver.New(l, httpserver.Port(cfg.HTTP.Port), httpserver.Prefork(cfg.HTTP.UsePreforkMode))
	restapi.NewRouter(httpServer.App, cfg, categoryUC, geoUC, userUC, incidentUC, pushUC, l)

	httpServer.Start()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		l.Info("app - Run - signal: %s", s.String())
	case err = <-httpServer.Notify():
		l.Error(fmt.Errorf("app - Run - httpServer.Notify: %w", err))
	}

	if err = httpServer.Shutdown(); err != nil {
		l.Error(fmt.Errorf("app - Run - httpServer.Shutdown: %w", err))
	}
}
