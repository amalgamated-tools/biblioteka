package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/db"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// koboSyncPageSize is the maximum number of books returned per sync request.
const koboSyncPageSize = 100

// KoboHandler handles Kobo sync device API endpoints and Kobo token management.
type KoboHandler struct {
	DB *db.DB
}

// ---- Token management (JWT-authenticated) ----

// HandleKoboTokens handles GET/POST /api/kobo/tokens.
func (h *KoboHandler) HandleKoboTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listKoboTokens(w, r)
	case http.MethodPost:
		h.createKoboToken(w, r)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// HandleKoboToken handles DELETE /api/kobo/tokens/{id}.
func (h *KoboHandler) HandleKoboToken(w http.ResponseWriter, r *http.Request) {
	id, ok := extractPathID(r.URL.Path, "/api/kobo/tokens/")
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid token ID")
		return
	}
	switch r.Method {
	case http.MethodDelete:
		h.deleteKoboToken(w, r, id)
	default:
		writeError(r.Context(), w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *KoboHandler) listKoboTokens(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	tokens, err := h.DB.ListKoboTokens(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to list kobo tokens", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to list Kobo tokens")
		return
	}
	if tokens == nil {
		tokens = []db.KoboToken{}
	}
	writeJSON(r.Context(), w, http.StatusOK, tokens)
}

type koboTokenCreateRequest struct {
	Name string `json:"name"`
}

func (h *KoboHandler) createKoboToken(w http.ResponseWriter, r *http.Request) {
	var req koboTokenCreateRequest
	if !decodeJSON(r, w, &req) {
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "name is required")
		return
	}
	if len(name) > 100 {
		writeError(r.Context(), w, http.StatusBadRequest, "name must be at most 100 characters")
		return
	}

	// Generate a random 32-byte hex token (64 hex chars).
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		slog.ErrorContext(r.Context(), "failed to generate random bytes", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to generate Kobo token")
		return
	}
	token := hex.EncodeToString(raw)

	userID := auth.UserIDFromContext(r.Context())
	koboToken, err := h.DB.CreateKoboToken(r.Context(), userID, name, token)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to create kobo token", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to create Kobo token")
		return
	}

	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionKoboTokenCreated, "kobo_token", koboToken.ID, map[string]any{"name": name}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(r.Context(), w, http.StatusCreated, koboToken)
}

