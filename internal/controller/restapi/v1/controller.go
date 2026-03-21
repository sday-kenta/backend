package v1

import (
	"github.com/go-playground/validator/v10"
	"github.com/sday-kenta/backend/internal/usecase"
	"github.com/sday-kenta/backend/pkg/logger"
)

// V1 groups REST handlers for the first API version.
type V1 struct {
	c                    usecase.Category
	g                    usecase.Geo
	l                    logger.Interface
	v                    *validator.Validate
	categoryMediaBaseURL string
}
