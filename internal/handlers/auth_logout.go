package handlers

import "net/http"

// Logout godoc
//
//	@Summary		Log out
//	@Description	Clears the authentication cookie
//	@Tags			Auth
//	@Produce		json
//	@Success		200	{object}	object{message=string}
//	@Failure		405	{object}	errorResponse
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !sameOrigin(r) {
		writeError(r.Context(), w, http.StatusForbidden, "invalid logout request origin")
		return
	}

	clearAuthCookie(w, h.SecureCookies)
	writeJSON(r.Context(), w, http.StatusOK, map[string]string{"message": "logged out"})
}