func (h *KoboHandler) deleteKoboToken(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.UserIDFromContext(r.Context())

	token, err := h.DB.GetKoboToken(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusNotFound, "Kobo token not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to fetch kobo token", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete Kobo token")
		return
	}

	if err := h.DB.DeleteKoboToken(r.Context(), id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(r.Context(), w, http.StatusNotFound, "Kobo token not found")
			return
		}
		slog.ErrorContext(r.Context(), "failed to delete kobo token", slog.Any(otelkeys.Error, err))
		writeError(r.Context(), w, http.StatusInternalServerError, "failed to delete Kobo token")
		return
	}

	if err := h.DB.CreateAuditLog(r.Context(), userID, db.AuditActionKoboTokenDeleted, "kobo_token", id, map[string]any{"name": token.Name}); err != nil {
		slog.WarnContext(r.Context(), "failed to write audit log", slog.Any(otelkeys.Error, err))
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---- Kobo device API ----

// HandleKobo is the entry point for all Kobo device requests at /kobo/{token}/...
// It validates the token, injects the user ID, and dispatches to sub-handlers.
func (h *KoboHandler) HandleKobo(w http.ResponseWriter, r *http.Request) {
	// Parse token and sub-path from /kobo/{token}/...
	rest := strings.TrimPrefix(r.URL.Path, "/kobo/")
	slashIdx := strings.Index(rest, "/")
	var tokenValue, subPath string
	if slashIdx < 0 {
		tokenValue = rest
	} else {
		tokenValue = rest[:slashIdx]
		subPath = rest[slashIdx:] // starts with /
	}

	if tokenValue == "" {
		writeKoboJSON(w, http.StatusOK, map[string]any{})
		return
	}

	// Validate the Kobo token.
	koboToken, err := h.DB.GetKoboTokenByToken(r.Context(), tokenValue)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return an empty 401 to keep the Kobo device from showing an error.
			writeKoboJSON(w, http.StatusUnauthorized, map[string]any{})
			return
		}
		slog.ErrorContext(r.Context(), "failed to look up kobo token", slog.Any(otelkeys.Error, err))
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}

	// Inject user ID into the request context.
	ctx := auth.ContextWithUserID(r.Context(), koboToken.UserID)
	r = r.WithContext(ctx)

	slog.DebugContext(r.Context(), "kobo device request",
		slog.String(otelkeys.UserID, koboToken.UserID),
		slog.String(otelkeys.Path, subPath),
	)

	switch {
	case subPath == "" || subPath == "/":
		writeKoboJSON(w, http.StatusOK, map[string]any{})

	case subPath == "/v1/initialization":
		h.handleInit(w, r, tokenValue)

	case subPath == "/v1/auth/device" || subPath == "/v1/auth/refresh" || subPath == "/v1/auth/exchange":
		h.handleAuth(w, r)

	case subPath == "/v1/library/sync":
		h.handleSync(w, r, tokenValue)

	case strings.HasPrefix(subPath, "/v1/library/") && strings.HasSuffix(subPath, "/metadata"):
		h.handleBookMetadata(w, r, subPath, tokenValue)

	case strings.HasPrefix(subPath, "/v1/library/") && strings.HasSuffix(subPath, "/state"):
		h.handleBookState(w, r, subPath)

	case strings.HasPrefix(subPath, "/download/"):
		h.handleDownload(w, r, subPath)

	case strings.HasPrefix(subPath, "/covers/"):
		h.handleCoverImage(w, r, subPath)

	case subPath == "/v1/user/loyalty/benefits":
		writeKoboJSON(w, http.StatusOK, map[string]any{"Benefits": map[string]any{}})

	case subPath == "/v1/analytics/gettests":
		userKey := r.Header.Get("X-Kobo-userkey")
		writeKoboJSON(w, http.StatusOK, map[string]any{
			"Result":  "Success",
			"TestKey": userKey,
			"Tests":   map[string]any{},
		})

	default:
		// Return empty JSON for all other unimplemented endpoints so the device
		// continues its sync flow without errors.
		writeKoboJSON(w, http.StatusOK, map[string]any{})
	}
}

// writeKoboJSON writes a JSON response with the content type expected by Kobo devices.
func writeKoboJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// schemeAndHost returns the scheme and host for building absolute URLs.
func schemeAndHost(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		host = fwd
	}
	return scheme + "://" + host
}

