// Package repo implements application outer layer logic. Each logic group in own file.
package repo

import (
	"context"

	"github.com/evrone/go-clean-template/internal/entity"
)

//go:generate mockgen -source=contracts.go -destination=../usecase/mocks_repo_test.go -package=usecase_test

type (
	// TranslationRepo -.
	TranslationRepo interface {
		Store(context.Context, entity.Translation) error
		GetHistory(context.Context) ([]entity.Translation, error)
	}

	// TranslationWebAPI -.
	TranslationWebAPI interface {
		Translate(entity.Translation) (entity.Translation, error)
	}

	// UserRepo describes persistence operations for users.
	UserRepo interface {
		Create(ctx context.Context, user *entity.User) error
		Delete(ctx context.Context, id int64) error
		GetByID(ctx context.Context, id int64) (entity.User, error)
		List(ctx context.Context) ([]entity.User, error)
		Update(ctx context.Context, user *entity.User) error
		UpdateAvatar(ctx context.Context, id int64, avatar []byte) error
	}
)
