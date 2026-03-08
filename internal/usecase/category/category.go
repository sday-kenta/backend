// backend/internal/usecase/category/category.go

package category

import (
	"context"
	"fmt"

	"github.com/evrone/go-clean-template/internal/entity"
	"github.com/evrone/go-clean-template/internal/repo"
)

// UseCase -.
type UseCase struct {
	repo repo.CategoryRepo
}

// New -.
func New(r repo.CategoryRepo) *UseCase {
	return &UseCase{
		repo: r,
	}
}

// GetAll - получение списка активных рубрик
func (uc *UseCase) GetAll(ctx context.Context) ([]entity.Category, error) {
	categories, err := uc.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("CategoryUseCase - GetAll - uc.repo.GetAll: %w", err)
	}

	return categories, nil
}

func (uc *UseCase) GetByID(ctx context.Context, id int) (entity.Category, error) {
	category, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return entity.Category{}, fmt.Errorf("CategoryUseCase - GetByID - uc.repo.GetByID: %w", err)
	}
	return category, nil
}

// Create -.
func (uc *UseCase) Create(ctx context.Context, input entity.CreateCategoryInput) (entity.Category, error) {
	id, err := uc.repo.Create(ctx, input)
	if err != nil {
		return entity.Category{}, fmt.Errorf("CategoryUseCase - Create - uc.repo.Create: %w", err)
	}
	return uc.GetByID(ctx, id)
}

// Update -.
func (uc *UseCase) Update(ctx context.Context, id int, input entity.UpdateCategoryInput) (entity.Category, error) {
	err := uc.repo.Update(ctx, id, input)
	if err != nil {
		return entity.Category{}, fmt.Errorf("CategoryUseCase - Update - uc.repo.Update: %w", err)
	}
	return uc.GetByID(ctx, id)
}

// Delete -.
func (uc *UseCase) Delete(ctx context.Context, id int) error {
	err := uc.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("CategoryUseCase - Delete - uc.repo.Delete: %w", err)
	}
	return nil
}