// handleInit handles GET /kobo/{token}/v1/initialization.
// It returns the Kobo API resource map pointing back to this server.
func (h *KoboHandler) handleInit(w http.ResponseWriter, r *http.Request, tokenValue string) {
	base := schemeAndHost(r)
	tb := base + "/kobo/" + tokenValue // token base URL

	resources := map[string]any{
		"account_page":           "https://www.kobo.com/account/settings",
		"account_page_rakuten":   "https://my.rakuten.co.jp/",
		"add_entitlement":        tb + "/v1/library/{RevisionIds}",
		"affiliate":              tb + "/v1/affiliate",
		"assets":                 tb + "/v1/assets",
		"audiobook":              tb + "/v1/products/audiobooks/{ProductId}",
		"audiobook_landing_page": "https://www.kobo.com/ebooks",
		"audiobook_subscription_orange_deal_inclusion_url": "https://authorize.kobo.com/inclusion",
		"authorproduct_recommendations":                    tb + "/v1/products/books/authors/recommendations",
		"autocomplete":                                     tb + "/v1/products/autocomplete",
		"blackstone_header": map[string]any{
			"key":   "x-amz-request-payer",
			"value": "requester",
		},
		"book":                            tb + "/v1/products/books/{ProductId}",
		"book_landing_page":               "https://www.kobo.com/ebooks",
		"browse_history":                  tb + "/v1/user/browsehistory",
		"categories":                      tb + "/v1/categories",
		"checkout_borrowed_book":          tb + "/v1/library/borrow",
		"client_authd_referral":           "https://authorize.kobo.com/api/AuthenticatedReferral/client/v1/getLink",
		"configuration_data":              tb + "/v1/configuration",
		"content_access_book":             tb + "/v1/products/books/{ProductId}/access",
		"daily_deal":                      tb + "/v1/products/dailydeal",
		"deals":                           tb + "/v1/deals",
		"delete_entitlement":              tb + "/v1/library/{Ids}",
		"delete_tag":                      tb + "/v1/library/tags/{TagId}",
		"delete_tag_items":                tb + "/v1/library/tags/{TagId}/items/delete",
		"device_auth":                     tb + "/v1/auth/device",
		"device_refresh":                  tb + "/v1/auth/refresh",
		"dictionary_host":                 "https://ereaderfiles.kobo.com",
		"discovery_host":                  "https://discovery.kobobooks.com",
		"exchange_auth":                   tb + "/v1/auth/exchange",
		"external_book":                   tb + "/v1/products/books/external/{Ids}",
		"featured_list":                   tb + "/v1/products/featured/{FeaturedListId}",
		"featured_lists":                  tb + "/v1/products/featured",
		"get_download_keys":               tb + "/v1/library/downloadkeys",
		"get_download_link":               tb + "/v1/library/downloadlink",
		"get_tests_request":               tb + "/v1/analytics/gettests",
		"gpb_flow_enabled":                "False",
		"help_page":                       "https://www.kobo.com/help",
		"image_host":                      base,
		"image_url_quality_template":      base + "/kobo/" + tokenValue + "/covers/{ImageId}/{Width}/{Height}/{Quality}/{IsGreyscale}/image.jpg",
		"image_url_template":              base + "/kobo/" + tokenValue + "/covers/{ImageId}/{Width}/{Height}/false/image.jpg",
		"instapaper_enabled":              "False",
		"library_metadata":                tb + "/v1/library/{Ids}",
		"library_prices":                  tb + "/v1/library/{Ids}/prices",
		"library_stack":                   tb + "/v1/user/library/stack",
		"library_sync":                    tb + "/v1/library/sync",
		"new_entitlement":                 tb + "/v1/library/{RevisionId}",
		"new_recommendation":              tb + "/v1/user/recommendations",
		"new_wishlist_item":               tb + "/v1/user/wishlist",
		"partner_agreements":              tb + "/v1/user/partneragreements",
		"product_nextread":                tb + "/v1/products/{ProductId}/nextread",
		"product_prices":                  tb + "/v1/products/{ProductIds}/prices",
		"product_recommendations":         tb + "/v1/products/{ProductId}/recommendations",
		"product_reviews":                 tb + "/v1/products/{ProductId}/reviews",
		"products_v2":                     tb + "/v2/products",
		"reading_services_host":           "https://readingservices.kobo.com",
		"recommendations":                 tb + "/v1/user/recommendations",
		"review_sentiment":                tb + "/v1/user/reviews/ratings",
		"search":                          tb + "/v1/products/search",
		"social_authorization":            "https://social.kobo.com",
		"store_home":                      "https://www.kobo.com/ebooks",
		"store_host":                      "https://www.kobo.com",
		"tag_items":                       tb + "/v1/library/tags/{TagId}/items",
		"tag_list":                        tb + "/v1/library/tags",
		"taste_profile":                   tb + "/v1/products/tasteprofile",
		"update_accessibility_to_preview": tb + "/v1/user/library/accessibility/{EntitlementId}",
		"user_loyalty_benefits":           tb + "/v1/user/loyalty/benefits",
		"user_platform":                   tb + "/v1/user/platform",
		"user_profile":                    tb + "/v1/user/profile",
		"user_ratings":                    tb + "/v1/user/ratings",
		"user_recommendations":            tb + "/v1/user/recommendations",
		"user_reviews":                    tb + "/v1/user/reviews",
		"user_wishlist":                   tb + "/v1/user/wishlist",
		"userguide_host":                  "https://ereaderfiles.kobo.com",
		"wishlist_list":                   tb + "/v1/user/wishlist",
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// x-kobo-apitoken is required by Kobo devices; "e30=" is base64("{}")
	w.Header().Set("x-kobo-apitoken", "e30=")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"Resources": resources})
}

