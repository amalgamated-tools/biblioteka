package handlers

import (
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/db"
)

type signupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type authResponse struct {
	Token string  `json:"token"`
	User  userDTO `json:"user"`
}

type userDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	OIDCLinked bool   `json:"oidc_linked"`
	IsAdmin    bool   `json:"is_admin"`
}

func toUserDTO(u *db.User) userDTO {
	return userDTO{
		ID:         u.ID,
		Name:       u.Name,
		Email:      u.Email,
		OIDCLinked: u.OIDCSubject != nil,
		IsAdmin:    u.IsAdmin,
	}
}

type updateProfileRequest struct {
	Name string `json:"name"`
}

func redactEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 1 {
		return "***"
	}
	return email[:1] + "***" + email[at:]
}
