package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/usererr"
)

func TestRegisterCreatesPendingRegularUser(t *testing.T) {
	t.Parallel()

	repo := &userRepoStub{}
	uc := New(repo)

	user, err := uc.Register(context.Background(), entity.User{
		Login:     "user123",
		Email:     "USER@example.com",
		LastName:  "Иванов",
		FirstName: "Иван",
		Phone:     "+7 (999) 123-45-67",
		City:      "Москва",
		Street:    "Тверская",
		House:     "1",
		Apartment: "10",
		Role:      entity.UserRoleAdmin,
		IsBlocked: true,
	}, "qwerty123")

	require.NoError(t, err)
	require.NotNil(t, repo.pending)
	require.Equal(t, entity.UserRoleUser, repo.pending.Role)
	require.Equal(t, "79991234567", repo.pending.Phone)
	require.Equal(t, "user@example.com", repo.pending.Email)
	require.False(t, user.EmailVerified)
	require.False(t, user.IsBlocked)
	require.Equal(t, entity.UserRoleUser, user.Role)
	require.False(t, repo.createCalled)
}

func TestCreateByAdminCreatesUserDirectly(t *testing.T) {
	t.Parallel()

	repo := &userRepoStub{}
	uc := New(repo)

	user, err := uc.CreateByAdmin(context.Background(), entity.User{
		Login:     "manager1",
		Email:     "manager@example.com",
		LastName:  "Петров",
		FirstName: "Пётр",
		Phone:     "+7 (901) 000-00-00",
		City:      "Самара",
		Street:    "Ленина",
		House:     "2",
		Role:      entity.UserRolePremium,
		IsBlocked: true,
	}, "qwerty123")

	require.NoError(t, err)
	require.True(t, repo.createCalled)
	require.NotNil(t, repo.created)
	require.Nil(t, repo.pending)
	require.Equal(t, entity.UserRolePremium, repo.created.Role)
	require.True(t, repo.created.IsBlocked)
	require.True(t, repo.created.EmailVerified)
	require.Equal(t, repo.created.ID, user.ID)
	require.Equal(t, "79010000000", repo.created.Phone)
}

type userRepoStub struct {
	createCalled bool
	created      *entity.User
	pending      *entity.PendingRegistration
}

func (s *userRepoStub) Create(_ context.Context, user *entity.User) error {
	s.createCalled = true
	copyUser := *user
	copyUser.ID = 42
	s.created = &copyUser
	user.ID = copyUser.ID
	return nil
}

func (s *userRepoStub) Delete(_ context.Context, _ int64) error {
	return nil
}

func (s *userRepoStub) GetByID(_ context.Context, id int64) (entity.User, error) {
	if s.created != nil && s.created.ID == id {
		return *s.created, nil
	}
	return entity.User{}, usererr.ErrNotFound
}

func (s *userRepoStub) GetByIdentifier(_ context.Context, _ string) (entity.User, error) {
	return entity.User{}, usererr.ErrNotFound
}

func (s *userRepoStub) CreateEmailVerificationCode(_ context.Context, _, _, _ string, _ int64) error {
	return nil
}

func (s *userRepoStub) ConsumeEmailVerificationCode(_ context.Context, _, _, _ string, _ int64) error {
	return nil
}

func (s *userRepoStub) List(_ context.Context) ([]entity.User, error) {
	return nil, nil
}

func (s *userRepoStub) Update(_ context.Context, _ *entity.User) error {
	return nil
}

func (s *userRepoStub) UpdateAvatar(_ context.Context, _ int64, _ string) error {
	return nil
}

func (s *userRepoStub) UpdatePasswordHashByEmail(_ context.Context, _, _ string) error {
	return nil
}

func (s *userRepoStub) SetEmailVerifiedByEmail(_ context.Context, _ string, _ bool) error {
	return nil
}

func (s *userRepoStub) UpsertPendingRegistration(_ context.Context, p *entity.PendingRegistration) error {
	copyPending := *p
	s.pending = &copyPending
	return nil
}

func (s *userRepoStub) GetPendingByEmail(_ context.Context, _ string) (*entity.PendingRegistration, error) {
	return nil, nil
}

func (s *userRepoStub) GetPendingByLogin(_ context.Context, _ string) (*entity.PendingRegistration, error) {
	return nil, nil
}

func (s *userRepoStub) DeletePendingByEmail(_ context.Context, _ string) error {
	return nil
}

func TestCreateByAdminDefaultsRoleToUser(t *testing.T) {
	t.Parallel()

	repo := &userRepoStub{}
	uc := New(repo)

	user, err := uc.CreateByAdmin(context.Background(), entity.User{
		Login:     "user123",
		Email:     "user@example.com",
		LastName:  "Иванов",
		FirstName: "Иван",
		Phone:     "89991234567",
		City:      "Москва",
		Street:    "Тверская",
		House:     "1",
	}, "qwerty123")

	require.NoError(t, err)
	require.Equal(t, entity.UserRoleUser, repo.created.Role)
	require.Equal(t, entity.UserRoleUser, user.Role)
}

func TestRegisterPropagatesDuplicateErrors(t *testing.T) {
	t.Parallel()

	repo := &userRepoDuplicateEmailStub{}
	uc := New(repo)

	_, err := uc.Register(context.Background(), entity.User{
		Login:     "user123",
		Email:     "user@example.com",
		LastName:  "Иванов",
		FirstName: "Иван",
		Phone:     "89991234567",
		City:      "Москва",
		Street:    "Тверская",
		House:     "1",
	}, "qwerty123")

	require.ErrorIs(t, err, usererr.ErrDuplicateEmail)
}

type userRepoDuplicateEmailStub struct {
	userRepoStub
}

func (s *userRepoDuplicateEmailStub) GetByIdentifier(_ context.Context, identifier string) (entity.User, error) {
	if strings.EqualFold(identifier, "user@example.com") {
		return entity.User{ID: 1}, nil
	}
	return entity.User{}, usererr.ErrNotFound
}

func (s *userRepoDuplicateEmailStub) GetPendingByEmail(_ context.Context, _ string) (*entity.PendingRegistration, error) {
	return nil, errors.New("should not be called")
}
