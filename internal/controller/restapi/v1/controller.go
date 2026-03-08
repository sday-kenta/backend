package v1

import (
	"github.com/evrone/go-clean-template/internal/usecase"
	"github.com/evrone/go-clean-template/pkg/logger"
	"github.com/go-playground/validator/v10"
)

// V1 groups REST handlers for the first API version.
type V1 struct {
	c usecase.Category
	g usecase.Geo
	l logger.Interface
	v *validator.Validate
}
