package v1

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
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
		incidentsGroup.Post("", r.createIncident)
		incidentsGroup.Get("", r.listIncidents)
		incidentsGroup.Get("/:id", r.getIncident)
		incidentsGroup.Patch("/:id", r.updateIncident)
		incidentsGroup.Delete("/:id", r.deleteIncident)
		incidentsGroup.Post("/:id/photos", r.uploadIncidentPhotos)
		incidentsGroup.Delete("/:id/photos/:photoId", r.deleteIncidentPhoto)
		incidentsGroup.Get("/:id/document/download", r.downloadIncidentDocument)
		incidentsGroup.Get("/:id/document/print", r.printIncidentDocument)
		incidentsGroup.Post("/:id/document/email", r.emailIncidentDocument)
	}

	myGroup := apiV1Group.Group("/my")
	{
		myGroup.Get("/incidents", r.listMyIncidents)
	}
}
