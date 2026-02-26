package category

import (
	"context"
	"fmt"

	"github.com/evrone/go-clean-template/internal/entity"
	"github.com/evrone/go-clean-template/internal/usecase"
)

// UseCase -.
type UseCase struct {
	repo usecase.CategoryRepo
}

// New -.
func New(r usecase.CategoryRepo) *UseCase {
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
