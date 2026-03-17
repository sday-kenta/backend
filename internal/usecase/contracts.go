package usecase

import (
	"context"

	"github.com/sday-kenta/backend/internal/entity"
)

//go:generate mockgen -source=contracts.go -destination=./mocks_usecase_test.go -package=usecase_test

type (
	Category interface {
		GetAll(ctx context.Context) ([]entity.Category, error)
		GetByID(ctx context.Context, id int) (entity.Category, error)
		Create(ctx context.Context, input entity.CreateCategoryInput) (entity.Category, error)
		Update(ctx context.Context, id int, input entity.UpdateCategoryInput) (entity.Category, error)
		Delete(ctx context.Context, id int) error
	}

	Geo interface {
		ReverseGeocode(ctx context.Context, lat, lon float64) (entity.Address, error)
		Search(ctx context.Context, query, city string) ([]entity.Address, error)
		ReloadCities(ctx context.Context) error
	}

	User interface {
		Create(context.Context, entity.User, string) (entity.User, error)
		Delete(context.Context, int64) error
		GetByID(context.Context, int64) (entity.User, error)
		Authenticate(context.Context, string, string) (entity.User, error)
		SendEmailVerificationCode(context.Context, string, string) error
		VerifyEmailVerificationCode(context.Context, string, string, string) error
		List(context.Context) ([]entity.User, error)
		Update(context.Context, entity.User) (entity.User, error)
		UpdateAvatar(context.Context, int64, string) error
	}

	Incident interface {
		Create(context.Context, int64, entity.CreateIncidentInput) (entity.Incident, error)
		List(context.Context, entity.IncidentFilter) ([]entity.Incident, error)
		GetByID(context.Context, int64) (entity.Incident, error)
		Update(context.Context, int64, bool, int64, entity.UpdateIncidentInput) (entity.Incident, error)
		Delete(context.Context, int64, bool, int64) ([]entity.IncidentPhoto, error)
		CreatePhoto(context.Context, int64, bool, int64, entity.IncidentPhoto) (entity.IncidentPhoto, error)
		DeletePhoto(context.Context, int64, bool, int64, int64) (entity.IncidentPhoto, error)
		RenderDocument(context.Context, int64, bool, int64) (entity.IncidentDocument, error)
		SendDocumentByEmail(context.Context, int64, bool, int64, string) error
	}
)
