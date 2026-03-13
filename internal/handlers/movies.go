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

// // MovieHandler holds dependencies for movie endpoints.
// type MovieHandler struct {
// 	DB         *db.DB
// 	TmdbClient *tmdb.Client
// 	Worker     *worker.Worker
// }

// type movieSearchResult struct {
// 	TmdbID   int    `json:"tmdb_id"`
// 	Title    string `json:"title"`
// 	Year     int    `json:"year"`
// 	Overview string `json:"overview"`
// 	Poster   string `json:"poster_url"`
// 	Liked    bool   `json:"liked"`
// 	MovieID  string `json:"id,omitempty"`
// }

// type likeMovieRequest struct {
// 	Title     string  `json:"title"`
// 	Overview  *string `json:"overview"`
// 	Year      *int    `json:"year"`
// 	PosterURL *string `json:"poster_url"`
// }

// const tmdbLogoBase = "https://image.tmdb.org/t/p/original"

// type watchProvider struct {
// 	ID      int    `json:"id"`
// 	Name    string `json:"name"`
// 	LogoURL string `json:"logo_url"`
// }

// type movieProvidersResponse struct {
// 	TmdbID int             `json:"tmdb_id"`
// 	Link   string          `json:"link,omitempty"`
// 	Stream []watchProvider `json:"stream"`
// 	Rent   []watchProvider `json:"rent"`
// 	Buy    []watchProvider `json:"buy"`
// }

// type movieDTO struct {
// 	ID        string       `json:"id"`
// 	Title     string       `json:"title"`
// 	Overview  *string      `json:"overview"`
// 	Year      *int         `json:"year"`
// 	TmdbID    *int         `json:"tmdb_id"`
// 	PosterURL *string      `json:"poster_url"`
// 	Status    string       `json:"status"`
// 	CreatedAt db.Timestamp `json:"created_at"`
// }

// func toMovieDTO(m *db.Movie) movieDTO {
// 	return movieDTO{
// 		ID:        m.ID,
// 		Title:     m.Title,
// 		Overview:  m.Overview,
// 		Year:      m.Year,
// 		TmdbID:    m.TmdbID,
// 		PosterURL: m.PosterURL,
// 		Status:    m.Status,
// 		CreatedAt: m.CreatedAt,
// 	}
// }

// // HandleMovieSearch handles GET /api/movies/search?q=...
// func (h *MovieHandler) HandleMovieSearch(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodGet {
// 		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
// 		return
// 	}

// 	if h.TmdbClient == nil {
// 		writeError(w, http.StatusServiceUnavailable, "TMDB is not configured")
// 		return
// 	}

// 	userID := auth.UserIDFromContext(r.Context())
// 	query := strings.TrimSpace(r.URL.Query().Get("q"))

// 	// Build a set of tmdb_ids the user has liked
// 	likedMovies, err := h.DB.ListMoviesByUser(userID)
// 	if err != nil {
// 		slog.Error("failed to list movies for user", "user_id", userID, "error", err)
// 		writeError(w, http.StatusInternalServerError, "failed to load liked movies")
// 		return
// 	}
// 	likedTmdbIDs := make(map[int]string) // tmdb_id -> movie.id
// 	for _, m := range likedMovies {
// 		if m.TmdbID != nil {
// 			likedTmdbIDs[*m.TmdbID] = m.ID
// 		}
// 	}

// 	tmdbClient := h.TmdbClient.ClientWithResponses()

// 	type tmdbMovie struct {
// 		Id          int
// 		Title       string
// 		ReleaseDate string
// 		Overview    string
// 		PosterPath  string
// 	}

