// backend/internal/repo/persistent/category_postgres.go

package persistent

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/sday-kenta/backend/internal/categoryerr"
	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/pkg/postgres"
)

// CategoryRepo -.
type CategoryRepo struct {
	*postgres.Postgres
}

// NewCategoryRepo -.
func NewCategoryRepo(pg *postgres.Postgres) *CategoryRepo {
	return &CategoryRepo{pg}
}

func mapCategoryErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return categoryerr.ErrNotFound
	}

	return err
}

// GetAll -.
func (r *CategoryRepo) GetAll(ctx context.Context) ([]entity.Category, error) {
	// Строим SQL запрос через Squirrel
	sql, args, err := r.Builder.
		Select("id, title, icon_url").
		From("categories").
		Where("is_active = true").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("CategoryRepo - GetAll - r.Builder: %w", err)
	}

	// Выполняем запрос
	rows, err := r.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("CategoryRepo - GetAll - r.Pool.Query: %w", err)
	}
	defer rows.Close()

	entities := make([]entity.Category, 0, 10)

	for rows.Next() {
		var e entity.Category
		var iconURL *string // указатель, так как в базе может быть NULL

		err = rows.Scan(&e.ID, &e.Title, &iconURL)
		if err != nil {
			return nil, fmt.Errorf("CategoryRepo - GetAll - rows.Scan: %w", err)
		}

		if iconURL != nil {
			e.IconURL = *iconURL
		}

		entities = append(entities, e)
	}

	return entities, nil
}

func (r *CategoryRepo) GetByID(ctx context.Context, id int) (entity.Category, error) {
	sql, args, err := r.Builder.
		Select("id, title, icon_url").
		From("categories").
		Where(map[string]interface{}{"id": id, "is_active": true}).
		ToSql()
	if err != nil {
		return entity.Category{}, fmt.Errorf("CategoryRepo - GetByID - r.Builder: %w", err)
	}

	var e entity.Category
	var iconURL *string
	err = r.Pool.QueryRow(ctx, sql, args...).Scan(&e.ID, &e.Title, &iconURL)
	if err != nil {
		return entity.Category{}, mapCategoryErr(err)
	}
	if iconURL != nil {
		e.IconURL = *iconURL
	}

	return e, nil
}

// Create -.
func (r *CategoryRepo) Create(ctx context.Context, input entity.CreateCategoryInput) (int, error) {
	sql, args, err := r.Builder.
		Insert("categories").
		Columns("title", "icon_url").
		Values(input.Title, input.IconURL).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("CategoryRepo - Create - r.Builder: %w", err)
	}

	var id int
	err = r.Pool.QueryRow(ctx, sql, args...).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("CategoryRepo - Create - r.Pool.QueryRow: %w", err)
	}

	return id, nil
}

// Update -.
func (r *CategoryRepo) Update(ctx context.Context, id int, input entity.UpdateCategoryInput) error {
	builder := r.Builder.Update("categories").Where(map[string]interface{}{"id": id, "is_active": true})

	if input.Title != nil {
		builder = builder.Set("title", *input.Title)
	}
	if input.IconURL != nil {
		builder = builder.Set("icon_url", *input.IconURL)
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("CategoryRepo - Update - r.Builder: %w", err)
	}

	res, err := r.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("CategoryRepo - Update - r.Pool.Exec: %w", err)
	}
	if res.RowsAffected() == 0 {
		return categoryerr.ErrNotFound
	}

	return nil
}

// Delete (Soft delete) -.
func (r *CategoryRepo) Delete(ctx context.Context, id int) error {
	sql, args, err := r.Builder.
		Update("categories").
		Set("is_active", false).
		Where(map[string]interface{}{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("CategoryRepo - Delete - r.Builder: %w", err)
	}

	res, err := r.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("CategoryRepo - Delete - r.Pool.Exec: %w", err)
	}
	if res.RowsAffected() == 0 {
		return categoryerr.ErrNotFound
	}

	return nil
}
