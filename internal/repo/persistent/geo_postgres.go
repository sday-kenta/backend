package persistent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/evrone/go-clean-template/internal/entity"
	"github.com/evrone/go-clean-template/pkg/postgres"
	"github.com/jackc/pgx/v5"
)

const defaultZoneName = "samara"

// GeoRepo persists cached addresses and validates project area via PostGIS.
type GeoRepo struct {
	*postgres.Postgres
	cacheRadiusMeters int
	zoneName          string
}

func NewGeoRepo(pg *postgres.Postgres, cacheRadiusMeters int, zoneName string) *GeoRepo {
	if cacheRadiusMeters <= 0 {
		cacheRadiusMeters = 20
	}
	if zoneName == "" {
		zoneName = defaultZoneName
	}

	return &GeoRepo{
		Postgres:          pg,
		cacheRadiusMeters: cacheRadiusMeters,
		zoneName:          zoneName,
	}
}

func (r *GeoRepo) GetAddressByCoords(ctx context.Context, lat, lon float64) (entity.Address, error) {
	query := r.Builder.
		Select("lat", "lon", "city", "road", "house_number", "full_address").
		From("addresses").
		Where(squirrel.Expr(
			"ST_DWithin(geom::geography, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, ?)",
			lon,
			lat,
			r.cacheRadiusMeters,
		)).
		OrderByClause(
			"ST_Distance(geom::geography, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography)",
			lon,
			lat,
		).
		Limit(1)

	sql, args, err := query.ToSql()
	if err != nil {
		return entity.Address{}, fmt.Errorf("GeoRepo - GetAddressByCoords - query.ToSql: %w", err)
	}

	var address entity.Address
	if err = r.Pool.QueryRow(ctx, sql, args...).Scan(
		&address.Lat,
		&address.Lon,
		&address.City,
		&address.Road,
		&address.HouseNumber,
		&address.FullAddress,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Address{}, entity.ErrAddressNotFound
		}

		return entity.Address{}, fmt.Errorf("GeoRepo - GetAddressByCoords - r.Pool.QueryRow: %w", err)
	}

	return address, nil
}

func (r *GeoRepo) SaveAddress(ctx context.Context, addr entity.Address) error {
	fullAddress := strings.TrimSpace(addr.FullAddress)
	if fullAddress == "" {
		return nil
	}

	var existing int
	err := r.Pool.QueryRow(ctx, `SELECT 1 FROM addresses WHERE full_address = $1 LIMIT 1`, fullAddress).Scan(&existing)
	if err == nil {
		return nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("GeoRepo - SaveAddress - duplicate check: %w", err)
	}

	query := r.Builder.
		Insert("addresses").
		Columns("lat", "lon", "city", "road", "house_number", "full_address", "geom").
		Values(
			addr.Lat,
			addr.Lon,
			addr.City,
			addr.Road,
			addr.HouseNumber,
			fullAddress,
			squirrel.Expr("ST_SetSRID(ST_MakePoint(?, ?), 4326)", addr.Lon, addr.Lat),
		)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("GeoRepo - SaveAddress - query.ToSql: %w", err)
	}

	if _, err = r.Pool.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("GeoRepo - SaveAddress - r.Pool.Exec: %w", err)
	}

	return nil
}

func (r *GeoRepo) IsInAllowedZone(ctx context.Context, lat, lon float64) (bool, error) {
	sql := `
SELECT EXISTS (
    SELECT 1
    FROM zones
    WHERE name = $1
      AND ST_Covers(geom, ST_SetSRID(ST_MakePoint($2, $3), 4326))
)`

	var allowed bool
	if err := r.Pool.QueryRow(ctx, sql, r.zoneName, lon, lat).Scan(&allowed); err != nil {
		return false, fmt.Errorf("GeoRepo - IsInAllowedZone - r.Pool.QueryRow: %w", err)
	}

	return allowed, nil
}
