package persistent

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/evrone/go-clean-template/internal/entity"
	"github.com/evrone/go-clean-template/pkg/postgres"
)

// UserRepo implements repo.UserRepo using Postgres.
type UserRepo struct {
	*postgres.Postgres
}

// NewUserRepo creates a new user repository.
func NewUserRepo(pg *postgres.Postgres) *UserRepo {
	return &UserRepo{pg}
}

// Create inserts a new user and sets its ID.
func (r *UserRepo) Create(ctx context.Context, u *entity.User) error {
	sql, args, err := r.Builder.
		Insert("users").
		Columns(
			"login",
			"email",
			"password_hash",
			"last_name",
			"first_name",
			"middle_name",
			"phone",
			"city",
			"street",
			"house",
			"apartment",
			"is_blocked",
			"is_admin",
		).
		Values(
			u.Login,
			u.Email,
			u.PasswordHash,
			u.LastName,
			u.FirstName,
			u.MiddleName,
			u.Phone,
			u.City,
			u.Street,
			u.House,
			u.Apartment,
			u.IsBlocked,
			u.IsAdmin,
		).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - Create - r.Builder: %w", err)
	}

	row := r.Pool.QueryRow(ctx, sql, args...)

	if err = row.Scan(&u.ID); err != nil {
		return fmt.Errorf("UserRepo - Create - row.Scan: %w", err)
	}

	return nil
}

// Delete removes a user by ID.
func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	sql, args, err := r.Builder.
		Delete("users").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - Delete - r.Builder: %w", err)
	}

	_, err = r.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("UserRepo - Delete - r.Pool.Exec: %w", err)
	}

	return nil
}

// GetByID returns a single user by ID.
func (r *UserRepo) GetByID(ctx context.Context, id int64) (entity.User, error) {
	sql, args, err := r.Builder.
		Select(
			"id",
			"login",
			"email",
			"password_hash",
			"last_name",
			"first_name",
			"middle_name",
			"phone",
			"city",
			"street",
			"house",
			"apartment",
			"is_blocked",
			"is_admin",
			"created_at",
			"updated_at",
		).
		From("users").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return entity.User{}, fmt.Errorf("UserRepo - GetByID - r.Builder: %w", err)
	}

	row := r.Pool.QueryRow(ctx, sql, args...)

	var u entity.User

	err = row.Scan(
		&u.ID,
		&u.Login,
		&u.Email,
		&u.PasswordHash,
		&u.LastName,
		&u.FirstName,
		&u.MiddleName,
		&u.Phone,
		&u.City,
		&u.Street,
		&u.House,
		&u.Apartment,
		&u.IsBlocked,
		&u.IsAdmin,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return entity.User{}, fmt.Errorf("UserRepo - GetByID - row.Scan: %w", err)
	}

	return u, nil
}

// List returns all users.
func (r *UserRepo) List(ctx context.Context) ([]entity.User, error) {
	sql, _, err := r.Builder.
		Select(
			"id",
			"login",
			"email",
			"password_hash",
			"last_name",
			"first_name",
			"middle_name",
			"phone",
			"city",
			"street",
			"house",
			"apartment",
			"is_blocked",
			"is_admin",
			"created_at",
			"updated_at",
		).
		From("users").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("UserRepo - List - r.Builder: %w", err)
	}

	rows, err := r.Pool.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - List - r.Pool.Query: %w", err)
	}
	defer rows.Close()

	const defaultCap = 64

	users := make([]entity.User, 0, defaultCap)

	for rows.Next() {
		var u entity.User

		err = rows.Scan(
			&u.ID,
			&u.Login,
			&u.Email,
			&u.PasswordHash,
			&u.LastName,
			&u.FirstName,
			&u.MiddleName,
			&u.Phone,
			&u.City,
			&u.Street,
			&u.House,
			&u.Apartment,
			&u.IsBlocked,
			&u.IsAdmin,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("UserRepo - List - rows.Scan: %w", err)
		}

		users = append(users, u)
	}

	return users, nil
}

// Update updates user fields by ID.
func (r *UserRepo) Update(ctx context.Context, u *entity.User) error {
	sql, args, err := r.Builder.
		Update("users").
		SetMap(map[string]any{
			"login":       u.Login,
			"email":       u.Email,
			"last_name":   u.LastName,
			"first_name":  u.FirstName,
			"middle_name": u.MiddleName,
			"phone":       u.Phone,
			"city":        u.City,
			"street":      u.Street,
			"house":       u.House,
			"apartment":   u.Apartment,
			"is_blocked":  u.IsBlocked,
			"is_admin":    u.IsAdmin,
		}).
		Where(squirrel.Eq{"id": u.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - Update - r.Builder: %w", err)
	}

	_, err = r.Pool.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("UserRepo - Update - r.Pool.Exec: %w", err)
	}

	return nil
}