// handleAuth handles POST /kobo/{token}/v1/auth/device and /v1/auth/refresh.
// The Kobo device doesn't use these tokens for our server, so we return dummy values.
func (h *KoboHandler) handleAuth(w http.ResponseWriter, r *http.Request) {
	var userKey string
	if r.Body != nil {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			if k, ok := body["UserKey"].(string); ok {
				userKey = k
			}
		}
	}

	buf := make([]byte, 24)
	_, _ = rand.Read(buf)
	accessToken := base64.StdEncoding.EncodeToString(buf)
	_, _ = rand.Read(buf)
	refreshToken := base64.StdEncoding.EncodeToString(buf)

	writeKoboJSON(w, http.StatusOK, map[string]any{
		"AccessToken":  accessToken,
		"RefreshToken": refreshToken,
		"TokenType":    "Bearer",
		"TrackingId":   koboRandomUUID(),
		"UserKey":      userKey,
	})
}

// koboRandomUUID generates a random UUID v4-like string.
func koboRandomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// ---- Kobo sync token ----

// koboSyncToken tracks the high-water marks for the Kobo sync.
type koboSyncToken struct {
	BooksLastModified        time.Time
	ReadingStateLastModified time.Time
}

type koboSyncTokenPayload struct {
	Version string         `json:"version"`
	Data    map[string]any `json:"data"`
}

func parseKoboSyncToken(header string) koboSyncToken {
	if header == "" {
		return koboSyncToken{}
	}
	decoded, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return koboSyncToken{}
	}
	var payload koboSyncTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return koboSyncToken{}
	}
	var result koboSyncToken
	if s, ok := payload.Data["BooksLastModified"].(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			result.BooksLastModified = t
		}
	}
	if s, ok := payload.Data["ReadingStateLastModified"].(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			result.ReadingStateLastModified = t
		}
	}
	return result
}

func encodeKoboSyncToken(tok koboSyncToken) string {
	payload := koboSyncTokenPayload{
		Version: "1-0-0",
		Data: map[string]any{
			"BooksLastModified":        tok.BooksLastModified.UTC().Format(time.RFC3339),
			"ReadingStateLastModified": tok.ReadingStateLastModified.UTC().Format(time.RFC3339),
		},
	}
	b, _ := json.Marshal(payload)
	return base64.StdEncoding.EncodeToString(b)
}

// ---- Sync ----