// 	var movies []tmdbMovie
// 	if query == "" {
// 		resp, err := tmdbClient.TrendingMoviesWithResponse(r.Context(), tmdb.TrendingMoviesParamsTimeWindowWeek, &tmdb.TrendingMoviesParams{})
// 		if err != nil {
// 			slog.Error("failed to fetch trending movies from TMDB", "error", err)
// 			writeError(w, http.StatusBadGateway, "failed to fetch trending movies")
// 			return
// 		}
// 		if resp.JSON200 == nil {
// 			slog.Error("unexpected TMDB response status", "status", resp.Status())
// 			writeError(w, http.StatusBadGateway, "unexpected response from trending movies")
// 			return
// 		}
// 		for _, m := range resp.JSON200.Results {
// 			movies = append(movies, tmdbMovie{Id: m.Id, Title: m.Title, ReleaseDate: m.ReleaseDate, Overview: m.Overview, PosterPath: m.PosterPath})
// 		}
// 	} else {
// 		resp, err := tmdbClient.SearchMovieWithResponse(r.Context(), &tmdb.SearchMovieParams{Query: query})
// 		if err != nil {
// 			slog.Error("failed to search TMDB for movies", "error", err)
// 			writeError(w, http.StatusBadGateway, "failed to search movies")
// 			return
// 		}
// 		if resp.JSON200 == nil {
// 			slog.Error("unexpected TMDB response status", "status", resp.Status())
// 			writeError(w, http.StatusBadGateway, "unexpected response from movie search")
// 			return
// 		}
// 		for _, m := range resp.JSON200.Results {
// 			movies = append(movies, tmdbMovie{Id: m.Id, Title: m.Title, ReleaseDate: m.ReleaseDate, Overview: m.Overview, PosterPath: m.PosterPath})
// 		}
// 	}

// 	results := make([]movieSearchResult, 0, len(movies))
// 	for _, m := range movies {
// 		year := 0
// 		if len(m.ReleaseDate) >= 4 {
// 			year, _ = strconv.Atoi(m.ReleaseDate[:4])
// 		}
// 		poster := tmdbLogoURL(tmdbImageBase, m.PosterPath)
// 		result := movieSearchResult{
// 			TmdbID:   m.Id,
// 			Title:    m.Title,
// 			Year:     year,
// 			Overview: m.Overview,
// 			Poster:   poster,
// 		}
// 		if movieID, ok := likedTmdbIDs[m.Id]; ok {
// 			result.Liked = true
// 			result.MovieID = movieID
// 		}
// 		results = append(results, result)
// 	}
// 	writeJSON(w, http.StatusOK, results)
// }

// // HandleMovies handles GET (list liked) on /api/movies
// func (h *MovieHandler) HandleMovies(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodGet {
// 		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
// 		return
// 	}
// 	h.listMovies(w, r)
// }

// // HandleMovie handles routes under /api/movies/{id}/ including like and providers sub-resources
// func (h *MovieHandler) HandleMovie(w http.ResponseWriter, r *http.Request) {
// 	rest := strings.TrimPrefix(r.URL.Path, "/api/movies/")
// 	rest = strings.TrimSuffix(rest, "/")

// 	if parts := strings.SplitN(rest, "/", 2); len(parts) == 2 {
// 		switch parts[1] {
// 		case "like":
// 			h.handleLikeToggle(w, r, parts[0])
// 			return
// 		case "providers":
// 			h.handleMovieProviders(w, r, parts[0])
// 			return
// 		}
// 	}

// 	writeError(w, http.StatusNotFound, "not found")
// }

// func (h *MovieHandler) handleMovieProviders(w http.ResponseWriter, r *http.Request, tmdbIDStr string) {
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
// 	cached, err := h.DB.GetMovieWatchProvidersByTmdbID(tmdbID)
// 	if err == nil && len(cached) > 0 {
// 		result := movieProvidersResponse{
// 			TmdbID: tmdbID,
// 			Stream: make([]watchProvider, 0),
// 			Rent:   make([]watchProvider, 0),
// 			Buy:    make([]watchProvider, 0),
// 		}
// 		for _, p := range cached {
// 			wp := watchProvider{ID: p.ProviderID, Name: p.ProviderName, LogoURL: p.LogoPath}
// 			switch p.ProviderType {
// 			case "stream":
// 				result.Stream = append(result.Stream, wp)
// 			case "rent":
// 				result.Rent = append(result.Rent, wp)
// 			case "buy":
// 				result.Buy = append(result.Buy, wp)
// 			}
// 		}
// 		writeJSON(w, http.StatusOK, result)
// 		return
// 	}

