package db

import (
	"database/sql"
	"errors"
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

// CreateUser inserts a new user and returns it.
// Returns ErrEmailExists if the email is already registered.
// The first user created is automatically promoted to admin.
func (d *DB) CreateUser(name, email, passwordHash string) (*User, error) {
	var exists bool
	if err := d.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email) = LOWER($1))`, email).Scan(&exists); err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailExists
	}

	var userCount int
	if err := d.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return nil, err
	}
	isAdmin := userCount == 0

	return scanUser(d.QueryRow(
		`INSERT INTO users (name, email, password_hash, is_admin) VALUES ($1, $2, $3, $4) RETURNING `+userColumns,
		name, email, passwordHash, isAdmin,
	))
}

// CreateOIDCUser inserts a new user authenticated via OIDC and returns it.
// OIDC users have an empty password_hash since they authenticate externally.
// Returns ErrEmailExists if the email is already registered.
// The first user created is automatically promoted to admin.
func (d *DB) CreateOIDCUser(name, email, oidcSubject string) (*User, error) {
	var exists bool
	if err := d.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email) = LOWER($1))`, email).Scan(&exists); err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailExists
	}

	var userCount int
	if err := d.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return nil, err
	}
	isAdmin := userCount == 0

	return scanUser(d.QueryRow(
		`INSERT INTO users (name, email, password_hash, oidc_subject, is_admin) VALUES ($1, $2, '', $3, $4) RETURNING `+userColumns,
		name, email, oidcSubject, isAdmin,
	))
}

// GetUserByEmail returns a user by email, or sql.ErrNoRows if not found.
func (d *DB) GetUserByEmail(email string) (*User, error) {
	return scanUser(d.QueryRow(
		`SELECT `+userColumns+` FROM users WHERE LOWER(email) = LOWER($1)`,
		email,
	))
}

// GetUserByID returns a user by ID, or sql.ErrNoRows if not found.
func (d *DB) GetUserByID(id string) (*User, error) {
	return scanUser(d.QueryRow(
		`SELECT `+userColumns+` FROM users WHERE id = $1`,
		id,
	))
}

// GetUserByOIDCSubject returns a user by OIDC subject, or sql.ErrNoRows if not found.
func (d *DB) GetUserByOIDCSubject(subject string) (*User, error) {
	return scanUser(d.QueryRow(
		`SELECT `+userColumns+` FROM users WHERE oidc_subject = $1`,
		subject,
	))
}

// LinkOIDCSubject sets the OIDC subject on an existing user.
func (d *DB) LinkOIDCSubject(userID, oidcSubject string) error {
	res, err := d.Exec(`UPDATE users SET oidc_subject = $1 WHERE id = $2`, oidcSubject, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdatePassword updates a user's password hash.
func (d *DB) UpdatePassword(userID, newPasswordHash string) error {
	res, err := d.Exec(`UPDATE users SET password_hash = $1 WHERE id = $2`, newPasswordHash, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// IsAdmin returns true if the given user has the admin role.
func (d *DB) IsAdmin(userID string) (bool, error) {
	var admin bool
	err := d.QueryRow(`SELECT is_admin FROM users WHERE id = $1`, userID).Scan(&admin)
	if err != nil {
		return false, err
	}
	return admin, nil
}

// SetAdmin sets the is_admin flag on a user. Returns sql.ErrNoRows if user doesn't exist.
func (d *DB) SetAdmin(userID string, isAdmin bool) error {
	res, err := d.Exec(`UPDATE users SET is_admin = $1 WHERE id = $2`, isAdmin, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListUsers returns all users ordered by creation time.
func (d *DB) ListUsers() ([]User, error) {
	orderBy := "ORDER BY created_at ASC, rowid ASC"
	if d.Dialect == DialectPostgres {
		orderBy = "ORDER BY created_at ASC, id ASC"
	}
	rows, err := d.Query(`SELECT ` + userColumns + ` FROM users ` + orderBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}