// handleSync handles GET /kobo/{token}/v1/library/sync.
func (h *KoboHandler) handleSync(w http.ResponseWriter, r *http.Request, tokenValue string) {
	ctx := r.Context()
	userID := auth.UserIDFromContext(ctx)

	syncToken := parseKoboSyncToken(r.Header.Get("x-kobo-synctoken"))
	slog.InfoContext(ctx, "kobo library sync request", slog.String(otelkeys.UserID, userID))

	// Fetch one more than the page size to detect whether there are more results.
	books, err := h.DB.ListBooksModifiedSince(ctx, syncToken.BooksLastModified, koboSyncPageSize+1)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list books for kobo sync", slog.Any(otelkeys.Error, err))
		writeKoboJSON(w, http.StatusInternalServerError, []any{})
		return
	}

	hasMore := len(books) > koboSyncPageSize
	if hasMore {
		books = books[:koboSyncPageSize]
	}

	// Batch-load authors, files, and series for the returned books.
	bookIDs := make([]string, len(books))
	for i, b := range books {
		bookIDs[i] = b.ID
	}

	authorsByBook, err := h.DB.GetAuthorsForBooks(ctx, bookIDs)
	if err != nil {
		slog.WarnContext(ctx, "failed to batch-load authors for kobo sync", slog.Any(otelkeys.Error, err))
	}
	filesByBook, err := h.DB.GetFilesForBooks(ctx, bookIDs)
	if err != nil {
		slog.WarnContext(ctx, "failed to batch-load files for kobo sync", slog.Any(otelkeys.Error, err))
	}
	seriesByBook, err := h.DB.GetSeriesForBooks(ctx, bookIDs)
	if err != nil {
		slog.WarnContext(ctx, "failed to batch-load series for kobo sync", slog.Any(otelkeys.Error, err))
	}

	// Fetch reading states changed since last sync.
	readingStates, err := h.DB.ListKoboReadingStatesSince(ctx, userID, syncToken.ReadingStateLastModified)
	if err != nil {
		slog.WarnContext(ctx, "failed to list reading states for kobo sync", slog.Any(otelkeys.Error, err))
	}
	readingStatesByBook := make(map[string]*db.KoboReadingState, len(readingStates))
	for i := range readingStates {
		readingStatesByBook[readingStates[i].BookID] = &readingStates[i]
	}

	base := schemeAndHost(r)
	var newBooksLastModified time.Time
	newReadingStateLastModified := syncToken.ReadingStateLastModified

	syncResults := make([]any, 0, len(books))
	for _, book := range books {
		bk := book // copy
		files := filesByBook[bk.ID]
		authors := authorsByBook[bk.ID]
		series := seriesByBook[bk.ID]

		downloadURLs := koboDownloadURLs(base, tokenValue, bk.ID, files)
		if len(downloadURLs) == 0 {
			// Skip books that have no downloadable files.
			continue
		}

		if bk.UpdatedAt.After(newBooksLastModified) {
			newBooksLastModified = bk.UpdatedAt.Time
		}

		entitlement := map[string]any{
			"BookEntitlement": koboBookEntitlement(&bk),
			"BookMetadata":    koboBookMetadata(&bk, authors, series, downloadURLs),
		}

		if rs, ok := readingStatesByBook[bk.ID]; ok {
			if rs.UpdatedAt.After(syncToken.ReadingStateLastModified) {
				entitlement["ReadingState"] = koboReadingStateResponse(rs)
				if rs.UpdatedAt.After(newReadingStateLastModified) {
					newReadingStateLastModified = rs.UpdatedAt.Time
				}
			}
		}

		if bk.CreatedAt.After(syncToken.BooksLastModified) {
			syncResults = append(syncResults, map[string]any{"NewEntitlement": entitlement})
		} else {
			syncResults = append(syncResults, map[string]any{"ChangedEntitlement": entitlement})
		}
	}

	newSyncToken := koboSyncToken{
		BooksLastModified:        syncToken.BooksLastModified,
		ReadingStateLastModified: newReadingStateLastModified,
	}
	if newBooksLastModified.After(syncToken.BooksLastModified) {
		newSyncToken.BooksLastModified = newBooksLastModified
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("x-kobo-synctoken", encodeKoboSyncToken(newSyncToken))
	if hasMore {
		w.Header().Set("x-kobo-sync", "continue")
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(syncResults)
}

// ---- Metadata ----

// handleBookMetadata handles GET /kobo/{token}/v1/library/{uuid}/metadata.
func (h *KoboHandler) handleBookMetadata(w http.ResponseWriter, r *http.Request, subPath, tokenValue string) {
	bookID := strings.TrimSuffix(strings.TrimPrefix(subPath, "/v1/library/"), "/metadata")
	if bookID == "" {
		writeKoboJSON(w, http.StatusNotFound, map[string]any{})
		return
	}

	book, err := h.DB.GetBook(r.Context(), bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeKoboJSON(w, http.StatusNotFound, map[string]any{})
			return
		}
		slog.ErrorContext(r.Context(), "failed to fetch book for kobo metadata", slog.Any(otelkeys.Error, err))
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}

	authors, _ := h.DB.GetBookAuthors(r.Context(), bookID)
	files, _ := h.DB.ListBookFiles(r.Context(), bookID)
	series, _ := h.DB.GetBookSeries(r.Context(), bookID)

	base := schemeAndHost(r)
	downloadURLs := koboDownloadURLs(base, tokenValue, bookID, files)
	writeKoboJSON(w, http.StatusOK, []any{koboBookMetadata(book, authors, series, downloadURLs)})
}

