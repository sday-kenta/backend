// backend/cmd/app/main.go

// @title       Сознательный гражданин API
// @description API для мобильного приложения и админ-панели проекта "ЭкоВыбор"
// @version     1.0
// @host        localhost:8080
// @BasePath    /v1
package main

import (
	"log"

	"github.com/evrone/go-clean-template/config"
	"github.com/evrone/go-clean-template/internal/app"
)

func main() {
	// Configuration
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Run
	app.Run(cfg)
}
