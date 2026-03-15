// backend/internal/repo/contracts.go
// Package repo implements application outer layer logic. Each logic group in own file.
package repo

import (
	"context"

	"github.com/evrone/go-clean-template/internal/entity"
)

//go:generate mockgen -source=contracts.go -destination=../usecase/mocks_repo_test.go -package=usecase_test

type (
	// TranslationRepo -.
	/*TranslationRepo interface {
		Store(context.Context, entity.Translation) error
		GetHistory(context.Context) ([]entity.Translation, error)
	}

	// TranslationWebAPI -.
	TranslationWebAPI interface {
		Translate(entity.Translation) (entity.Translation, error)
	}
	*/

	CategoryRepo interface {
		GetAll(ctx context.Context) ([]entity.Category, error)
		GetByID(ctx context.Context, id int) (entity.Category, error)
		Create(ctx context.Context, input entity.CreateCategoryInput) (int, error)
		Update(ctx context.Context, id int, input entity.UpdateCategoryInput) error
		Delete(ctx context.Context, id int) error
	}
	// GeoRepo описывает работу с нашей БД (PostGIS)
	GeoRepo interface {
		// GetAddressByCoords ищет закэшированный адрес в радиусе ~10-20 метров
		GetAddressByCoords(ctx context.Context, lat, lon float64) (entity.Address, error)

		// SaveAddress сохраняет новый адрес в кэш
		SaveAddress(ctx context.Context, addr entity.Address) error

		// IsPointInZone проверяет, попадает ли точка в конкретную зону работы
		IsPointInZone(ctx context.Context, lat, lon float64, zoneName string) (bool, error)
		// FindContainingZone ищет первую зону, в которую попадает точка.
		FindContainingZone(ctx context.Context, lat, lon float64) (entity.Zone, error)
		// GetZones returns supported project zones with user-facing display names.
		GetZones(ctx context.Context) ([]entity.Zone, error)
	}

	// GeoWebAPI описывает работу с внешним сервисом Nominatim
	GeoWebAPI interface {
		Reverse(ctx context.Context, lat, lon float64) (entity.Address, error)
		Search(ctx context.Context, query, city string) ([]entity.Address, error)
	}
)
