package db

import "database/sql"

// TvSeriesWatchProvider represents a row in the tv_series_watch_providers table.
type TvSeriesWatchProvider struct {
	ID              string    `json:"id"`
	TvSeriesID      string    `json:"tv_series_id"`
	ProviderName    string    `json:"provider_name"`
	ProviderID      int       `json:"provider_id"`
	LogoPath        string    `json:"logo_path"`
	ProviderType    string    `json:"provider_type"` // "stream", "rent", or "buy"
	DisplayPriority int       `json:"display_priority"`
	CreatedAt       Timestamp `json:"created_at"`
}

// SaveTvSeriesWatchProviders replaces all watch providers for a TV series.
func (d *DB) SaveTvSeriesWatchProviders(tvSeriesID string, providers []TvSeriesWatchProvider) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(`DELETE FROM tv_series_watch_providers WHERE tv_series_id = $1`, tvSeriesID); err != nil {
		return err
	}

	stmt, err := tx.Prepare(
		`INSERT INTO tv_series_watch_providers (tv_series_id, provider_name, provider_id, logo_path, provider_type, display_priority)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
	)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for _, p := range providers {
		if _, err := stmt.Exec(p.TvSeriesID, p.ProviderName, p.ProviderID, p.LogoPath, p.ProviderType, p.DisplayPriority); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`UPDATE tv_series SET providers_fetched_at = `+d.now()+` WHERE id = $1`, tvSeriesID); err != nil {
		return err
	}

	return tx.Commit()
}

// GetTvSeriesWatchProviders returns all watch providers for a TV series by ID.
func (d *DB) GetTvSeriesWatchProviders(tvSeriesID string) ([]TvSeriesWatchProvider, error) {
	rows, err := d.Query(
		`SELECT id, tv_series_id, provider_name, provider_id, logo_path, provider_type, display_priority, created_at
		 FROM tv_series_watch_providers
		 WHERE tv_series_id = $1
		 ORDER BY provider_type, display_priority`,
		tvSeriesID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var providers []TvSeriesWatchProvider
	for rows.Next() {
		var p TvSeriesWatchProvider
		if err := rows.Scan(&p.ID, &p.TvSeriesID, &p.ProviderName, &p.ProviderID, &p.LogoPath, &p.ProviderType, &p.DisplayPriority, &p.CreatedAt); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

// GetTvSeriesWatchProvidersByTmdbID returns all watch providers for a TV series by TMDB ID.
func (d *DB) GetTvSeriesWatchProvidersByTmdbID(tmdbID int) ([]TvSeriesWatchProvider, error) {
	rows, err := d.Query(
		`SELECT p.id, p.tv_series_id, p.provider_name, p.provider_id, p.logo_path, p.provider_type, p.display_priority, p.created_at
		 FROM tv_series_watch_providers p
		 JOIN tv_series s ON s.id = p.tv_series_id
		 WHERE s.tmdb_id = $1
		 ORDER BY p.provider_type, p.display_priority`,
		tmdbID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var providers []TvSeriesWatchProvider
	for rows.Next() {
		var p TvSeriesWatchProvider
		if err := rows.Scan(&p.ID, &p.TvSeriesID, &p.ProviderName, &p.ProviderID, &p.LogoPath, &p.ProviderType, &p.DisplayPriority, &p.CreatedAt); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

// IsTvSeriesProvidersFetched reports whether watch providers have been fetched for a TV series.
func (d *DB) IsTvSeriesProvidersFetched(tmdbID int) (bool, error) {
	var fetched bool
	err := d.QueryRow(
		`SELECT providers_fetched_at IS NOT NULL FROM tv_series WHERE tmdb_id = $1`, tmdbID,
	).Scan(&fetched)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return fetched, nil
}
