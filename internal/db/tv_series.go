package db

import "database/sql"

// TvSeries represents a row in the tv_series table.
type TvSeries struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	SortTitle     *string    `json:"sort_title"`
	Overview      *string    `json:"overview"`
	Year          *int       `json:"year"`
	Runtime       *int       `json:"runtime"`
	Certification *string    `json:"certification"`
	Network       *string    `json:"network"`
	Genres        *string    `json:"genres"`
	Status        string     `json:"status"`
	SeriesType    string     `json:"series_type"`
	ImdbID        *string    `json:"imdb_id"`
	TmdbID        *int       `json:"tmdb_id"`
	TvdbID        *int       `json:"tvdb_id"`
	PosterURL     *string    `json:"poster_url"`
	FirstAired    *Timestamp `json:"first_aired"`
	CreatedAt     Timestamp  `json:"created_at"`
	UpdatedAt     Timestamp  `json:"updated_at"`
}

const tvSeriesColumns = `id, title, sort_title, overview, year, runtime,
	certification, network, genres, status, series_type, imdb_id, tmdb_id, tvdb_id,
	poster_url, first_aired, created_at, updated_at`

func scanTvSeries(row interface{ Scan(dest ...any) error }) (*TvSeries, error) {
	var s TvSeries
	err := row.Scan(
		&s.ID, &s.Title, &s.SortTitle, &s.Overview,
		&s.Year, &s.Runtime, &s.Certification, &s.Network, &s.Genres,
		&s.Status, &s.SeriesType, &s.ImdbID, &s.TmdbID, &s.TvdbID,
		&s.PosterURL, &s.FirstAired,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// CreateTvSeries inserts a new TV series and returns it.
func (d *DB) CreateTvSeries(title string, sortTitle, overview *string,
	year, runtime *int, certification, network, genres *string, status, seriesType string,
	imdbID *string, tmdbID, tvdbID *int, posterURL, firstAired *string) (*TvSeries, error) {

	return scanTvSeries(d.QueryRow(
		`INSERT INTO tv_series (title, sort_title, overview, year, runtime,
			certification, network, genres, status, series_type, imdb_id, tmdb_id, tvdb_id,
			poster_url, first_aired)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		 RETURNING `+tvSeriesColumns,
		title, sortTitle, overview, year, runtime,
		certification, network, genres, status, seriesType,
		imdbID, tmdbID, tvdbID, posterURL, firstAired,
	))
}

// GetTvSeries returns a single TV series by ID.
func (d *DB) GetTvSeries(id string) (*TvSeries, error) {
	return scanTvSeries(d.QueryRow(
		`SELECT `+tvSeriesColumns+` FROM tv_series WHERE id = $1`, id,
	))
}

// GetTvSeriesByTmdbID returns a TV series by TMDB ID.
func (d *DB) GetTvSeriesByTmdbID(tmdbID int) (*TvSeries, error) {
	return scanTvSeries(d.QueryRow(
		`SELECT `+tvSeriesColumns+` FROM tv_series WHERE tmdb_id = $1`, tmdbID,
	))
}

// GetTvSeriesByImdbID returns a TV series by IMDB ID.
func (d *DB) GetTvSeriesByImdbID(imdbID string) (*TvSeries, error) {
	return scanTvSeries(d.QueryRow(
		`SELECT `+tvSeriesColumns+` FROM tv_series WHERE imdb_id = $1`, imdbID,
	))
}

// GetTvSeriesByTvdbID returns a TV series by TVDB ID.
func (d *DB) GetTvSeriesByTvdbID(tvdbID int) (*TvSeries, error) {
	return scanTvSeries(d.QueryRow(
		`SELECT `+tvSeriesColumns+` FROM tv_series WHERE tvdb_id = $1`, tvdbID,
	))
}

// UpdateTvSeries updates an existing TV series.
func (d *DB) UpdateTvSeries(id, title string, sortTitle, overview *string,
	year, runtime *int, certification, network, genres *string, status, seriesType string,
	imdbID *string, tmdbID, tvdbID *int, posterURL, firstAired *string) (*TvSeries, error) {

	return scanTvSeries(d.QueryRow(
		`UPDATE tv_series
		 SET title = $1, sort_title = $2, overview = $3, year = $4, runtime = $5,
			 certification = $6, network = $7, genres = $8, status = $9, series_type = $10,
			 imdb_id = $11, tmdb_id = $12, tvdb_id = $13,
			 poster_url = $14, first_aired = $15,
			 updated_at = `+d.now()+`
		 WHERE id = $16
		 RETURNING `+tvSeriesColumns,
		title, sortTitle, overview, year, runtime,
		certification, network, genres, status, seriesType,
		imdbID, tmdbID, tvdbID, posterURL, firstAired,
		id,
	))
}

// DeleteTvSeries deletes a TV series by ID.
func (d *DB) DeleteTvSeries(id string) error {
	res, err := d.Exec(`DELETE FROM tv_series WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// LikeTvSeries associates a user with a TV series.
func (d *DB) LikeTvSeries(userID, tvSeriesID string) error {
	query := `INSERT OR IGNORE INTO user_tv_series (user_id, tv_series_id) VALUES ($1, $2)`
	if d.Dialect == DialectPostgres {
		query = `INSERT INTO user_tv_series (user_id, tv_series_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	}
	_, err := d.Exec(query, userID, tvSeriesID)
	return err
}

// UnlikeTvSeries removes the association between a user and a TV series.
func (d *DB) UnlikeTvSeries(userID, tvSeriesID string) error {
	res, err := d.Exec(
		`DELETE FROM user_tv_series WHERE user_id = $1 AND tv_series_id = $2`,
		userID, tvSeriesID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// qualifiedTvSeriesColumns is tvSeriesColumns with each column prefixed by "s." for use in JOINs.
const qualifiedTvSeriesColumns = `s.id, s.title, s.sort_title, s.overview, s.year, s.runtime,
	s.certification, s.network, s.genres, s.status, s.series_type, s.imdb_id, s.tmdb_id, s.tvdb_id,
	s.poster_url, s.first_aired, s.created_at, s.updated_at`

// ListTvSeriesByUser returns all TV series liked by a user.
func (d *DB) ListTvSeriesByUser(userID string) ([]TvSeries, error) {
	rows, err := d.Query(
		`SELECT `+qualifiedTvSeriesColumns+`
		 FROM tv_series s
		 JOIN user_tv_series us ON us.tv_series_id = s.id
		 WHERE us.user_id = $1
		 ORDER BY s.title ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var series []TvSeries
	for rows.Next() {
		s, err := scanTvSeries(rows)
		if err != nil {
			return nil, err
		}
		series = append(series, *s)
	}
	return series, rows.Err()
}
