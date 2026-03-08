package geo

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/evrone/go-clean-template/internal/entity"
	"github.com/evrone/go-clean-template/internal/repo"
)

type UseCase struct {
	geoRepo repo.GeoRepo
	webAPI  repo.GeoWebAPI
}

func New(geoRepo repo.GeoRepo, webAPI repo.GeoWebAPI) *UseCase {
	return &UseCase{
		geoRepo: geoRepo,
		webAPI:  webAPI,
	}
}

func (uc *UseCase) ReverseGeocode(ctx context.Context, lat, lon float64) (entity.Address, error) {
	if !validCoordinates(lat, lon) {
		return entity.Address{}, entity.ErrInvalidCoordinates
	}

	allowed, err := uc.geoRepo.IsInAllowedZone(ctx, lat, lon)
	if err != nil {
		return entity.Address{}, fmt.Errorf("GeoUseCase - ReverseGeocode - uc.geoRepo.IsInAllowedZone: %w", err)
	}
	if !allowed {
		return entity.Address{}, entity.ErrOutOfAllowedZone
	}

	cachedAddress, err := uc.geoRepo.GetAddressByCoords(ctx, lat, lon)
	if err == nil {
		return cachedAddress, nil
	}
	if !errors.Is(err, entity.ErrAddressNotFound) {
		return entity.Address{}, fmt.Errorf("GeoUseCase - ReverseGeocode - uc.geoRepo.GetAddressByCoords: %w", err)
	}

	address, err := uc.webAPI.Reverse(ctx, lat, lon)
	if err != nil {
		return entity.Address{}, fmt.Errorf("GeoUseCase - ReverseGeocode - uc.webAPI.Reverse: %w", err)
	}

	if err = uc.geoRepo.SaveAddress(ctx, address); err != nil {
		return entity.Address{}, fmt.Errorf("GeoUseCase - ReverseGeocode - uc.geoRepo.SaveAddress: %w", err)
	}

	return address, nil
}

func (uc *UseCase) Search(ctx context.Context, query string) ([]entity.Address, error) {
	trimmed := strings.TrimSpace(query)
	if len([]rune(trimmed)) < 3 {
		return nil, entity.ErrInvalidSearchQuery
	}

	addresses, err := uc.webAPI.Search(ctx, trimmed)
	if err != nil {
		return nil, fmt.Errorf("GeoUseCase - Search - uc.webAPI.Search: %w", err)
	}
	if len(addresses) == 0 {
		return []entity.Address{}, nil
	}

	filtered := make([]entity.Address, 0, len(addresses))
	outside := make([]entity.Address, 0, len(addresses))
	for _, address := range addresses {
		allowed, zoneErr := uc.geoRepo.IsInAllowedZone(ctx, address.Lat, address.Lon)
		if zoneErr != nil {
			return nil, fmt.Errorf("GeoUseCase - Search - uc.geoRepo.IsInAllowedZone: %w", zoneErr)
		}
		if !allowed {
			outside = append(outside, address)
			continue
		}

		filtered = append(filtered, address)
		_ = uc.geoRepo.SaveAddress(ctx, address)
	}

	if len(filtered) == 0 {
		if len(outside) > 0 && queryExplicitlyTargetsOutsideArea(trimmed, outside) {
			return nil, entity.ErrOutOfAllowedZone
		}
		return []entity.Address{}, nil
	}

	return filtered, nil
}

func queryExplicitlyTargetsOutsideArea(query string, candidates []entity.Address) bool {
	normQuery := normalizeSearchText(query)
	if normQuery == "" {
		return false
	}
	for _, candidate := range candidates {
		city := normalizeSearchText(candidate.City)
		if city != "" && strings.Contains(normQuery, city) {
			return true
		}
	}
	return false
}

func normalizeSearchText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "ё", "е")
	return strings.Join(strings.Fields(value), " ")
}

func validCoordinates(lat, lon float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) {
		return false
	}

	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}
