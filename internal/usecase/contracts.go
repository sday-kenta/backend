// Package usecase implements application business logic. Each logic group in own file.
package usecase

import (
	"context"

	"github.com/evrone/go-clean-template/internal/entity"
)

//go:generate mockgen -source=interfaces.go -destination=./mocks_usecase_test.go -package=usecase_test

type (
	Category interface {
		GetAll(ctx context.Context) ([]entity.Category, error)
		GetByID(ctx context.Context, id int) (entity.Category, error)
		Create(ctx context.Context, input entity.CreateCategoryInput) (entity.Category, error)
		Update(ctx context.Context, id int, input entity.UpdateCategoryInput) (entity.Category, error)
		Delete(ctx context.Context, id int) error
	}
	CategoryRepo interface {
		GetAll(ctx context.Context) ([]entity.Category, error)
		GetByID(ctx context.Context, id int) (entity.Category, error)
		Create(ctx context.Context, input entity.CreateCategoryInput) (int, error)
		Update(ctx context.Context, id int, input entity.UpdateCategoryInput) error
		Delete(ctx context.Context, id int) error
	}
)
