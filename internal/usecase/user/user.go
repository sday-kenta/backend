package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/repo"
	"github.com/sday-kenta/backend/internal/usererr"
	"github.com/sday-kenta/backend/pkg/mailsender"
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

func (uc *UseCase) SendEmailVerificationCode(ctx context.Context, email, purpose string) error {
	email = strings.TrimSpace(email)
	purpose = strings.TrimSpace(purpose)

	// 10 minutes TTL
	const ttl = 10 * time.Minute
	code := mailsender.RandomRumber().String()

	if err := uc.repo.CreateEmailVerificationCode(
		ctx,
		email,
		purpose,
		code,
		time.Now().Add(ttl).Unix(),
	); err != nil {
		return err
	}

	subject := "Email verification code"
	body := "Your code is " + code
	if err := mailsender.SendMail(subject, body, []string{email}); err != nil {
		return fmt.Errorf("UserUseCase - SendEmailVerificationCode - SendMail: %w", err)
	}

	return nil
}

func (uc *UseCase) VerifyEmailVerificationCode(ctx context.Context, email, purpose, code string) error {
	email = strings.TrimSpace(email)
	purpose = strings.TrimSpace(purpose)
	code = strings.TrimSpace(code)

	if err := uc.repo.ConsumeEmailVerificationCode(ctx, email, purpose, code, time.Now().Unix()); err != nil {
		return err
	}

	return nil
}

func (uc *UseCase) SendPasswordResetCode(ctx context.Context, email string) error {
	email = strings.TrimSpace(email)

	const ttl = 10 * time.Minute
	code := mailsender.RandomRumber().String()

	if err := uc.repo.CreateEmailVerificationCode(
		ctx,
		email,
		"password_reset",
		code,
		time.Now().Add(ttl).Unix(),
	); err != nil {
		return err
	}

	subject := "Password reset code"
	body := "Your code is " + code
	if err := mailsender.SendMail(subject, body, []string{email}); err != nil {
		return fmt.Errorf("UserUseCase - SendPasswordResetCode - SendMail: %w", err)
	}

	return nil
}

func (uc *UseCase) VerifyPasswordResetCode(ctx context.Context, email, code string) error {
	email = strings.TrimSpace(email)
	code = strings.TrimSpace(code)

	if err := uc.repo.CheckEmailVerificationCode(ctx, email, "password_reset", code, time.Now().Unix()); err != nil {
		return err
	}

	return nil
}

func (uc *UseCase) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	email = strings.TrimSpace(email)
	code = strings.TrimSpace(code)

	// One-time: consume code, then update password.
	if err := uc.repo.ConsumeEmailVerificationCode(ctx, email, "password_reset", code, time.Now().Unix()); err != nil {
		return err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("UserUseCase - ResetPassword - bcrypt.GenerateFromPassword: %w", err)
	}

	if err = uc.repo.UpdatePasswordHashByEmail(ctx, email, string(hashed)); err != nil {
		return err
	}

	return nil
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


