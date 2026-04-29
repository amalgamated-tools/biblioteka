package server

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// seedInitialAdmin reads the INITIAL_ADMIN_EMAIL, INITIAL_ADMIN_PASSWORD, and
// optional INITIAL_ADMIN_NAME environment variables and, when the users table
// is empty, creates the first admin user with a bcrypt-hashed password. It is
// a no-op when either email or password env var is unset or when at least one
// user already exists. The operation is idempotent: if a concurrent instance
// seeds the user first, the duplicate error is treated as a no-op.
func (s *Server) seedInitialAdmin(ctx context.Context) error {
	email := strings.TrimSpace(os.Getenv("INITIAL_ADMIN_EMAIL"))
	password := os.Getenv("INITIAL_ADMIN_PASSWORD")
	if email == "" || password == "" {
		return nil
	}

	name := strings.TrimSpace(os.Getenv("INITIAL_ADMIN_NAME"))
	if name == "" {
		name = "Admin"
	}

	count, err := s.DB.CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		slog.InfoContext(ctx, "bootstrap: users already exist, skipping initial admin seeding",
			slog.String(otelkeys.Email, email),
		)
		return nil
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	// CreateUser automatically promotes the first user to admin (isAdmin = true
	// when the users table is empty), so no separate SetAdmin call is needed.
	user, err := s.DB.CreateUser(ctx, name, email, string(hash))
	if err != nil {
		// A concurrent instance may have seeded the admin between our CountUsers
		// check and this insert. Re-check: if a user now exists, treat as no-op.
		count, countErr := s.DB.CountUsers(ctx)
		if countErr != nil {
			return err
		}
		if count > 0 {
			slog.InfoContext(ctx, "bootstrap: initial admin seeding completed by another instance",
				slog.String(otelkeys.Email, email),
			)
			return nil
		}
		return err
	}

	slog.InfoContext(ctx, "bootstrap: initial admin user seeded",
		slog.String(otelkeys.UserID, user.ID),
		slog.String(otelkeys.Email, email),
	)
	return nil
}
