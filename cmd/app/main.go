// @title       Backend API
// @description Combined users and maps service API
// @version     1.0
// @host        localhost:8080
// @BasePath    /v1
package main

import (
	"log"

	"github.com/sday-kenta/backend/config"
	"github.com/sday-kenta/backend/internal/app"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	app.Run(cfg)
}
