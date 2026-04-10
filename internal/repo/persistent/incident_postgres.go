package persistent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/incidenterr"
	"github.com/sday-kenta/backend/pkg/postgres"
)

// IncidentRepo implements repo.IncidentRepo using Postgres.
type IncidentRepo struct {
	*postgres.Postgres
}

// NewIncidentRepo creates incident repository.
func NewIncidentRepo(pg *postgres.Postgres) *IncidentRepo {
	return &IncidentRepo{pg}
}

func mapIncidentErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return incidenterr.ErrNotFound
	}

	return err
}

// Create inserts a new incident and fills generated fields.
func (r *IncidentRepo) Create(ctx context.Context, incident *entity.Incident) error {
	query, args, err := r.Builder.
		Insert("incidents").
		Columns(
			"user_id",
			"category_id",
			"title",
			"description",
			"status",
			"department_name",
			"city",
			"street",
			"house",
			"address_text",
			"latitude",
			"longitude",
			"reporter_full_name",
			"reporter_email",
			"reporter_phone",
			"reporter_address",
			"published_at",
		).
		Values(
			incident.UserID,
			incident.CategoryID,
			incident.Title,
			incident.Description,
			incident.Status,
			incident.DepartmentName,
			incident.City,
			incident.Street,
			incident.House,
			incident.AddressText,
			incident.Latitude,
			incident.Longitude,
			incident.ReporterFullName,
			incident.ReporterEmail,
			incident.ReporterPhone,
			incident.ReporterAddress,
			incident.PublishedAt,
		).
		Suffix("RETURNING id, created_at, updated_at, published_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("IncidentRepo - Create - r.Builder: %w", err)
	}

	var publishedAt sql.NullTime
	if err = r.Pool.QueryRow(ctx, query, args...).Scan(
		&incident.ID,
		&incident.CreatedAt,
		&incident.UpdatedAt,
		&publishedAt,
	); err != nil {
		return mapIncidentErr(err)
	}
	if publishedAt.Valid {
		incident.PublishedAt = &publishedAt.Time
	}

	return nil
}

// GetByID returns a single incident with photos.
func (r *IncidentRepo) GetByID(ctx context.Context, id int64) (entity.Incident, error) {
	query, args, err := r.Builder.
		Select(
			"i.id",
			"i.user_id",
			"i.category_id",
			"c.title",
			"i.title",
			"i.description",
			"i.status",
			"i.department_name",
			"i.city",
			"i.street",
			"i.house",
			"i.address_text",
			"i.latitude",
			"i.longitude",
			"i.reporter_full_name",
			"i.reporter_email",
			"i.reporter_phone",
			"i.reporter_address",
			"i.created_at",
			"i.updated_at",
			"i.published_at",
		).
		From("incidents i").
		Join("categories c ON c.id = i.category_id").
		Where(squirrel.Eq{"i.id": id}).
		ToSql()
	if err != nil {
		return entity.Incident{}, fmt.Errorf("IncidentRepo - GetByID - r.Builder: %w", err)
	}

	incident, err := scanIncidentRow(r.Pool.QueryRow(ctx, query, args...))
	if err != nil {
		return entity.Incident{}, mapIncidentErr(err)
	}

	photosMap, err := r.loadPhotosByIncidentIDs(ctx, []int64{id})
	if err != nil {
		return entity.Incident{}, err
	}
	incident.Photos = photosMap[id]

	return incident, nil
}