// 	// If providers were already fetched but the result was empty (no US providers),
// 	// serve the empty response from cache instead of hitting TMDB again.
// 	if fetched, err := h.DB.IsProvidersFetched(tmdbID); err != nil {
// 		slog.Error("failed to check providers_fetched_at", "tmdb_id", tmdbID, "error", err)
// 		writeError(w, http.StatusInternalServerError, "internal server error")
// 		return
// 	} else if fetched {
// 		writeJSON(w, http.StatusOK, movieProvidersResponse{
// 			TmdbID: tmdbID,
// 			Stream: make([]watchProvider, 0),
// 			Rent:   make([]watchProvider, 0),
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
// 	resp, err := tmdbClient.MovieWatchProvidersWithResponse(r.Context(), int32(tmdbID))
// 	if err != nil {
// 		slog.Error("failed to fetch watch providers", "tmdb_id", tmdbID, "error", err)
// 		writeError(w, http.StatusBadGateway, "failed to fetch watch providers")
// 		return
// 	}
// 	if resp.JSON200 == nil {
// 		slog.Error("unexpected TMDB watch provider response status", "tmdb_id", tmdbID, "status", resp.Status())
// 		writeError(w, http.StatusBadGateway, "unexpected response from TMDB")
// 		return
// 	}

// 	us := resp.JSON200.Results.US
// 	result := movieProvidersResponse{
// 		TmdbID: tmdbID,
// 		Link:   us.Link,
// 		Stream: make([]watchProvider, 0, len(us.Flatrate)),
// 		Rent:   make([]watchProvider, 0, len(us.Rent)),
// 		Buy:    make([]watchProvider, 0, len(us.Buy)),
// 	}

// 	for _, p := range us.Flatrate {
// 		result.Stream = append(result.Stream, watchProvider{
// 			ID:      p.ProviderId,
// 			Name:    p.ProviderName,
// 			LogoURL: tmdbLogoURL(tmdbLogoBase, p.LogoPath),
// 		})
// 	}
// 	for _, p := range us.Rent {
// 		result.Rent = append(result.Rent, watchProvider{
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
// 	movie, _ := h.DB.GetMovieByTmdbID(tmdbID)
// 	if movie != nil {
// 		h.enqueueProviderFetch(r, movie.ID, tmdbID)
// 	}

// 	writeJSON(w, http.StatusOK, result)
// }

// func (h *MovieHandler) listMovies(w http.ResponseWriter, r *http.Request) {
// 	userID := auth.UserIDFromContext(r.Context())

// 	movies, err := h.DB.ListMoviesByUser(userID)
// 	if err != nil {
// 		slog.Error("failed to list movies for user", "user_id", userID, "error", err)
// 		writeError(w, http.StatusInternalServerError, "failed to list movies")
// 		return
// 	}

// 	dtos := make([]movieDTO, 0, len(movies))
// 	for i := range movies {
// 		dtos = append(dtos, toMovieDTO(&movies[i]))
// 	}
// 	writeJSON(w, http.StatusOK, dtos)
// }

// func (h *MovieHandler) handleLikeToggle(w http.ResponseWriter, r *http.Request, tmdbIDStr string) {
// 	tmdbID, err := strconv.Atoi(tmdbIDStr)
// 	if err != nil || tmdbID <= 0 {
// 		writeError(w, http.StatusBadRequest, "invalid tmdb ID")
// 		return
// 	}

// 	userID := auth.UserIDFromContext(r.Context())

// 	switch r.Method {
// 	case http.MethodPost:
// 		// Like: create movie if needed, then like
// 		var req likeMovieRequest
// 		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 			writeError(w, http.StatusBadRequest, "invalid request body")
// 			return
// 		}

// 		if strings.TrimSpace(req.Title) == "" {
// 			writeError(w, http.StatusBadRequest, "title is required")
// 			return
// 		}

