package persistent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/usererr"
	"github.com/sday-kenta/backend/pkg/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// UserRepo implements repo.UserRepo using Postgres.
type UserRepo struct {
	*postgres.Postgres
}

// NewUserRepo creates a new user repository.
func NewUserRepo(pg *postgres.Postgres) *UserRepo {
	return &UserRepo{pg}
}

func mapPgError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return usererr.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "users_login_key", "users_login_unique":
			return usererr.ErrDuplicateLogin
		case "users_email_key", "users_email_unique":
			return usererr.ErrDuplicateEmail
		case "users_phone_key", "users_phone_unique":
			return usererr.ErrDuplicatePhone
		}
	}
	return err
}

// Create inserts a new user and sets its ID.
func (r *UserRepo) Create(ctx context.Context, u *entity.User) error {
	// Resolve role name to ID from roles table
	var roleID int64
	err := r.Pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = $1 LIMIT 1", u.Role).Scan(&roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return usererr.ErrInvalidRole
		}
		return fmt.Errorf("UserRepo - Create - roles lookup: %w", err)
	}

	query, args, err := r.Builder.
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
			"role_id",
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
			roleID,
		).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - Create - r.Builder: %w", err)
	}

	row := r.Pool.QueryRow(ctx, query, args...)

	if err = row.Scan(&u.ID); err != nil {
		return mapPgError(err)
	}

	return nil
}

// Delete removes a user by ID.
func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	query, args, err := r.Builder.
		Delete("users").
		Where(squirrel.Eq{"id": id}).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - Delete - r.Builder: %w", err)
	}

	row := r.Pool.QueryRow(ctx, query, args...)
	var deletedID int64
	if err = row.Scan(&deletedID); err != nil {
		return mapPgError(err)
	}

	return nil
}

// GetByID returns a single user by ID.
func (r *UserRepo) GetByID(ctx context.Context, id int64) (entity.User, error) {
	query, args, err := r.Builder.
		Select(
			"u.id",
			"u.login",
			"u.email",
			"u.password_hash",
			"u.last_name",
			"u.first_name",
			"u.middle_name",
			"u.phone",
			"u.city",
			"u.street",
			"u.house",
			"u.apartment",
			"u.avatar_url",
			"u.is_blocked",
			"r.name as role",
			"u.created_at",
			"u.updated_at",
		).
		From("users u").
		Join("roles r ON u.role_id = r.id").
		Where(squirrel.Eq{"u.id": id}).
		ToSql()
	if err != nil {
		return entity.User{}, fmt.Errorf("UserRepo - GetByID - r.Builder: %w", err)
	}

	row := r.Pool.QueryRow(ctx, query, args...)

	var (
		u          entity.User
		middleName sql.NullString
		apartment  sql.NullString
		avatarURL  sql.NullString
	)

	err = row.Scan(
		&u.ID,
		&u.Login,
		&u.Email,
		&u.PasswordHash,
		&u.LastName,
		&u.FirstName,
		&middleName,
		&u.Phone,
		&u.City,
		&u.Street,
		&u.House,
		&apartment,
		&avatarURL,
		&u.IsBlocked,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return entity.User{}, mapPgError(err)
	}

	if middleName.Valid {
		u.MiddleName = middleName.String
	}
	if apartment.Valid {
		u.Apartment = apartment.String
	}
	if avatarURL.Valid {
		u.AvatarURL = avatarURL.String
	}

	return u, nil
}

