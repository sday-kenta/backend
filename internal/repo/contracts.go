package repo

import (
	"context"

	"github.com/sday-kenta/backend/internal/entity"
)

//go:generate mockgen -source=contracts.go -destination=../usecase/mocks_repo_test.go -package=usecase_test

type (
	CategoryRepo interface {
		GetAll(ctx context.Context) ([]entity.Category, error)
		GetByID(ctx context.Context, id int) (entity.Category, error)
		Create(ctx context.Context, input entity.CreateCategoryInput) (int, error)
		Update(ctx context.Context, id int, input entity.UpdateCategoryInput) error
		UpdateIcon(ctx context.Context, id int, iconURL *string) error
		Delete(ctx context.Context, id int) error
	}

	GeoRepo interface {
		GetAddressByCoords(ctx context.Context, lat, lon float64) (entity.Address, error)
		SaveAddress(ctx context.Context, addr entity.Address) error
		IsPointInZone(ctx context.Context, lat, lon float64, zoneName string) (bool, error)
		FindContainingZone(ctx context.Context, lat, lon float64) (entity.Zone, error)
		GetZones(ctx context.Context) ([]entity.Zone, error)
	}

	GeoWebAPI interface {
		Reverse(ctx context.Context, lat, lon float64) (entity.Address, error)
		Search(ctx context.Context, query, city string) ([]entity.Address, error)
	}

	UserRepo interface {
		Create(ctx context.Context, user *entity.User) error
		Delete(ctx context.Context, id int64) error
		GetByID(ctx context.Context, id int64) (entity.User, error)
		GetByIdentifier(ctx context.Context, identifier string) (entity.User, error)
		CreateEmailVerificationCode(ctx context.Context, email, purpose, code string, expiresAtUnix int64) error
		ConsumeEmailVerificationCode(ctx context.Context, email, purpose, code string, nowUnix int64) error
		List(ctx context.Context) ([]entity.User, error)
		Update(ctx context.Context, user *entity.User) error
		UpdateAvatar(ctx context.Context, id int64, avatarURL string) error
		UpdatePasswordHashByEmail(ctx context.Context, email, passwordHash string) error
		SetEmailVerifiedByEmail(ctx context.Context, email string, verified bool) error

		UpsertPendingRegistration(ctx context.Context, p *entity.PendingRegistration) error
		GetPendingByEmail(ctx context.Context, email string) (*entity.PendingRegistration, error)
		GetPendingByLogin(ctx context.Context, login string) (*entity.PendingRegistration, error)
		DeletePendingByEmail(ctx context.Context, email string) error
	}

	IncidentRepo interface {
		Create(ctx context.Context, incident *entity.Incident) error
		GetByID(ctx context.Context, id int64) (entity.Incident, error)
		List(ctx context.Context, filter entity.IncidentFilter) ([]entity.Incident, error)
		Update(ctx context.Context, incident *entity.Incident) error
		Delete(ctx context.Context, id int64) error
		CreatePhoto(ctx context.Context, photo *entity.IncidentPhoto) error
		DeletePhoto(ctx context.Context, incidentID, photoID int64) (entity.IncidentPhoto, error)
	}

	PushDeviceRepo interface {
		Upsert(ctx context.Context, device *entity.PushDevice) error
		ListByUserID(ctx context.Context, userID int64) ([]entity.PushDevice, error)
		DeleteByUserAndDeviceID(ctx context.Context, userID int64, deviceID string) error
		DeleteByToken(ctx context.Context, token string) error
	}
)