// List returns incidents list with photos.
func (r *IncidentRepo) List(ctx context.Context, filter entity.IncidentFilter) ([]entity.Incident, error) {
	builder := r.Builder.
		Select(
			"i.id",
			"i.user_id",
			"i.category_id",
			"c.title",
			"i.title",
			"i.description",
			"i.status",
			"i.department_name",
			"i.city",
			"i.street",
			"i.house",
			"i.address_text",
			"i.latitude",
			"i.longitude",
			"i.reporter_full_name",
			"i.reporter_email",
			"i.reporter_phone",
			"i.reporter_address",
			"i.created_at",
			"i.updated_at",
			"i.published_at",
		).
		From("incidents i").
		Join("categories c ON c.id = i.category_id").
		OrderBy("i.created_at DESC", "i.id DESC")

	if filter.UserID != nil {
		builder = builder.Where(squirrel.Eq{"i.user_id": *filter.UserID})
	}
	if filter.CategoryID != nil {
		builder = builder.Where(squirrel.Eq{"i.category_id": *filter.CategoryID})
	}
	if filter.Status != nil {
		builder = builder.Where(squirrel.Eq{"i.status": *filter.Status})
	}
	if len(filter.Statuses) > 0 {
		builder = builder.Where(squirrel.Eq{"i.status": filter.Statuses})
	}
	if filter.OnlyPublished {
		builder = builder.Where(squirrel.Eq{"i.status": entity.IncidentStatusPublished})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("IncidentRepo - List - builder.ToSql: %w", err)
	}

	rows, err := r.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("IncidentRepo - List - r.Pool.Query: %w", err)
	}
	defer rows.Close()

	incidents := make([]entity.Incident, 0)
	incidentIDs := make([]int64, 0)
	for rows.Next() {
		incident, scanErr := scanIncidentRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("IncidentRepo - List - rows.Scan: %w", scanErr)
		}
		incidents = append(incidents, incident)
		incidentIDs = append(incidentIDs, incident.ID)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("IncidentRepo - List - rows.Err: %w", err)
	}

	photosMap, err := r.loadPhotosByIncidentIDs(ctx, incidentIDs)
	if err != nil {
		return nil, err
	}
	for idx := range incidents {
		incidents[idx].Photos = photosMap[incidents[idx].ID]
	}

	return incidents, nil
}

// Update updates mutable incident fields.
func (r *IncidentRepo) Update(ctx context.Context, incident *entity.Incident) error {
	builder := r.Builder.
		Update("incidents").
		Set("category_id", incident.CategoryID).
		Set("title", incident.Title).
		Set("description", incident.Description).
		Set("status", incident.Status).
		Set("department_name", incident.DepartmentName).
		Set("city", incident.City).
		Set("street", incident.Street).
		Set("house", incident.House).
		Set("address_text", incident.AddressText).
		Set("latitude", incident.Latitude).
		Set("longitude", incident.Longitude).
		Set("published_at", incident.PublishedAt).
		Set("updated_at", squirrel.Expr("NOW()"))

	query, args, err := builder.
		Where(squirrel.Eq{"id": incident.ID}).
		Suffix("RETURNING updated_at, published_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("IncidentRepo - Update - builder.ToSql: %w", err)
	}

	var publishedAt sql.NullTime
	if err = r.Pool.QueryRow(ctx, query, args...).Scan(&incident.UpdatedAt, &publishedAt); err != nil {
		return mapIncidentErr(err)
	}
	incident.PublishedAt = nil
	if publishedAt.Valid {
		incident.PublishedAt = &publishedAt.Time
	}

	return nil
}

// Delete removes incident.
func (r *IncidentRepo) Delete(ctx context.Context, id int64) error {
	query, args, err := r.Builder.
		Delete("incidents").
		Where(squirrel.Eq{"id": id}).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("IncidentRepo - Delete - builder.ToSql: %w", err)
	}

	var deletedID int64
	if err = r.Pool.QueryRow(ctx, query, args...).Scan(&deletedID); err != nil {
		return mapIncidentErr(err)
	}

	return nil
}

// CreatePhoto inserts incident photo metadata.
func (r *IncidentRepo) CreatePhoto(ctx context.Context, photo *entity.IncidentPhoto) error {
	query, args, err := r.Builder.
		Insert("incident_photos").
		Columns("incident_id", "file_key", "file_url", "content_type", "size_bytes", "sort_order").
		Values(photo.IncidentID, photo.FileKey, photo.FileURL, photo.ContentType, photo.SizeBytes, photo.SortOrder).
		Suffix("RETURNING id, created_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("IncidentRepo - CreatePhoto - builder.ToSql: %w", err)
	}

	if err = r.Pool.QueryRow(ctx, query, args...).Scan(&photo.ID, &photo.CreatedAt); err != nil {
		return fmt.Errorf("IncidentRepo - CreatePhoto - QueryRow: %w", err)
	}

	return nil
}