// ---- Reading state ----

// handleBookState handles GET/PUT /kobo/{token}/v1/library/{uuid}/state.
func (h *KoboHandler) handleBookState(w http.ResponseWriter, r *http.Request, subPath string) {
	bookID := strings.TrimSuffix(strings.TrimPrefix(subPath, "/v1/library/"), "/state")
	userID := auth.UserIDFromContext(r.Context())

	switch r.Method {
	case http.MethodGet:
		h.getBookState(w, r, userID, bookID)
	case http.MethodPut:
		h.updateBookState(w, r, userID, bookID)
	default:
		writeKoboJSON(w, http.StatusOK, []any{})
	}
}

func (h *KoboHandler) getBookState(w http.ResponseWriter, r *http.Request, userID, bookID string) {
	book, err := h.DB.GetBook(r.Context(), bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeKoboJSON(w, http.StatusNotFound, map[string]any{})
			return
		}
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}

	state, err := h.DB.GetKoboReadingState(r.Context(), userID, bookID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}
	if state == nil {
		// Return a default "ReadyToRead" state.
		state = &db.KoboReadingState{
			BookID:    bookID,
			Status:    "ReadyToRead",
			CreatedAt: book.CreatedAt,
			UpdatedAt: book.UpdatedAt,
		}
	}

	writeKoboJSON(w, http.StatusOK, []any{koboReadingStateResponse(state)})
}

type koboStateUpdateRequest struct {
	ReadingStates []struct {
		CurrentBookmark *struct {
			ProgressPercent              *float64 `json:"ProgressPercent"`
			ContentSourceProgressPercent *float64 `json:"ContentSourceProgressPercent"`
			Location                     *struct {
				Value  string `json:"Value"`
				Type   string `json:"Type"`
				Source string `json:"Source"`
			} `json:"Location"`
		} `json:"CurrentBookmark"`
		Statistics *struct {
			SpentReadingMinutes  *int `json:"SpentReadingMinutes"`
			RemainingTimeMinutes *int `json:"RemainingTimeMinutes"`
		} `json:"Statistics"`
		StatusInfo *struct {
			Status string `json:"Status"`
		} `json:"StatusInfo"`
	} `json:"ReadingStates"`
}

func (h *KoboHandler) updateBookState(w http.ResponseWriter, r *http.Request, userID, bookID string) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req koboStateUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.ReadingStates) == 0 {
		writeKoboJSON(w, http.StatusBadRequest, map[string]any{"RequestResult": "BadRequest"})
		return
	}

	rs := req.ReadingStates[0]

	status := "ReadyToRead"
	if rs.StatusInfo != nil && rs.StatusInfo.Status != "" {
		status = rs.StatusInfo.Status
	}

	var percentRead *float64
	var locationValue, locationType, locationSource *string
	if rs.CurrentBookmark != nil {
		percentRead = rs.CurrentBookmark.ProgressPercent
		if rs.CurrentBookmark.Location != nil {
			lv := rs.CurrentBookmark.Location.Value
			lt := rs.CurrentBookmark.Location.Type
			ls := rs.CurrentBookmark.Location.Source
			locationValue = &lv
			locationType = &lt
			locationSource = &ls
		}
	}

	state, err := h.DB.UpsertKoboReadingState(r.Context(), userID, bookID, status, percentRead, locationValue, locationType, locationSource)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to upsert kobo reading state", slog.Any(otelkeys.Error, err))
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{"RequestResult": "ServerError"})
		return
	}

	updated := state.UpdatedAt.UTC().Format(time.RFC3339)
	writeKoboJSON(w, http.StatusOK, map[string]any{
		"RequestResult": "Success",
		"UpdateResults": []any{
			map[string]any{
				"EntitlementId":         bookID,
				"CurrentBookmarkResult": map[string]any{"Result": "Success"},
				"StatisticsResult":      map[string]any{"Result": "Success"},
				"StatusInfoResult":      map[string]any{"Result": "Success"},
				"LastModified":          updated,
				"PriorityTimestamp":     updated,
			},
		},
	})
}

// ---- Download ----

