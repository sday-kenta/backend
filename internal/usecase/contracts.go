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
)