// GetByIdentifier finds a user by login OR email OR phone.
// For login/email comparison is case-insensitive, phone is exact match.
func (r *UserRepo) GetByIdentifier(ctx context.Context, identifier string) (entity.User, error) {
	const q = `
SELECT
  u.id,
  u.login,
  u.email,
  u.password_hash,
  u.last_name,
  u.first_name,
  u.middle_name,
  u.phone,
  u.city,
  u.street,
  u.house,
  u.apartment,
  u.avatar_url,
  u.is_blocked,
  r.name as role,
  u.created_at,
  u.updated_at
FROM users u
JOIN roles r ON u.role_id = r.id
WHERE lower(u.login) = lower($1)
   OR lower(u.email) = lower($1)
   OR u.phone = $1
LIMIT 1`

	row := r.Pool.QueryRow(ctx, q, identifier)

	var (
		u          entity.User
		middleName sql.NullString
		apartment  sql.NullString
		avatarURL  sql.NullString
	)

	if err := row.Scan(
		&u.ID,
		&u.Login,
		&u.Email,
		&u.PasswordHash,
		&u.LastName,
		&u.FirstName,
		&middleName,
		&u.Phone,
		&u.City,
		&u.Street,
		&u.House,
		&apartment,
		&avatarURL,
		&u.IsBlocked,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
	); err != nil {
		return entity.User{}, mapPgError(err)
	}

	if middleName.Valid {
		u.MiddleName = middleName.String
	}
	if apartment.Valid {
		u.Apartment = apartment.String
	}
	if avatarURL.Valid {
		u.AvatarURL = avatarURL.String
	}

	return u, nil
}