// handleDownload handles GET /kobo/{token}/download/{bookID}/{format}.
func (h *KoboHandler) handleDownload(w http.ResponseWriter, r *http.Request, subPath string) {
	trimmed := strings.TrimPrefix(subPath, "/download/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		writeKoboJSON(w, http.StatusBadRequest, map[string]any{})
		return
	}
	bookID := parts[0]
	format := strings.ToLower(parts[1])

	files, err := h.DB.ListBookFiles(r.Context(), bookID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeKoboJSON(w, http.StatusNotFound, map[string]any{})
			return
		}
		slog.ErrorContext(r.Context(), "failed to list book files for kobo download", slog.Any(otelkeys.Error, err))
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}

	var target *db.BookFile
	for i := range files {
		if strings.ToLower(files[i].FileType) == format {
			target = &files[i]
			break
		}
	}
	if target == nil {
		writeKoboJSON(w, http.StatusNotFound, map[string]any{})
		return
	}

	f, err := os.Open(target.FilePath)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to open book file for kobo download",
			slog.String(otelkeys.Path, target.FilePath),
			slog.Any(otelkeys.Error, err),
		)
		writeKoboJSON(w, http.StatusNotFound, map[string]any{})
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		writeKoboJSON(w, http.StatusInternalServerError, map[string]any{})
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", target.FileName))
	http.ServeContent(w, r, target.FileName, stat.ModTime(), f)
}

// ---- Cover image ----

// handleCoverImage handles requests for book cover images.
// Path: /covers/{bookID}/{width}/{height}/{quality}/{isGreyscale}/image.jpg
// If the book has a cover_image_url, it redirects there; otherwise returns 404.
func (h *KoboHandler) handleCoverImage(w http.ResponseWriter, r *http.Request, subPath string) {
	trimmed := strings.TrimPrefix(subPath, "/covers/")
	bookID := strings.SplitN(trimmed, "/", 2)[0]
	if bookID == "" {
		http.NotFound(w, r)
		return
	}

	book, err := h.DB.GetBook(r.Context(), bookID)
	if err != nil || book.CoverImageURL == nil || *book.CoverImageURL == "" {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, *book.CoverImageURL, http.StatusTemporaryRedirect)
}

// ---- Metadata helpers ----

func koboDownloadURLs(base, tokenValue, bookID string, files []db.BookFile) []map[string]any {
	var urls []map[string]any
	for _, f := range files {
		koboFmt, ok := koboFormatForFileType(f.FileType)
		if !ok {
			continue
		}
		urls = append(urls, map[string]any{
			"Format":   koboFmt,
			"Size":     f.FileSize,
			"Url":      fmt.Sprintf("%s/kobo/%s/download/%s/%s", base, tokenValue, bookID, strings.ToLower(f.FileType)),
			"Platform": "Generic",
		})
	}
	return urls
}

// koboFormatForFileType maps our file_type values to Kobo format identifiers.
func koboFormatForFileType(fileType string) (string, bool) {
	switch strings.ToLower(fileType) {
	case "epub":
		return "EPUB3", true
	case "kepub":
		return "KEPUB", true
	case "mobi":
		return "MOBI", true
	case "pdf":
		return "PDF", true
	case "azw3":
		return "AZW3", true
	default:
		return "", false
	}
}

func koboBookEntitlement(book *db.Book) map[string]any {
	now := time.Now().UTC().Format(time.RFC3339)
	return map[string]any{
		"Accessibility":       "Full",
		"ActivePeriod":        map[string]any{"From": now},
		"Created":             book.CreatedAt.UTC().Format(time.RFC3339),
		"CrossRevisionId":     book.ID,
		"Id":                  book.ID,
		"IsRemoved":           false,
		"IsHiddenFromArchive": false,
		"IsLocked":            false,
		"LastModified":        book.UpdatedAt.UTC().Format(time.RFC3339),
		"OriginCategory":      "Imported",
		"RevisionId":          book.ID,
		"Status":              "Active",
	}
}

