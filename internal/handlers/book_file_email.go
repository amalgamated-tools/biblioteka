package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"os"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/amalgamated-tools/biblioteka/internal/smtp"
)

// maxEmailAttachmentSize is the maximum file size (in bytes) allowed for
// email attachments. Files larger than this are rejected with 413.
const maxEmailAttachmentSize int64 = 25 * 1024 * 1024 // 25 MB

// emailBookFileRequest is the request body for the email book file endpoint.
type emailBookFileRequest struct {
	To string `json:"to"`
}

// handleEmailBookFile sends a book file as an email attachment.
//
//	@Summary		Email a book file
//	@Description	Sends a book file as an email attachment to the specified address
//	@Tags			BookFiles
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Book File ID"
//	@Param			body	body		emailBookFileRequest	true	"Recipient email address"
//	@Success		200		{object}	object{message=string}
//	@Failure		400		{object}	errorResponse
//	@Failure		401		{object}	errorResponse
//	@Failure		404		{object}	errorResponse
//	@Failure		413		{object}	errorResponse
//	@Failure		502		{object}	errorResponse
//	@Router			/book-files/{id}/email [post]
func (h *BookFileHandler) handleEmailBookFile(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req emailBookFileRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	to := strings.TrimSpace(req.To)
	if to == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "to address is required")
		return
	}
	if strings.ContainsAny(to, "\r\n") {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid email address")
		return
	}
	parsedTo, err := mail.ParseAddress(to)
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid email address")
		return
	}
	to = parsedTo.Address

	bf, err := h.DB.GetBookFile(r.Context(), id)
	if handleDBErr(r.Context(), w, err, "book file") {
		return
	}

	if bf.FileSize > maxEmailAttachmentSize {
		slog.WarnContext(r.Context(), "book file too large for email attachment",
			slog.String(otelkeys.BookFileID, id),
			slog.Int64(otelkeys.FileSize, bf.FileSize),
		)
		writeError(r.Context(), w, http.StatusRequestEntityTooLarge, "file is too large to email (maximum 25 MB)")
		return
	}

	allowed, pathErr := isBookFilePathAllowed(r.Context(), h.DB, bf.FilePath)
	if pathErr != nil {
		slog.ErrorContext(r.Context(), "failed to validate book file path",
			slog.String(otelkeys.BookFileID, id),
			slog.Any(otelkeys.Error, pathErr),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to validate file path")
		return
	}
	if !allowed {
		slog.WarnContext(r.Context(), "email blocked: file path outside library roots",
			slog.String(otelkeys.BookFileID, id),
			slog.String(otelkeys.Path, bf.FilePath),
		)
		writeError(r.Context(), w, http.StatusForbidden, "file path is outside allowed library directories")
		return
	}

	cfg := smtp.ResolveConfig(r.Context(), h.DB.GetSetting)
	if cfg.Host == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "SMTP is not configured")
		return
	}
	if cfg.EnvOverride && cfg.From == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "incomplete SMTP environment configuration: SMTP_HOST is set but SMTP_FROM is missing")
		return
	}

	params, err := smtp.ValidateForSend(cfg)
	if err != nil {
		var ve *smtp.ValidationError
		if errors.As(err, &ve) {
			writeError(r.Context(), w, http.StatusBadRequest, ve.Error())
			return
		}
		slog.ErrorContext(r.Context(), "unexpected SMTP validation error", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to validate SMTP configuration")
		return
	}

	data, err := readBookFileData(bf.FilePath)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to read book file",
			slog.String(otelkeys.BookFileID, id),
			slog.String(otelkeys.FilePath, bf.FilePath),
			slog.Any(otelkeys.Error, err),
		)
		if errors.Is(err, os.ErrNotExist) {
			writeError(r.Context(), w, http.StatusNotFound, "book file not found on disk")
		} else {
			writeError(r.Context(), w, http.StatusInternalServerError, "failed to read book file")
		}
		return
	}

	msg, err := smtp.BuildAttachmentMessage(params, to, bf.FileName, bf.FileType, data)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to build email message",
			slog.String(otelkeys.BookFileID, id),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to build email message")
		return
	}

	send := smtp.Send
	if h.SendMailFunc != nil {
		send = h.SendMailFunc
	}
	if err := send(r.Context(), params.Addr, params.Auth, params.From, to, msg, params.TLS); err != nil {
		slog.ErrorContext(r.Context(), "failed to send book file email",
			slog.String(otelkeys.BookFileID, id),
			slog.String(otelkeys.Email, redactEmail(to)),
			slog.Any(otelkeys.Error, err),
		)
		writeError(r.Context(), w, http.StatusBadGateway, "failed to send email")
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	logAudit(r.Context(), h.DB, userID, db.AuditActionBookFileEmailed, "book_file", bf.ID, map[string]any{
		"book_id":   bf.BookID,
		"file_name": bf.FileName,
		"to":        to,
	})

	slog.InfoContext(r.Context(), "book file emailed",
		slog.String(otelkeys.BookFileID, id),
		slog.String(otelkeys.Email, redactEmail(to)),
	)
	writeJSON(r.Context(), w, http.StatusOK, map[string]string{"message": "Email sent successfully"})
}
