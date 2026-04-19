package server

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"golang.org/x/crypto/bcrypt"

	goauth "github.com/amalgamated-tools/goauth/auth"
)

// seedInitialAdmin reads the INITIAL_ADMIN_EMAIL and INITIAL_ADMIN_PASSWORD
// environment variables and, when the users table is empty, creates the first
// admin user with a bcrypt-hashed password. It is a no-op when either env var
// is unset or when at least one user already exists.
func (s *Server) seedInitialAdmin(ctx context.Context) error {
	email := strings.TrimSpace(os.Getenv("INITIAL_ADMIN_EMAIL"))
	password := strings.TrimSpace(os.Getenv("INITIAL_ADMIN_PASSWORD"))
	if email == "" || password == "" {
		return nil
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

	hash, err := bcrypt.GenerateFromPassword([]byte(password), goauth.BcryptCost)
	if err != nil {
		return err
	}

	// CreateUser automatically promotes the first user to admin (isAdmin = true
	// when the users table is empty), so no separate SetAdmin call is needed.
	user, err := s.DB.CreateUser(ctx, "Admin", email, string(hash))
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "bootstrap: initial admin user seeded",
		slog.String(otelkeys.UserID, user.ID),
		slog.String(otelkeys.Email, email),
	)
	return nil
}