func (r *UserRepo) CreateEmailVerificationCode(
	ctx context.Context,
	email, purpose, code string,
	expiresAtUnix int64,
) error {
	expiresAt := time.Unix(expiresAtUnix, 0).UTC()

	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("UserRepo - CreateEmailVerificationCode - Begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Keep a single active code per (email,purpose).
	if _, err = tx.Exec(
		ctx,
		"DELETE FROM email_verification_codes WHERE email = $1 AND purpose = $2 AND consumed_at IS NULL",
		email,
		purpose,
	); err != nil {
		return fmt.Errorf("UserRepo - CreateEmailVerificationCode - delete active: %w", err)
	}

	if _, err = tx.Exec(
		ctx,
		`INSERT INTO email_verification_codes (email, purpose, code, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		email,
		purpose,
		code,
		expiresAt,
	); err != nil {
		return fmt.Errorf("UserRepo - CreateEmailVerificationCode - insert: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("UserRepo - CreateEmailVerificationCode - Commit: %w", err)
	}

	return nil
}

func (r *UserRepo) CheckEmailVerificationCode(
	ctx context.Context,
	email, purpose, code string,
	nowUnix int64,
) error {
	now := time.Unix(nowUnix, 0).UTC()

	var (
		dbCode    string
		expiresAt time.Time
	)

	err := r.Pool.QueryRow(
		ctx,
		`SELECT code, expires_at
		 FROM email_verification_codes
		 WHERE email = $1 AND purpose = $2 AND consumed_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT 1`,
		email,
		purpose,
	).Scan(&dbCode, &expiresAt)
	if err != nil {
		return mapPgError(err)
	}

	if now.After(expiresAt) {
		return usererr.ErrCodeExpired
	}
	if dbCode != code {
		return usererr.ErrInvalidCode
	}

	return nil
}

func (r *UserRepo) ConsumeEmailVerificationCode(
	ctx context.Context,
	email, purpose, code string,
	nowUnix int64,
) error {
	now := time.Unix(nowUnix, 0).UTC()

	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("UserRepo - ConsumeEmailVerificationCode - Begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		dbCode    string
		expiresAt time.Time
	)

	err = tx.QueryRow(
		ctx,
		`SELECT code, expires_at
		 FROM email_verification_codes
		 WHERE email = $1 AND purpose = $2 AND consumed_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT 1
		 FOR UPDATE`,
		email,
		purpose,
	).Scan(&dbCode, &expiresAt)
	if err != nil {
		return mapPgError(err)
	}

	if now.After(expiresAt) {
		return usererr.ErrCodeExpired
	}
	if dbCode != code {
		return usererr.ErrInvalidCode
	}

	if _, err = tx.Exec(
		ctx,
		`UPDATE email_verification_codes
		 SET consumed_at = $1
		 WHERE email = $2 AND purpose = $3 AND consumed_at IS NULL`,
		now,
		email,
		purpose,
	); err != nil {
		return fmt.Errorf("UserRepo - ConsumeEmailVerificationCode - update: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("UserRepo - ConsumeEmailVerificationCode - Commit: %w", err)
	}

	return nil
}

func (r *UserRepo) UpdatePasswordHashByEmail(ctx context.Context, email, passwordHash string) error {
	var updatedID int64
	err := r.Pool.QueryRow(
		ctx,
		`UPDATE users
		 SET password_hash = $1, updated_at = NOW()
		 WHERE lower(email) = lower($2)
		 RETURNING id`,
		passwordHash,
		email,
	).Scan(&updatedID)
	if err != nil {
		return mapPgError(err)
	}

	return nil
}

// List returns all users.
func (r *UserRepo) List(ctx context.Context) ([]entity.User, error) {
	query, _, err := r.Builder.
		Select(
			"u.id",
			"u.login",
			"u.email",
			"u.password_hash",
			"u.last_name",
			"u.first_name",
			"u.middle_name",
			"u.phone",
			"u.city",
			"u.street",
			"u.house",
			"u.apartment",
			"u.avatar_url",
			"u.is_blocked",
			"r.name as role",
			"u.created_at",
			"u.updated_at",
		).
		From("users u").
		Join("roles r ON u.role_id = r.id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("UserRepo - List - r.Builder: %w", err)
	}

	rows, err := r.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("UserRepo - List - r.Pool.Query: %w", err)
	}
	defer rows.Close()

	const defaultCap = 64

	users := make([]entity.User, 0, defaultCap)

	for rows.Next() {
		var (
			u          entity.User
			middleName sql.NullString
			apartment  sql.NullString
			avatarURL  sql.NullString
		)

		err = rows.Scan(
			&u.ID,
			&u.Login,
			&u.Email,
			&u.PasswordHash,
			&u.LastName,
			&u.FirstName,
			&middleName,
			&u.Phone,
			&u.City,
			&u.Street,
			&u.House,
			&apartment,
			&avatarURL,
			&u.IsBlocked,
			&u.Role,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("UserRepo - List - rows.Scan: %w", err)
		}

		if middleName.Valid {
			u.MiddleName = middleName.String
		}
		if apartment.Valid {
			u.Apartment = apartment.String
		}
		if avatarURL.Valid {
			u.AvatarURL = avatarURL.String
		}

		users = append(users, u)
	}

	return users, nil
}

// Update updates user fields by ID.
func (r *UserRepo) Update(ctx context.Context, u *entity.User) error {
	query, args, err := r.Builder.
		Update("users").
		Set("login", u.Login).
		Set("email", u.Email).
		Set("last_name", u.LastName).
		Set("first_name", u.FirstName).
		Set("middle_name", u.MiddleName).
		Set("phone", u.Phone).
		Set("city", u.City).
		Set("street", u.Street).
		Set("house", u.House).
		Set("apartment", u.Apartment).
		Set("is_blocked", u.IsBlocked).
		Set("role_id", squirrel.Expr("(SELECT id FROM roles WHERE name = ? LIMIT 1)", u.Role)).
		Where(squirrel.Eq{"id": u.ID}).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - Update - r.Builder: %w", err)
	}

	var updatedID int64
	if err = r.Pool.QueryRow(ctx, query, args...).Scan(&updatedID); err != nil {
		return mapPgError(err)
	}

	return nil
}

// UpdateAvatar updates the avatar identifier/URL for a user by ID.
func (r *UserRepo) UpdateAvatar(ctx context.Context, id int64, avatarURL string) error {
	query, args, err := r.Builder.
		Update("users").
		Set("avatar_url", avatarURL).
		Where(squirrel.Eq{"id": id}).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("UserRepo - UpdateAvatar - r.Builder: %w", err)
	}

	var updatedID int64
	if err = r.Pool.QueryRow(ctx, query, args...).Scan(&updatedID); err != nil {
		return mapPgError(err)
	}

	return nil
}
