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

// GeoRepo persists cached addresses and validates project area via PostGIS.
type GeoRepo struct {
	*postgres.Postgres
	cacheRadiusMeters int
}

func NewGeoRepo(pg *postgres.Postgres, cacheRadiusMeters int) *GeoRepo {
	if cacheRadiusMeters <= 0 {
		cacheRadiusMeters = 20
	}

	return &GeoRepo{
		Postgres:          pg,
		cacheRadiusMeters: cacheRadiusMeters,
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

func (r *GeoRepo) GetZones(ctx context.Context) ([]entity.Zone, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT name, COALESCE(NULLIF(BTRIM(display_name), ''), name)
		FROM zones
		ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("GeoRepo - GetZones - r.Pool.Query: %w", err)
	}
	defer rows.Close()

	result := make([]entity.Zone, 0)
	for rows.Next() {
		var zone entity.Zone
		if err := rows.Scan(&zone.Name, &zone.DisplayName); err != nil {
			return nil, fmt.Errorf("GeoRepo - GetZones - rows.Scan: %w", err)
		}
		zone.Name = strings.TrimSpace(zone.Name)
		zone.DisplayName = strings.TrimSpace(zone.DisplayName)
		if zone.Name == "" {
			continue
		}
		if zone.DisplayName == "" {
			zone.DisplayName = zone.Name
		}
		result = append(result, zone)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GeoRepo - GetZones - rows.Err: %w", err)
	}
	return result, nil
}

func (r *GeoRepo) IsPointInZone(ctx context.Context, lat, lon float64, zoneName string) (bool, error) {
	zoneName = strings.TrimSpace(zoneName)
	if zoneName == "" {
		return false, entity.ErrZoneNotFound
	}

	sql := `
SELECT EXISTS (
    SELECT 1
    FROM zones
    WHERE name = $1
      AND ST_Covers(geom, ST_SetSRID(ST_MakePoint($2, $3), 4326))
)`

	var allowed bool
	if err := r.Pool.QueryRow(ctx, sql, zoneName, lon, lat).Scan(&allowed); err != nil {
		return false, fmt.Errorf("GeoRepo - IsPointInZone - r.Pool.QueryRow: %w", err)
	}

	return allowed, nil
}

func (r *GeoRepo) FindContainingZone(ctx context.Context, lat, lon float64) (entity.Zone, error) {
	sql := `
SELECT name, COALESCE(NULLIF(BTRIM(display_name), ''), name)
FROM zones
WHERE ST_Covers(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326))
ORDER BY ST_Area(geom) ASC, id ASC
LIMIT 1`

	var zone entity.Zone
	if err := r.Pool.QueryRow(ctx, sql, lon, lat).Scan(&zone.Name, &zone.DisplayName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Zone{}, entity.ErrZoneNotFound
		}
		return entity.Zone{}, fmt.Errorf("GeoRepo - FindContainingZone - r.Pool.QueryRow: %w", err)
	}
	zone.Name = strings.TrimSpace(zone.Name)
	zone.DisplayName = strings.TrimSpace(zone.DisplayName)
	if zone.DisplayName == "" {
		zone.DisplayName = zone.Name
	}
	return zone, nil
}
