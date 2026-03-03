// Package usecase implements application business logic. Each logic group in own file.
package usecase

import (
	"context"

	"github.com/evrone/go-clean-template/internal/entity"
)

//go:generate mockgen -source=interfaces.go -destination=./mocks_usecase_test.go -package=usecase_test

type (
	// Translation -.
	Translation interface {
		Translate(context.Context, entity.Translation) (entity.Translation, error)
		History(context.Context) (entity.TranslationHistory, error)
	}

	// User describes user-related business operations.
	User interface {
		Create(context.Context, entity.User, string) (entity.User, error)
		Delete(context.Context, int64) error
		GetByID(context.Context, int64) (entity.User, error)
		List(context.Context) ([]entity.User, error)
		Update(context.Context, entity.User) (entity.User, error)
		UpdateAvatar(context.Context, int64, []byte) error
	}
)