func koboBookMetadata(book *db.Book, authors []db.Author, series []db.BookSeriesEntry, downloadURLs []map[string]any) map[string]any {
	var contributorRoles []map[string]any
	var contributors []string
	for _, a := range authors {
		contributorRoles = append(contributorRoles, map[string]any{"Name": a.Name})
		contributors = append(contributors, a.Name)
	}

	metadata := map[string]any{
		"Categories":              []string{"00000000-0000-0000-0000-000000000001"},
		"CoverImageId":            book.ID,
		"CrossRevisionId":         book.ID,
		"CurrentDisplayPrice":     map[string]any{"CurrencyCode": "USD", "TotalAmount": 0},
		"CurrentLoveDisplayPrice": map[string]any{"TotalAmount": 0},
		"Description":             koboPtrStr(book.Description),
		"DownloadUrls":            downloadURLs,
		"EntitlementId":           book.ID,
		"ExternalIds":             []any{},
		"Genre":                   "00000000-0000-0000-0000-000000000001",
		"IsEligibleForKoboLove":   false,
		"IsInternetArchive":       false,
		"IsPreOrder":              false,
		"IsSocialEnabled":         true,
		"Language":                koboLanguage(book),
		"PhoneticPronunciations":  map[string]any{},
		"PublicationDate":         koboPubDate(book),
		"Publisher":               map[string]any{"Imprint": "", "Name": koboPtrStr(book.Publisher)},
		"RevisionId":              book.ID,
		"Title":                   book.Title,
		"WorkId":                  book.ID,
		"ContributorRoles":        contributorRoles,
		"Contributors":            contributors,
	}

	if len(series) > 0 {
		s := series[0]
		metadata["Series"] = map[string]any{
			"Name":        s.Series.Name,
			"Number":      koboSeriesNumber(s.Position),
			"NumberFloat": s.Position,
			"Id":          s.Series.ID,
		}
	}

	return metadata
}

func koboPtrStr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func koboLanguage(book *db.Book) string {
	if book.Language != nil && *book.Language != "" {
		return *book.Language
	}
	return "en"
}

func koboPubDate(book *db.Book) string {
	if book.PublicationDate != nil && *book.PublicationDate != "" {
		for _, layout := range []string{time.RFC3339, "2006-01-02", "2006"} {
			if t, err := time.Parse(layout, *book.PublicationDate); err == nil {
				return t.UTC().Format(time.RFC3339)
			}
		}
		return *book.PublicationDate
	}
	return time.Time{}.UTC().Format(time.RFC3339)
}

func koboSeriesNumber(pos *float64) any {
	if pos == nil {
		return 1
	}
	if *pos == float64(int(*pos)) {
		return int(*pos)
	}
	return *pos
}

func koboReadingStateResponse(state *db.KoboReadingState) map[string]any {
	now := time.Now().UTC().Format(time.RFC3339)

	updatedAt := now
	if !state.UpdatedAt.IsZero() {
		updatedAt = state.UpdatedAt.UTC().Format(time.RFC3339)
	}
	createdAt := updatedAt
	if !state.CreatedAt.IsZero() {
		createdAt = state.CreatedAt.UTC().Format(time.RFC3339)
	}

	statusInfo := map[string]any{
		"LastModified":        updatedAt,
		"Status":              state.Status,
		"TimesStartedReading": 0,
	}

	currentBookmark := map[string]any{
		"LastModified": updatedAt,
	}
	if state.PercentRead != nil {
		currentBookmark["ProgressPercent"] = *state.PercentRead
		currentBookmark["ContentSourceProgressPercent"] = *state.PercentRead
	}
	if state.LocationValue != nil && state.LocationType != nil && state.LocationSource != nil {
		currentBookmark["Location"] = map[string]any{
			"Value":  *state.LocationValue,
			"Type":   *state.LocationType,
			"Source": *state.LocationSource,
		}
	}

	return map[string]any{
		"EntitlementId":     state.BookID,
		"Created":           createdAt,
		"LastModified":      updatedAt,
		"PriorityTimestamp": updatedAt,
		"StatusInfo":        statusInfo,
		"Statistics":        map[string]any{"LastModified": updatedAt},
		"CurrentBookmark":   currentBookmark,
	}
}
