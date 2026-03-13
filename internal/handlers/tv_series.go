package handlers

// import (
// 	"database/sql"
// 	"encoding/json"
// 	"log/slog"
// 	"math"
// 	"net/http"
// 	"strconv"
// 	"strings"

// 	"github.com/amalgamated-tools/biblioteka/internal/auth"
// 	"github.com/amalgamated-tools/biblioteka/internal/db"
// 	"github.com/amalgamated-tools/biblioteka/internal/jobs"
// 	"github.com/amalgamated-tools/biblioteka/internal/worker"
// 	"github.com/amalgamated-tools/biblioteka/pkg/tmdb"
// )

// // TvSeriesHandler holds dependencies for TV series endpoints.
// type TvSeriesHandler struct {
// 	DB         *db.DB
// 	TmdbClient *tmdb.Client
// 	Worker     *worker.Worker
// }

// type tvSearchResult struct {
// 	TmdbID   int    `json:"tmdb_id"`
// 	Title    string `json:"title"`
// 	Year     int    `json:"year"`
// 	Overview string `json:"overview"`
// 	Poster   string `json:"poster_url"`
// 	Liked    bool   `json:"liked"`
// 	SeriesID string `json:"id,omitempty"`
// }

// const tmdbImageBase = "https://image.tmdb.org/t/p/w300"

// type likeTvSeriesRequest struct {
// 	Title     string  `json:"title"`
// 	Overview  *string `json:"overview"`
// 	Year      *int    `json:"year"`
// 	PosterURL *string `json:"poster_url"`
// }

// type tvSeriesDTO struct {
// 	ID        string       `json:"id"`
// 	Title     string       `json:"title"`
// 	Overview  *string      `json:"overview"`
// 	Year      *int         `json:"year"`
// 	TmdbID    *int         `json:"tmdb_id"`
// 	PosterURL *string      `json:"poster_url"`
// 	Status    string       `json:"status"`
// 	CreatedAt db.Timestamp `json:"created_at"`
// }

// type tvProvidersResponse struct {
// 	TmdbID int             `json:"tmdb_id"`
// 	Link   string          `json:"link,omitempty"`
// 	Stream []watchProvider `json:"stream"`
// 	Buy    []watchProvider `json:"buy"`
// }

// func toTvSeriesDTO(s *db.TvSeries) tvSeriesDTO {
// 	return tvSeriesDTO{
// 		ID:        s.ID,
// 		Title:     s.Title,
// 		Overview:  s.Overview,
// 		Year:      s.Year,
// 		TmdbID:    s.TmdbID,
// 		PosterURL: s.PosterURL,
// 		Status:    s.Status,
// 		CreatedAt: s.CreatedAt,
// 	}
// }

// // HandleTvSeriesSearch handles GET /api/tv-series/search?q=...
// func (h *TvSeriesHandler) HandleTvSeriesSearch(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodGet {
// 		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
// 		return
// 	}

// 	if h.TmdbClient == nil {
// 		writeError(w, http.StatusServiceUnavailable, "TMDB is not configured")
// 		return
// 	}

// 	userID := auth.UserIDFromContext(r.Context())
// 	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))

// 	likedSeries, err := h.DB.ListTvSeriesByUser(userID)
// 	if err != nil {
// 		slog.ErrorContext(r.Context(), "failed to list liked TV series for user", "user_id", userID, "error", err)
// 		writeError(w, http.StatusInternalServerError, "failed to load liked TV series")
// 		return
// 	}
// 	likedTmdbIDs := make(map[int]string)
// 	for _, s := range likedSeries {
// 		if s.TmdbID != nil {
// 			likedTmdbIDs[*s.TmdbID] = s.ID
// 		}
// 	}

// 	tmdbClient := h.TmdbClient.ClientWithResponses()

// 	type tmdbTvShow struct {
// 		Id           int
// 		Name         string
// 		FirstAirDate string
// 		Overview     string
// 		PosterPath   string
// 	}

