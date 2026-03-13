package db

import "database/sql"

// MovieWatchProvider represents a row in the movie_watch_providers table.
type MovieWatchProvider struct {
	ID              string    `json:"id"`
	MovieID         string    `json:"movie_id"`
	ProviderName    string    `json:"provider_name"`
	ProviderID      int       `json:"provider_id"`
	LogoPath        string    `json:"logo_path"`
	ProviderType    string    `json:"provider_type"` // "stream", "rent", or "buy"
	DisplayPriority int       `json:"display_priority"`
	CreatedAt       Timestamp `json:"created_at"`
}

// SaveMovieWatchProviders replaces all watch providers for a movie.
func (d *DB) SaveMovieWatchProviders(movieID string, providers []MovieWatchProvider) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(`DELETE FROM movie_watch_providers WHERE movie_id = $1`, movieID); err != nil {
		return err
	}

	stmt, err := tx.Prepare(
		`INSERT INTO movie_watch_providers (movie_id, provider_name, provider_id, logo_path, provider_type, display_priority)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
	)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for _, p := range providers {
		if _, err := stmt.Exec(p.MovieID, p.ProviderName, p.ProviderID, p.LogoPath, p.ProviderType, p.DisplayPriority); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`UPDATE movies SET providers_fetched_at = `+d.now()+` WHERE id = $1`, movieID); err != nil {
		return err
	}

	return tx.Commit()
}

// GetMovieWatchProviders returns all watch providers for a movie by movie ID.
func (d *DB) GetMovieWatchProviders(movieID string) ([]MovieWatchProvider, error) {
	rows, err := d.Query(
		`SELECT id, movie_id, provider_name, provider_id, logo_path, provider_type, display_priority, created_at
		 FROM movie_watch_providers
		 WHERE movie_id = $1
		 ORDER BY provider_type, display_priority`,
		movieID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var providers []MovieWatchProvider
	for rows.Next() {
		var p MovieWatchProvider
		if err := rows.Scan(&p.ID, &p.MovieID, &p.ProviderName, &p.ProviderID, &p.LogoPath, &p.ProviderType, &p.DisplayPriority, &p.CreatedAt); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

// GetMovieWatchProvidersByTmdbID returns all watch providers for a movie by TMDB ID.
func (d *DB) GetMovieWatchProvidersByTmdbID(tmdbID int) ([]MovieWatchProvider, error) {
	rows, err := d.Query(
		`SELECT p.id, p.movie_id, p.provider_name, p.provider_id, p.logo_path, p.provider_type, p.display_priority, p.created_at
		 FROM movie_watch_providers p
		 JOIN movies m ON m.id = p.movie_id
		 WHERE m.tmdb_id = $1
		 ORDER BY p.provider_type, p.display_priority`,
		tmdbID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var providers []MovieWatchProvider
	for rows.Next() {
		var p MovieWatchProvider
		if err := rows.Scan(&p.ID, &p.MovieID, &p.ProviderName, &p.ProviderID, &p.LogoPath, &p.ProviderType, &p.DisplayPriority, &p.CreatedAt); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

// IsProvidersFetched reports whether watch providers have been fetched for a movie.
// Returns (false, nil) when the movie is not in the database.
func (d *DB) IsProvidersFetched(tmdbID int) (bool, error) {
	var fetched bool
	err := d.QueryRow(
		`SELECT providers_fetched_at IS NOT NULL FROM movies WHERE tmdb_id = $1`, tmdbID,
	).Scan(&fetched)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return fetched, nil
}
