package user

import (
	"context"
	"fmt"

	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/repo"
	"github.com/sday-kenta/backend/internal/usererr"
	"golang.org/x/crypto/bcrypt"
)

// UseCase implements usecase.User.
type UseCase struct {
	repo repo.UserRepo
}

// New creates a new user use case.
func New(r repo.UserRepo) *UseCase {
	return &UseCase{
		repo: r,
	}
}

// Create creates a new user with hashed password.
func (uc *UseCase) Create(ctx context.Context, u entity.User, password string) (entity.User, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return entity.User{}, fmt.Errorf("UserUseCase - Create - bcrypt.GenerateFromPassword: %w", err)
	}

	u.PasswordHash = string(hashed)
	if u.Role == "" {
		u.Role = "user"
	}

	if err = uc.repo.Create(ctx, &u); err != nil {
		return entity.User{}, err
	}

	return u, nil
}

// Delete removes a user by ID.
func (uc *UseCase) Delete(ctx context.Context, id int64) error {
	if err := uc.repo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

// GetByID returns user by ID.
func (uc *UseCase) GetByID(ctx context.Context, id int64) (entity.User, error) {
	u, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return entity.User{}, err
	}

	return u, nil
}

// Authenticate checks identifier (login/email/phone) and password against DB.
func (uc *UseCase) Authenticate(ctx context.Context, identifier, password string) (entity.User, error) {
	u, err := uc.repo.GetByIdentifier(ctx, identifier)
	if err != nil {
		if err == usererr.ErrNotFound {
			return entity.User{}, usererr.ErrInvalidCredentials
		}
		return entity.User{}, err
	}

	if u.IsBlocked {
		return entity.User{}, usererr.ErrUserBlocked
	}

	if err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return entity.User{}, usererr.ErrInvalidCredentials
	}

	return u, nil
}

// List returns all users.
func (uc *UseCase) List(ctx context.Context) ([]entity.User, error) {
	users, err := uc.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("UserUseCase - List - uc.repo.List: %w", err)
	}

	return users, nil
}

// Update updates user fields (without changing password).
func (uc *UseCase) Update(ctx context.Context, u entity.User) (entity.User, error) {
	if err := uc.repo.Update(ctx, &u); err != nil {
		return entity.User{}, err
	}

	return u, nil
}

// UpdateAvatar updates the avatar identifier/URL for a user.
func (uc *UseCase) UpdateAvatar(ctx context.Context, id int64, avatarURL string) error {
	if err := uc.repo.UpdateAvatar(ctx, id, avatarURL); err != nil {
		return err
	}

	return nil
}