// 	var shows []tmdbTvShow
// 	if query == "" {
// 		resp, err := tmdbClient.TrendingTvWithResponse(r.Context(), tmdb.TrendingTvParamsTimeWindowWeek, &tmdb.TrendingTvParams{})
// 		if err != nil {
// 			slog.ErrorContext(r.Context(), "failed to fetch trending TV series from TMDB", "error", err)
// 			writeError(w, http.StatusBadGateway, "failed to fetch trending TV series")
// 			return
// 		}
// 		if resp.JSON200 == nil {
// 			slog.ErrorContext(r.Context(), "unexpected TMDB response status", "status", resp.Status())
// 			writeError(w, http.StatusBadGateway, "unexpected response from trending TV")
// 			return
// 		}
// 		for _, s := range resp.JSON200.Results {
// 			shows = append(shows, tmdbTvShow{Id: s.Id, Name: s.Name, FirstAirDate: s.FirstAirDate, Overview: s.Overview, PosterPath: s.PosterPath})
// 		}
// 	} else {
// 		resp, err := tmdbClient.SearchTvWithResponse(r.Context(), &tmdb.SearchTvParams{Query: query})
// 		if err != nil {
// 			slog.ErrorContext(r.Context(), "failed to search TMDB for TV series", "error", err)
// 			writeError(w, http.StatusBadGateway, "failed to search TV series")
// 			return
// 		}
// 		if resp.JSON200 == nil {
// 			slog.ErrorContext(r.Context(), "unexpected TMDB response status", "status", resp.Status())
// 			writeError(w, http.StatusBadGateway, "unexpected response from TV search")
// 			return
// 		}
// 		for _, s := range resp.JSON200.Results {
// 			shows = append(shows, tmdbTvShow{Id: s.Id, Name: s.Name, FirstAirDate: s.FirstAirDate, Overview: s.Overview, PosterPath: s.PosterPath})
// 		}
// 	}

// 	results := make([]tvSearchResult, 0, len(shows))
// 	for _, s := range shows {
// 		year := 0
// 		if len(s.FirstAirDate) >= 4 {
// 			year, _ = strconv.Atoi(s.FirstAirDate[:4])
// 		}
// 		poster := tmdbLogoURL(tmdbImageBase, s.PosterPath)
// 		result := tvSearchResult{
// 			TmdbID:   s.Id,
// 			Title:    s.Name,
// 			Year:     year,
// 			Overview: s.Overview,
// 			Poster:   poster,
// 		}
// 		if seriesID, ok := likedTmdbIDs[s.Id]; ok {
// 			result.Liked = true
// 			result.SeriesID = seriesID
// 		}
// 		results = append(results, result)
// 	}
// 	writeJSON(w, http.StatusOK, results)
// }

// // HandleTvSeriesList handles GET (list liked) on /api/tv-series
// func (h *TvSeriesHandler) HandleTvSeriesList(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodGet {
// 		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
// 		return
// 	}

// 	userID := auth.UserIDFromContext(r.Context())
// 	series, err := h.DB.ListTvSeriesByUser(userID)
// 	if err != nil {
// 		slog.ErrorContext(r.Context(), "failed to list liked TV series for user", "user_id", userID, "error", err)
// 		writeError(w, http.StatusInternalServerError, "failed to list tv series")
// 		return
// 	}

// 	dtos := make([]tvSeriesDTO, 0, len(series))
// 	for i := range series {
// 		dtos = append(dtos, toTvSeriesDTO(&series[i]))
// 	}
// 	writeJSON(w, http.StatusOK, dtos)
// }

// // HandleTvSeriesResource handles routes under /api/tv-series/{id}/ including like sub-resource
// func (h *TvSeriesHandler) HandleTvSeriesResource(w http.ResponseWriter, r *http.Request) {
// 	rest := strings.TrimPrefix(r.URL.Path, "/api/tv-series/")
// 	rest = strings.TrimSuffix(rest, "/")

// 	if parts := strings.SplitN(rest, "/", 2); len(parts) == 2 {
// 		switch parts[1] {
// 		case "like":
// 			h.handleLikeToggle(w, r, parts[0])
// 			return
// 		case "providers":
// 			h.handleTvSeriesProviders(w, r, parts[0])
// 			return
// 		}
// 	}

// 	writeError(w, http.StatusNotFound, "not found")
// }

// func (h *TvSeriesHandler) handleLikeToggle(w http.ResponseWriter, r *http.Request, tmdbIDStr string) {
// 	tmdbID, err := strconv.Atoi(tmdbIDStr)
// 	if err != nil || tmdbID <= 0 {
// 		writeError(w, http.StatusBadRequest, "invalid tmdb ID")
// 		return
// 	}

