package db

import "database/sql"

// Movie represents a row in the movies table.
type Movie struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	SortTitle        *string   `json:"sort_title"`
	OriginalTitle    *string   `json:"original_title"`
	Overview         *string   `json:"overview"`
	Year             *int      `json:"year"`
	Runtime          *int      `json:"runtime"`
	Certification    *string   `json:"certification"`
	Studio           *string   `json:"studio"`
	Genres           *string   `json:"genres"`
	Status           string    `json:"status"`
	ImdbID           *string   `json:"imdb_id"`
	TmdbID           *int      `json:"tmdb_id"`
	YouTubeTrailerID *string   `json:"youtube_trailer_id"`
	PosterURL        *string   `json:"poster_url"`
	CreatedAt        Timestamp `json:"created_at"`
	UpdatedAt        Timestamp `json:"updated_at"`
}

const movieColumns = `id, title, sort_title, original_title, overview, year, runtime,
	certification, studio, genres, status, imdb_id, tmdb_id, youtube_trailer_id, poster_url,
	created_at, updated_at`

func scanMovie(row interface{ Scan(dest ...any) error }) (*Movie, error) {
	var m Movie
	err := row.Scan(
		&m.ID, &m.Title, &m.SortTitle, &m.OriginalTitle, &m.Overview,
		&m.Year, &m.Runtime, &m.Certification, &m.Studio, &m.Genres, &m.Status,
		&m.ImdbID, &m.TmdbID, &m.YouTubeTrailerID, &m.PosterURL,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// CreateMovie inserts a new movie and returns it.
func (d *DB) CreateMovie(title string, sortTitle, originalTitle, overview *string,
	year, runtime *int, certification, studio, genres *string, status string,
	imdbID *string, tmdbID *int, youtubeTrailerID, posterURL *string) (*Movie, error) {

	return scanMovie(d.QueryRow(
		`INSERT INTO movies (title, sort_title, original_title, overview, year, runtime,
			certification, studio, genres, status, imdb_id, tmdb_id, youtube_trailer_id, poster_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		 RETURNING `+movieColumns,
		title, sortTitle, originalTitle, overview, year, runtime,
		certification, studio, genres, status, imdbID, tmdbID, youtubeTrailerID, posterURL,
	))
}

// GetMovie returns a single movie by ID.
func (d *DB) GetMovie(id string) (*Movie, error) {
	return scanMovie(d.QueryRow(
		`SELECT `+movieColumns+` FROM movies WHERE id = $1`, id,
	))
}

// GetMovieByTmdbID returns a movie by TMDB ID.
func (d *DB) GetMovieByTmdbID(tmdbID int) (*Movie, error) {
	return scanMovie(d.QueryRow(
		`SELECT `+movieColumns+` FROM movies WHERE tmdb_id = $1`, tmdbID,
	))
}

// GetMovieByImdbID returns a movie by IMDB ID.
func (d *DB) GetMovieByImdbID(imdbID string) (*Movie, error) {
	return scanMovie(d.QueryRow(
		`SELECT `+movieColumns+` FROM movies WHERE imdb_id = $1`, imdbID,
	))
}

// UpdateMovie updates an existing movie.
func (d *DB) UpdateMovie(id, title string, sortTitle, originalTitle, overview *string,
	year, runtime *int, certification, studio, genres *string, status string,
	imdbID *string, tmdbID *int, youtubeTrailerID, posterURL *string) (*Movie, error) {

	return scanMovie(d.QueryRow(
		`UPDATE movies
		 SET title = $1, sort_title = $2, original_title = $3, overview = $4, year = $5, runtime = $6,
			 certification = $7, studio = $8, genres = $9, status = $10,
			 imdb_id = $11, tmdb_id = $12, youtube_trailer_id = $13, poster_url = $14,
			 updated_at = `+d.now()+`
		 WHERE id = $15
		 RETURNING `+movieColumns,
		title, sortTitle, originalTitle, overview, year, runtime,
		certification, studio, genres, status,
		imdbID, tmdbID, youtubeTrailerID, posterURL,
		id,
	))
}

// DeleteMovie deletes a movie by ID.
func (d *DB) DeleteMovie(id string) error {
	res, err := d.Exec(`DELETE FROM movies WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// LikeMovie associates a user with a movie.
func (d *DB) LikeMovie(userID, movieID string) error {
	query := `INSERT OR IGNORE INTO user_movies (user_id, movie_id) VALUES ($1, $2)`
	if d.Dialect == DialectPostgres {
		query = `INSERT INTO user_movies (user_id, movie_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	}
	_, err := d.Exec(query, userID, movieID)
	return err
}

// UnlikeMovie removes the association between a user and a movie.
func (d *DB) UnlikeMovie(userID, movieID string) error {
	res, err := d.Exec(
		`DELETE FROM user_movies WHERE user_id = $1 AND movie_id = $2`,
		userID, movieID,
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

// qualifiedMovieColumns is movieColumns with each column prefixed by "m." for use in JOINs.
const qualifiedMovieColumns = `m.id, m.title, m.sort_title, m.original_title, m.overview, m.year, m.runtime,
	m.certification, m.studio, m.genres, m.status, m.imdb_id, m.tmdb_id, m.youtube_trailer_id, m.poster_url,
	m.created_at, m.updated_at`

// ListMoviesByUser returns all movies liked by a user.
func (d *DB) ListMoviesByUser(userID string) ([]Movie, error) {
	rows, err := d.Query(
		`SELECT `+qualifiedMovieColumns+` FROM movies m
		 JOIN user_movies um ON um.movie_id = m.id
		 WHERE um.user_id = $1
		 ORDER BY m.title ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var movies []Movie
	for rows.Next() {
		m, err := scanMovie(rows)
		if err != nil {
			return nil, err
		}
		movies = append(movies, *m)
	}
	return movies, rows.Err()
}
