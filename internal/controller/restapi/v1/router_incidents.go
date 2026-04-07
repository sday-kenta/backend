package v1

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	authmw "github.com/sday-kenta/backend/internal/controller/restapi/middleware"
	"github.com/sday-kenta/backend/internal/usecase"
	"github.com/sday-kenta/backend/pkg/logger"
)

func NewIncidentRoutes(apiV1Group fiber.Router, i usecase.Incident, l logger.Interface, mediaBaseURL string) {
	r := &IncidentsV1{
		i:            i,
		l:            l,
		v:            validator.New(validator.WithRequiredStructEnabled()),
		mediaBaseURL: mediaBaseURL,
	}

	incidentsGroup := apiV1Group.Group("/incidents")
	{
		incidentsGroup.Post("", authmw.RequireAuth(), r.createIncident)
		incidentsGroup.Get("", r.listIncidents)
		incidentsGroup.Get("/:id", r.getIncident)
		incidentsGroup.Patch("/:id", authmw.RequireAuth(), r.updateIncident)
		incidentsGroup.Delete("/:id", authmw.RequireAuth(), r.deleteIncident)
		incidentsGroup.Post("/:id/photos", authmw.RequireAuth(), r.uploadIncidentPhotos)
		incidentsGroup.Delete("/:id/photos/:photoId", authmw.RequireAuth(), r.deleteIncidentPhoto)
		incidentsGroup.Get("/:id/document/download", authmw.RequireAuth(), r.downloadIncidentDocument)
		incidentsGroup.Get("/:id/document/print", authmw.RequireAuth(), r.printIncidentDocument)
		incidentsGroup.Post("/:id/document/email", authmw.RequireAuth(), r.emailIncidentDocument)
	}

	myGroup := apiV1Group.Group("/my")
	{
		myGroup.Get("/incidents", authmw.RequireAuth(), r.listMyIncidents)
	}
}