// 		existing, _ := h.DB.GetMovieByTmdbID(tmdbID)
// 		if existing != nil {
// 			if err := h.DB.LikeMovie(userID, existing.ID); err != nil {
// 				slog.Error("failed to like movie", "movie_id", existing.ID, "user_id", userID, "error", err)
// 				writeError(w, http.StatusInternalServerError, "failed to like movie")
// 				return
// 			}
// 			h.enqueueProviderFetch(r, existing.ID, tmdbID)
// 			writeJSON(w, http.StatusOK, toMovieDTO(existing))
// 			return
// 		}

// 		movie, err := h.DB.CreateMovie(
// 			strings.TrimSpace(req.Title),
// 			nil, nil, req.Overview,
// 			req.Year, nil, nil, nil, nil,
// 			"added",
// 			nil, &tmdbID, nil, req.PosterURL,
// 		)
// 		if err != nil {
// 			// Handle race condition: another request may have created this movie concurrently
// 			existing, fetchErr := h.DB.GetMovieByTmdbID(tmdbID)
// 			if fetchErr != nil || existing == nil {
// 				slog.Error("failed to create movie", "user_id", userID, "error", fetchErr, "create_error", err)
// 				writeError(w, http.StatusInternalServerError, "failed to create movie")
// 				return
// 			}
// 			if err := h.DB.LikeMovie(userID, existing.ID); err != nil {
// 				slog.Error("failed to like movie", "movie_id", existing.ID, "user_id", userID, "error", err)
// 				writeError(w, http.StatusInternalServerError, "failed to like movie")
// 				return
// 			}
// 			h.enqueueProviderFetch(r, existing.ID, tmdbID)
// 			writeJSON(w, http.StatusOK, toMovieDTO(existing))
// 			return
// 		}

// 		if err := h.DB.LikeMovie(userID, movie.ID); err != nil {
// 			slog.Error("failed to like movie", "movie_id", movie.ID, "user_id", userID, "error", err)
// 			writeError(w, http.StatusInternalServerError, "failed to like movie")
// 			return
// 		}
// 		h.enqueueProviderFetch(r, movie.ID, tmdbID)
// 		writeJSON(w, http.StatusCreated, toMovieDTO(movie))

// 	case http.MethodDelete:
// 		// Unlike: find movie by tmdb_id, then unlike
// 		existing, err := h.DB.GetMovieByTmdbID(tmdbID)
// 		if err != nil {
// 			if err == sql.ErrNoRows {
// 				writeError(w, http.StatusNotFound, "movie not found")
// 				return
// 			}
// 			slog.Error("failed to get movie by tmdb_id", "tmdb_id", tmdbID, "error", err)
// 			writeError(w, http.StatusInternalServerError, "failed to find movie")
// 			return
// 		}

// 		if err := h.DB.UnlikeMovie(userID, existing.ID); err != nil {
// 			if err == sql.ErrNoRows {
// 				writeError(w, http.StatusNotFound, "movie not in your collection")
// 				return
// 			}
// 			slog.Error("failed to unlike movie", "movie_id", existing.ID, "user_id", userID, "error", err)
// 			writeError(w, http.StatusInternalServerError, "failed to unlike movie")
// 			return
// 		}
// 		w.WriteHeader(http.StatusNoContent)

// 	default:
// 		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
// 	}
// }

// func (h *MovieHandler) enqueueProviderFetch(r *http.Request, movieID string, tmdbID int) {
// 	if h.Worker == nil || h.TmdbClient == nil {
// 		return
// 	}
// 	_, err := h.Worker.Enqueue(r.Context(), jobs.FetchMovieProviders, jobs.FetchProvidersPayload{
// 		MovieID: movieID,
// 		TmdbID:  tmdbID,
// 	})
// 	if err != nil {
// 		slog.Warn("failed to enqueue provider fetch", "movie_id", movieID, "tmdb_id", tmdbID, "error", err)
// 	}
// }