// 	userID := auth.UserIDFromContext(r.Context())

// 	switch r.Method {
// 	case http.MethodPost:
// 		var req likeTvSeriesRequest
// 		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 			writeError(w, http.StatusBadRequest, "invalid request body")
// 			return
// 		}

// 		if strings.TrimSpace(req.Title) == "" {
// 			writeError(w, http.StatusBadRequest, "title is required")
// 			return
// 		}

// 		existing, _ := h.DB.GetTvSeriesByTmdbID(tmdbID)
// 		if existing != nil {
// 			if err := h.DB.LikeTvSeries(userID, existing.ID); err != nil {
// 				slog.ErrorContext(r.Context(), "failed to like tv series", "tv_series_id", existing.ID, "user_id", userID, "error", err)
// 				writeError(w, http.StatusInternalServerError, "failed to like tv series")
// 				return
// 			}
// 			writeJSON(w, http.StatusOK, toTvSeriesDTO(existing))
// 			return
// 		}

// 		series, err := h.DB.CreateTvSeries(
// 			strings.TrimSpace(req.Title),
// 			nil, req.Overview,
// 			req.Year, nil, nil, nil, nil,
// 			"added", "standard",
// 			nil, &tmdbID, nil, req.PosterURL, nil,
// 		)
// 		if err != nil {
// 			// Handle race condition: another request may have created this series concurrently
// 			existing, fetchErr := h.DB.GetTvSeriesByTmdbID(tmdbID)
// 			if fetchErr != nil || existing == nil {
// 				slog.ErrorContext(r.Context(), "failed to create tv series", "user_id", userID, "error", err)
// 				writeError(w, http.StatusInternalServerError, "failed to create tv series")
// 				return
// 			}
// 			if err := h.DB.LikeTvSeries(userID, existing.ID); err != nil {
// 				slog.ErrorContext(r.Context(), "failed to like tv series", "tv_series_id", existing.ID, "user_id", userID, "error", err)
// 				writeError(w, http.StatusInternalServerError, "failed to like tv series")
// 				return
// 			}
// 			writeJSON(w, http.StatusOK, toTvSeriesDTO(existing))
// 			return
// 		}

// 		if err := h.DB.LikeTvSeries(userID, series.ID); err != nil {
// 			slog.ErrorContext(r.Context(), "failed to like tv series", "tv_series_id", series.ID, "user_id", userID, "error", err)
// 			writeError(w, http.StatusInternalServerError, "failed to like tv series")
// 			return
// 		}
// 		writeJSON(w, http.StatusCreated, toTvSeriesDTO(series))

// 	case http.MethodDelete:
// 		existing, err := h.DB.GetTvSeriesByTmdbID(tmdbID)
// 		if err != nil {
// 			if err == sql.ErrNoRows {
// 				writeError(w, http.StatusNotFound, "tv series not found")
// 				return
// 			}
// 			slog.ErrorContext(r.Context(), "failed to get tv series by tmdb_id", "tmdb_id", tmdbID, "error", err)
// 			writeError(w, http.StatusInternalServerError, "failed to find tv series")
// 			return
// 		}

// 		if err := h.DB.UnlikeTvSeries(userID, existing.ID); err != nil {
// 			if err == sql.ErrNoRows {
// 				writeError(w, http.StatusNotFound, "tv series not in your collection")
// 				return
// 			}
// 			slog.ErrorContext(r.Context(), "failed to unlike tv series", "tv_series_id", existing.ID, "user_id", userID, "error", err)
// 			writeError(w, http.StatusInternalServerError, "failed to unlike tv series")
// 			return
// 		}
// 		w.WriteHeader(http.StatusNoContent)

// 	default:
// 		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
// 	}
// }

// func (h *TvSeriesHandler) handleTvSeriesProviders(w http.ResponseWriter, r *http.Request, tmdbIDStr string) {
// 	if r.Method != http.MethodGet {
// 		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
// 		return
// 	}

// 	tmdbID, err := strconv.Atoi(tmdbIDStr)
// 	if err != nil || tmdbID <= 0 || tmdbID > math.MaxInt32 {
// 		writeError(w, http.StatusBadRequest, "invalid tmdb ID")
// 		return
// 	}

