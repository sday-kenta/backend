package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sday-kenta/backend/config"
	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/repo"
	"github.com/sday-kenta/backend/internal/usererr"
	"github.com/sday-kenta/backend/pkg/mailsender"
	"golang.org/x/crypto/bcrypt"
)

const (
	purposePasswordReset = "password_reset"
	purposeRegister      = "register"
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

// EnsureBootstrapAdmin creates the first admin from env or elevates an existing user.
func (uc *UseCase) EnsureBootstrapAdmin(ctx context.Context, cfg config.AdminBootstrap) (entity.User, error) {
	if !cfg.Enabled {
		return entity.User{}, nil
	}

	email := strings.TrimSpace(cfg.Email)
	password := strings.TrimSpace(cfg.Password)
	phone := strings.TrimSpace(cfg.Phone)
	if email == "" || password == "" || phone == "" {
		return entity.User{}, fmt.Errorf("admin bootstrap requires ADMIN_BOOTSTRAP_EMAIL, ADMIN_BOOTSTRAP_PASSWORD and ADMIN_BOOTSTRAP_PHONE")
	}

	login := strings.TrimSpace(cfg.Login)
	if login == "" {
		login = "admin"
	}

	existing, err := uc.repo.GetByIdentifier(ctx, email)
	switch {
	case err == nil:
		existing.Role = "admin"
		existing.IsBlocked = false
		if _, updateErr := uc.Update(ctx, existing); updateErr != nil {
			return entity.User{}, fmt.Errorf("UserUseCase - EnsureBootstrapAdmin - uc.Update: %w", updateErr)
		}
		return uc.repo.GetByID(ctx, existing.ID)
	case !errors.Is(err, usererr.ErrNotFound):
		return entity.User{}, fmt.Errorf("UserUseCase - EnsureBootstrapAdmin - uc.repo.GetByIdentifier: %w", err)
	}

	admin, err := uc.CreateByAdmin(ctx, entity.User{
		Login:      login,
		Email:      email,
		LastName:   strings.TrimSpace(cfg.LastName),
		FirstName:  strings.TrimSpace(cfg.FirstName),
		MiddleName: strings.TrimSpace(cfg.MiddleName),
		Phone:      phone,
		City:       strings.TrimSpace(cfg.City),
		Street:     strings.TrimSpace(cfg.Street),
		House:      strings.TrimSpace(cfg.House),
		Apartment:  strings.TrimSpace(cfg.Apartment),
		IsBlocked:  false,
		Role:       "admin",
	}, password)
	if err != nil {
		return entity.User{}, fmt.Errorf("UserUseCase - EnsureBootstrapAdmin - uc.CreateByAdmin: %w", err)
	}

	return admin, nil
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// normalizePhone extracts digits and normalizes to format +7 (XXX) XXX-XX-XX.
// Accepts: 79991234567, 89991234567 (→7999...), 9991234567 (→7999...), +7 (999) 123-45-67.
// Returns normalized digits (e.g. "79997777777") or error.
func normalizePhone(phone *string) error {
	var digits []byte
	for _, r := range *phone {
		if r >= '0' && r <= '9' {
			digits = append(digits, byte(r))
		}
	}
	s := string(digits)
	switch {
	case len(s) == 11 && s[0] == '7':
		// already ok
	case len(s) == 11 && s[0] == '8':
		s = "7" + s[1:]
	case len(s) == 10 && s[0] == '9':
		s = "7" + s
	default:
		return usererr.ErrInvalidPhone
	}
	*phone = s
	return nil
}

// Register creates a regular user registration request that must be confirmed by email code.
func (uc *UseCase) Register(ctx context.Context, u entity.User, password string) (entity.User, error) {
	if err := normalizePhone(&u.Phone); err != nil {
		return entity.User{}, err
	}
	u.Email = normalizeEmail(u.Email)
	u.Login = strings.TrimSpace(u.Login)
	u.Role = entity.UserRoleUser
	u.IsBlocked = false

	return uc.createPendingRegistration(ctx, u, password)
}

// CreateByAdmin creates a user immediately without email confirmation flow.
func (uc *UseCase) CreateByAdmin(ctx context.Context, u entity.User, password string) (entity.User, error) {
	if err := normalizePhone(&u.Phone); err != nil {
		return entity.User{}, err
	}
	u.Email = normalizeEmail(u.Email)
	u.Login = strings.TrimSpace(u.Login)
	if strings.TrimSpace(u.Role) == "" {
		u.Role = entity.UserRoleUser
	}

	return uc.createUserDirect(ctx, u, password)
}

func (uc *UseCase) createUserDirect(ctx context.Context, u entity.User, password string) (entity.User, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return entity.User{}, fmt.Errorf("UserUseCase - Create - bcrypt.GenerateFromPassword: %w", err)
	}
	u.PasswordHash = string(hashed)
	u.EmailVerified = true
	if err := uc.repo.Create(ctx, &u); err != nil {
		return entity.User{}, err
	}
	return u, nil
}

func (uc *UseCase) createPendingRegistration(ctx context.Context, u entity.User, password string) (entity.User, error) {
	if _, err := uc.repo.GetByIdentifier(ctx, u.Email); err == nil {
		return entity.User{}, usererr.ErrDuplicateEmail
	} else if !errors.Is(err, usererr.ErrNotFound) {
		return entity.User{}, err
	}
	if _, err := uc.repo.GetByIdentifier(ctx, u.Login); err == nil {
		return entity.User{}, usererr.ErrDuplicateLogin
	} else if !errors.Is(err, usererr.ErrNotFound) {
		return entity.User{}, err
	}
	if pend, err := uc.repo.GetPendingByEmail(ctx, u.Email); err != nil {
		return entity.User{}, err
	} else if pend != nil {
		return entity.User{}, usererr.ErrDuplicateEmail
	}
	if pend, err := uc.repo.GetPendingByLogin(ctx, u.Login); err != nil {
		return entity.User{}, err
	} else if pend != nil {
		return entity.User{}, usererr.ErrDuplicateLogin
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return entity.User{}, fmt.Errorf("UserUseCase - Create - bcrypt.GenerateFromPassword: %w", err)
	}

	pend := &entity.PendingRegistration{
		Email:        u.Email,
		Login:        u.Login,
		PasswordHash: string(hashed),
		LastName:     u.LastName,
		FirstName:    u.FirstName,
		MiddleName:   u.MiddleName,
		Phone:        u.Phone,
		City:         u.City,
		Street:       u.Street,
		House:        u.House,
		Apartment:    u.Apartment,
		Role:         u.Role,
	}
	if err := uc.repo.UpsertPendingRegistration(ctx, pend); err != nil {
		return entity.User{}, err
	}

	return entity.User{
		ID:            0,
		Login:         u.Login,
		Email:         u.Email,
		EmailVerified: false,
		LastName:      u.LastName,
		FirstName:     u.FirstName,
		MiddleName:    u.MiddleName,
		Phone:         u.Phone,
		City:          u.City,
		Street:        u.Street,
		House:         u.House,
		Apartment:     u.Apartment,
		IsBlocked:     u.IsBlocked,
		Role:          u.Role,
	}, nil
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

	if !u.EmailVerified {
		return entity.User{}, usererr.ErrEmailNotVerified
	}

	return u, nil
}

func (uc *UseCase) SendEmailVerificationCode(ctx context.Context, email, purpose string) error {
	email = strings.TrimSpace(email)
	purpose = strings.TrimSpace(purpose)

	if purpose == purposeRegister {
		if u, err := uc.repo.GetByIdentifier(ctx, email); err == nil {
			email = u.Email
		} else if errors.Is(err, usererr.ErrNotFound) {
			pend, perr := uc.repo.GetPendingByEmail(ctx, normalizeEmail(email))
			if perr != nil {
				return perr
			}
			if pend == nil {
				return usererr.ErrNotFound
			}
			email = pend.Email
		} else {
			return err
		}
	}

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

	if purpose == purposeRegister {
		var canon string
		if pend, err := uc.repo.GetPendingByEmail(ctx, normalizeEmail(email)); err != nil {
			return err
		} else if pend != nil {
			canon = pend.Email
		} else {
			u, err := uc.repo.GetByIdentifier(ctx, email)
			if err != nil {
				return err
			}
			canon = u.Email
		}
		email = canon
	}

	if err := uc.repo.ConsumeEmailVerificationCode(ctx, email, purpose, code, time.Now().Unix()); err != nil {
		return err
	}

	if purpose != purposeRegister {
		return nil
	}

	pend, err := uc.repo.GetPendingByEmail(ctx, email)
	if err != nil {
		return err
	}
	if pend != nil {
		nu := entity.User{
			Login:         pend.Login,
			Email:         pend.Email,
			PasswordHash:  pend.PasswordHash,
			LastName:      pend.LastName,
			FirstName:     pend.FirstName,
			MiddleName:    pend.MiddleName,
			Phone:         pend.Phone,
			City:          pend.City,
			Street:        pend.Street,
			House:         pend.House,
			Apartment:     pend.Apartment,
			Role:          pend.Role,
			IsBlocked:     false,
			EmailVerified: true,
		}
		if nu.Role == "" {
			nu.Role = "user"
		}
		if err := normalizePhone(&nu.Phone); err != nil {
			return err
		}
		if err := uc.repo.Create(ctx, &nu); err != nil {
			return err
		}
		if err := uc.repo.DeletePendingByEmail(ctx, email); err != nil {
			return err
		}
		return nil
	}

	if err := uc.repo.SetEmailVerifiedByEmail(ctx, email, true); err != nil {
		return err
	}
	return nil
}

// SendPasswordResetCode stores a one-time code and emails it (only if a user with this email exists).
func (uc *UseCase) SendPasswordResetCode(ctx context.Context, email string) error {
	email = strings.TrimSpace(email)
	u, err := uc.repo.GetByIdentifier(ctx, email)
	if err != nil {
		if errors.Is(err, usererr.ErrNotFound) {
			return nil
		}
		return err
	}

	const ttl = 10 * time.Minute
	code := mailsender.RandomRumber().String()
	if err := uc.repo.CreateEmailVerificationCode(
		ctx,
		u.Email,
		purposePasswordReset,
		code,
		time.Now().Add(ttl).Unix(),
	); err != nil {
		return err
	}

	subject := "Password Recovery 'SdayKenta'"
	body := "Your code is " + code
	if err := mailsender.SendMail(subject, body, []string{u.Email}); err != nil {
		return fmt.Errorf("UserUseCase - SendPasswordResetCode - SendMail: %w", err)
	}

	return nil
}

// ResetPasswordWithCode verifies the emailed code and sets a new password.
func (uc *UseCase) ResetPasswordWithCode(ctx context.Context, email, code, newPassword string) error {
	email = strings.TrimSpace(email)
	code = strings.TrimSpace(code)
	newPassword = strings.TrimSpace(newPassword)
	if len(newPassword) < 6 {
		return usererr.ErrPasswordTooShort
	}

	u, err := uc.repo.GetByIdentifier(ctx, email)
	if err != nil {
		return err
	}

	if err := uc.repo.ConsumeEmailVerificationCode(
		ctx,
		u.Email,
		purposePasswordReset,
		code,
		time.Now().Unix(),
	); err != nil {
		return err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("UserUseCase - ResetPasswordWithCode - bcrypt: %w", err)
	}

	if err := uc.repo.UpdatePasswordHashByEmail(ctx, u.Email, string(hashed)); err != nil {
		return err
	}

	if err := uc.repo.SetEmailVerifiedByEmail(ctx, u.Email, true); err != nil {
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
	if err := normalizePhone(&u.Phone); err != nil {
		return entity.User{}, err
	}
	u.Email = normalizeEmail(u.Email)
	old, err := uc.repo.GetByID(ctx, u.ID)
	if err != nil {
		return entity.User{}, err
	}
	if !strings.EqualFold(old.Email, strings.TrimSpace(u.Email)) {
		u.EmailVerified = false
	} else {
		u.EmailVerified = old.EmailVerified
	}
	if err := uc.repo.Update(ctx, &u); err != nil {
		return entity.User{}, err
	}
	out, err := uc.repo.GetByID(ctx, u.ID)
	if err != nil {
		return entity.User{}, err
	}
	return out, nil
}

// UpdateAvatar updates the avatar identifier/URL for a user.
func (uc *UseCase) UpdateAvatar(ctx context.Context, id int64, avatarURL string) error {
	if err := uc.repo.UpdateAvatar(ctx, id, avatarURL); err != nil {
		return err
	}

	return nil
}
