package persistent

import (
	"context"
	"fmt"

	"github.com/evrone/go-clean-template/internal/entity"
	"github.com/evrone/go-clean-template/pkg/postgres"
)

// CategoryRepo -.
type CategoryRepo struct {
	*postgres.Postgres
}

// NewCategoryRepo -.
func NewCategoryRepo(pg *postgres.Postgres) *CategoryRepo {
	return &CategoryRepo{pg}
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