// 	// Try serving from DB cache first
// 	cached, err := h.DB.GetTvSeriesWatchProvidersByTmdbID(tmdbID)
// 	if err == nil && len(cached) > 0 {
// 		result := tvProvidersResponse{
// 			TmdbID: tmdbID,
// 			Stream: make([]watchProvider, 0),
// 			Buy:    make([]watchProvider, 0),
// 		}
// 		for _, p := range cached {
// 			wp := watchProvider{ID: p.ProviderID, Name: p.ProviderName, LogoURL: p.LogoPath}
// 			switch p.ProviderType {
// 			case "stream":
// 				result.Stream = append(result.Stream, wp)
// 			case "buy":
// 				result.Buy = append(result.Buy, wp)
// 			}
// 		}
// 		writeJSON(w, http.StatusOK, result)
// 		return
// 	}

// 	// If providers were already fetched but the result was empty,
// 	// serve the empty response from cache instead of hitting TMDB again.
// 	if fetched, err := h.DB.IsTvSeriesProvidersFetched(tmdbID); err != nil {
// 		slog.ErrorContext(r.Context(), "failed to check if TV series providers were fetched", "tmdb_id", tmdbID, "error", err)
// 		writeError(w, http.StatusInternalServerError, "internal server error")
// 		return
// 	} else if fetched {
// 		writeJSON(w, http.StatusOK, tvProvidersResponse{
// 			TmdbID: tmdbID,
// 			Stream: make([]watchProvider, 0),
// 			Buy:    make([]watchProvider, 0),
// 		})
// 		return
// 	}

// 	// Fallback: fetch live from TMDB
// 	if h.TmdbClient == nil {
// 		writeError(w, http.StatusServiceUnavailable, "TMDB is not configured")
// 		return
// 	}
// 	tmdbClient := h.TmdbClient.ClientWithResponses()
// 	resp, err := tmdbClient.TvSeriesWatchProvidersWithResponse(r.Context(), int32(tmdbID))
// 	if err != nil {
// 		slog.ErrorContext(r.Context(), "failed to fetch watch providers for tv series", "tmdb_id", tmdbID, "error", err)
// 		writeError(w, http.StatusBadGateway, "failed to fetch watch providers")
// 		return
// 	}
// 	if resp.JSON200 == nil {
// 		slog.ErrorContext(r.Context(), "unexpected TMDB watch provider response status for tv series", "tmdb_id", tmdbID, "status", resp.Status())
// 		writeError(w, http.StatusBadGateway, "unexpected response from TMDB")
// 		return
// 	}

// 	us := resp.JSON200.Results.US
// 	result := tvProvidersResponse{
// 		TmdbID: tmdbID,
// 		Link:   us.Link,
// 		Stream: make([]watchProvider, 0, len(us.Flatrate)),
// 		Buy:    make([]watchProvider, 0, len(us.Buy)),
// 	}

// 	for _, p := range us.Flatrate {
// 		result.Stream = append(result.Stream, watchProvider{
// 			ID:      p.ProviderId,
// 			Name:    p.ProviderName,
// 			LogoURL: tmdbLogoURL(tmdbLogoBase, p.LogoPath),
// 		})
// 	}
// 	for _, p := range us.Buy {
// 		result.Buy = append(result.Buy, watchProvider{
// 			ID:      p.ProviderId,
// 			Name:    p.ProviderName,
// 			LogoURL: tmdbLogoURL(tmdbLogoBase, p.LogoPath),
// 		})
// 	}

// 	// Enqueue a background job to cache the results for next time
// 	series, _ := h.DB.GetTvSeriesByTmdbID(tmdbID)
// 	if series != nil {
// 		h.enqueueTvProviderFetch(r, series.ID, tmdbID)
// 	}

// 	writeJSON(w, http.StatusOK, result)
// }

// func (h *TvSeriesHandler) enqueueTvProviderFetch(r *http.Request, tvSeriesID string, tmdbID int) {
// 	if h.Worker == nil || h.TmdbClient == nil {
// 		return
// 	}
// 	_, err := h.Worker.Enqueue(r.Context(), jobs.FetchTvSeriesProviders, jobs.FetchTvProvidersPayload{
// 		TvSeriesID: tvSeriesID,
// 		TmdbID:     tmdbID,
// 	})
// 	if err != nil {
// 		slog.ErrorContext(r.Context(), "failed to enqueue provider fetch for tv series", "tv_series_id", tvSeriesID, "tmdb_id", tmdbID, "error", err)
// 	}
// }
