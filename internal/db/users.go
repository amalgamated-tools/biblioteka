package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// ErrEmailExists is returned when a user with the given email already exists.
var ErrEmailExists = errors.New("email already exists")

// User represents a row in the users table.
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	OIDCSubject  *string   `json:"-"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    Timestamp `json:"created_at"`
}

// scanUser scans a user row into a User struct.
func scanUser(row interface{ Scan(...any) error }) (*User, error) {
	var u User
	var oidcSubject sql.NullString
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &oidcSubject, &u.IsAdmin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	if oidcSubject.Valid {
		u.OIDCSubject = &oidcSubject.String
	}
	return &u, nil
}

const userColumns = `id, name, email, password_hash, oidc_subject, is_admin, created_at`

type userListQuery struct{}

func (userListQuery) table() string   { return "users" }
func (userListQuery) columns() string { return userColumns }
func (userListQuery) orderBy(d *DB) string {
	return d.dialectOrderBy("created_at", "ASC")
}

// CreateUser inserts a new user and returns it.
// Returns ErrEmailExists if the email is already registered.
// The first user created is automatically promoted to admin.
func (d *DB) CreateUser(ctx context.Context, name, email, passwordHash string) (*User, error) {
	slog.DebugContext(ctx, "db: creating user", slog.String(otelkeys.Email, email))
	var exists bool
	if err := d.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email) = LOWER($1))`, email).Scan(&exists); err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailExists
	}

	var userCount int
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return nil, err
	}
	isAdmin := userCount == 0

	return scanUser(d.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, is_admin) VALUES ($1, $2, $3, $4) RETURNING `+userColumns,
		name, email, passwordHash, isAdmin,
	))
}

// CreateOIDCUser inserts a new user authenticated via OIDC and returns it.
// OIDC users have an empty password_hash since they authenticate externally.
// Returns ErrEmailExists if the email is already registered.
// The first user created is automatically promoted to admin.
func (d *DB) CreateOIDCUser(ctx context.Context, name, email, oidcSubject string) (*User, error) {
	slog.DebugContext(ctx, "db: creating OIDC user", slog.String(otelkeys.Email, email))
	var exists bool
	if err := d.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email) = LOWER($1))`, email).Scan(&exists); err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailExists
	}

	var userCount int
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return nil, err
	}
	isAdmin := userCount == 0

	return scanUser(d.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, oidc_subject, is_admin) VALUES ($1, $2, '', $3, $4) RETURNING `+userColumns,
		name, email, oidcSubject, isAdmin,
	))
}

// GetUserByEmail returns a user by email, or sql.ErrNoRows if not found.
func (d *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	slog.DebugContext(ctx, "db: fetching user by email", slog.String(otelkeys.Email, email))
	return scanUser(d.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE LOWER(email) = LOWER($1)`,
		email,
	))
}

// GetUserByID returns a user by ID, or sql.ErrNoRows if not found.
func (d *DB) GetUserByID(ctx context.Context, id string) (*User, error) {
	slog.DebugContext(ctx, "db: fetching user by ID", slog.String(otelkeys.ID, id))
	return scanUser(d.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`,
		id,
	))
}

// GetUserByOIDCSubject returns a user by OIDC subject, or sql.ErrNoRows if not found.
func (d *DB) GetUserByOIDCSubject(ctx context.Context, subject string) (*User, error) {
	slog.DebugContext(ctx, "db: fetching user by OIDC subject")
	return scanUser(d.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE oidc_subject = $1`,
		subject,
	))
}

// LinkOIDCSubject sets the OIDC subject on an existing user.
func (d *DB) LinkOIDCSubject(ctx context.Context, userID, oidcSubject string) error {
	slog.DebugContext(ctx, "db: linking OIDC subject", slog.String(otelkeys.UserID, userID))
	return d.execAffected(ctx, `UPDATE users SET oidc_subject = $1 WHERE id = $2`, oidcSubject, userID)
}

// UpdatePassword updates a user's password hash.
func (d *DB) UpdatePassword(ctx context.Context, userID, newPasswordHash string) error {
	slog.DebugContext(ctx, "db: updating password", slog.String(otelkeys.UserID, userID))
	return d.execAffected(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, newPasswordHash, userID)
}

// IsAdmin returns true if the given user has the admin role.
func (d *DB) IsAdmin(ctx context.Context, userID string) (bool, error) {
	var admin bool
	err := d.QueryRowContext(ctx, `SELECT is_admin FROM users WHERE id = $1`, userID).Scan(&admin)
	if err != nil {
		return false, err
	}
	return admin, nil
}

// SetAdmin sets the is_admin flag on a user. Returns sql.ErrNoRows if user doesn't exist.
func (d *DB) SetAdmin(ctx context.Context, userID string, isAdmin bool) error {
	slog.DebugContext(ctx, "db: setting admin status",
		slog.String(otelkeys.UserID, userID),
		slog.Bool(otelkeys.IsAdmin, isAdmin),
	)
	return d.execAffected(ctx, `UPDATE users SET is_admin = $1 WHERE id = $2`, isAdmin, userID)
}

// ListUsers returns all users ordered by creation time.
func (d *DB) ListUsers(ctx context.Context) ([]User, error) {
	slog.DebugContext(ctx, "db: listing users")
	return listAll(ctx, d, userListQuery{}, scanUser)
}