// DeletePhoto removes incident photo metadata and returns deleted row.
func (r *IncidentRepo) DeletePhoto(ctx context.Context, incidentID, photoID int64) (entity.IncidentPhoto, error) {
	query, args, err := r.Builder.
		Delete("incident_photos").
		Where(squirrel.Eq{"id": photoID, "incident_id": incidentID}).
		Suffix("RETURNING id, incident_id, file_key, file_url, content_type, size_bytes, sort_order, created_at").
		ToSql()
	if err != nil {
		return entity.IncidentPhoto{}, fmt.Errorf("IncidentRepo - DeletePhoto - builder.ToSql: %w", err)
	}

	var photo entity.IncidentPhoto
	if err = r.Pool.QueryRow(ctx, query, args...).Scan(
		&photo.ID,
		&photo.IncidentID,
		&photo.FileKey,
		&photo.FileURL,
		&photo.ContentType,
		&photo.SizeBytes,
		&photo.SortOrder,
		&photo.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.IncidentPhoto{}, incidenterr.ErrPhotoNotFound
		}
		return entity.IncidentPhoto{}, fmt.Errorf("IncidentRepo - DeletePhoto - QueryRow: %w", err)
	}

	return photo, nil
}

func (r *IncidentRepo) loadPhotosByIncidentIDs(ctx context.Context, incidentIDs []int64) (map[int64][]entity.IncidentPhoto, error) {
	result := make(map[int64][]entity.IncidentPhoto, len(incidentIDs))
	if len(incidentIDs) == 0 {
		return result, nil
	}

	rows, err := r.Pool.Query(ctx, `
SELECT id, incident_id, file_key, file_url, content_type, size_bytes, sort_order, created_at
FROM incident_photos
WHERE incident_id = ANY($1)
ORDER BY incident_id ASC, sort_order ASC, id ASC`, incidentIDs)
	if err != nil {
		return nil, fmt.Errorf("IncidentRepo - loadPhotosByIncidentIDs - Query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var photo entity.IncidentPhoto
		if err = rows.Scan(
			&photo.ID,
			&photo.IncidentID,
			&photo.FileKey,
			&photo.FileURL,
			&photo.ContentType,
			&photo.SizeBytes,
			&photo.SortOrder,
			&photo.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("IncidentRepo - loadPhotosByIncidentIDs - rows.Scan: %w", err)
		}

		result[photo.IncidentID] = append(result[photo.IncidentID], photo)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("IncidentRepo - loadPhotosByIncidentIDs - rows.Err: %w", err)
	}

	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanIncidentRow(scanner rowScanner) (entity.Incident, error) {
	var (
		incident    entity.Incident
		city        sql.NullString
		street      sql.NullString
		house       sql.NullString
		publishedAt sql.NullTime
	)

	err := scanner.Scan(
		&incident.ID,
		&incident.UserID,
		&incident.CategoryID,
		&incident.CategoryTitle,
		&incident.Title,
		&incident.Description,
		&incident.Status,
		&incident.DepartmentName,
		&city,
		&street,
		&house,
		&incident.AddressText,
		&incident.Latitude,
		&incident.Longitude,
		&incident.ReporterFullName,
		&incident.ReporterEmail,
		&incident.ReporterPhone,
		&incident.ReporterAddress,
		&incident.CreatedAt,
		&incident.UpdatedAt,
		&publishedAt,
	)
	if err != nil {
		return entity.Incident{}, err
	}

	if city.Valid {
		incident.City = city.String
	}
	if street.Valid {
		incident.Street = street.String
	}
	if house.Valid {
		incident.House = house.String
	}
	if publishedAt.Valid {
		incident.PublishedAt = &publishedAt.Time
	}

	return incident, nil
}
