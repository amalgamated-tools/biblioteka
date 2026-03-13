package db

import (
	"database/sql"
	"testing"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func createTestMovie(t *testing.T, d *DB, title string, tmdbID *int) *Movie {
	t.Helper()
	m, err := d.CreateMovie(title, nil, nil, nil, nil, nil, nil, nil, nil, "released", nil, tmdbID, nil, nil)
	if err != nil {
		t.Fatalf("createTestMovie(%q): %v", title, err)
	}
	return m
}

func TestCreateMovie(t *testing.T) {
	d := newTestDB(t)

	tmdbID := 12345
	m, err := d.CreateMovie(
		"The Matrix", strPtr("Matrix, The"), strPtr("The Matrix"), strPtr("A sci-fi film"),
		intPtr(1999), intPtr(136), strPtr("R"), strPtr("Warner Bros"), strPtr("sci-fi,action"),
		"released", strPtr("tt0133093"), &tmdbID, strPtr("trailerID"), strPtr("http://poster.jpg"),
	)
	if err != nil {
		t.Fatalf("CreateMovie() error: %v", err)
	}
	if m.ID == "" {
		t.Error("CreateMovie() returned empty ID")
	}
	if m.Title != "The Matrix" {
		t.Errorf("Title = %q, want %q", m.Title, "The Matrix")
	}
	if m.TmdbID == nil || *m.TmdbID != tmdbID {
		t.Errorf("TmdbID = %v, want %d", m.TmdbID, tmdbID)
	}
}

func TestGetMovie(t *testing.T) {
	d := newTestDB(t)

	created := createTestMovie(t, d, "Inception", intPtr(27205))

	found, err := d.GetMovie(created.ID)
	if err != nil {
		t.Fatalf("GetMovie() error: %v", err)
	}
	if found.Title != "Inception" {
		t.Errorf("Title = %q, want %q", found.Title, "Inception")
	}
}

func TestGetMovie_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetMovie("nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetMovieByTmdbID(t *testing.T) {
	d := newTestDB(t)

	createTestMovie(t, d, "Interstellar", intPtr(157336))

	found, err := d.GetMovieByTmdbID(157336)
	if err != nil {
		t.Fatalf("GetMovieByTmdbID() error: %v", err)
	}
	if found.Title != "Interstellar" {
		t.Errorf("Title = %q, want %q", found.Title, "Interstellar")
	}
}

func TestGetMovieByImdbID(t *testing.T) {
	d := newTestDB(t)

	m, err := d.CreateMovie("Fight Club", nil, nil, nil, nil, nil, nil, nil, nil,
		"released", strPtr("tt0137523"), nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateMovie() error: %v", err)
	}

	found, err := d.GetMovieByImdbID("tt0137523")
	if err != nil {
		t.Fatalf("GetMovieByImdbID() error: %v", err)
	}
	if found.ID != m.ID {
		t.Errorf("ID = %q, want %q", found.ID, m.ID)
	}
}

func TestUpdateMovie(t *testing.T) {
	d := newTestDB(t)

	m := createTestMovie(t, d, "Old Title", nil)

	updated, err := d.UpdateMovie(m.ID, "New Title", nil, nil, nil, nil, nil, nil, nil, nil, "released", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateMovie() error: %v", err)
	}
	if updated.Title != "New Title" {
		t.Errorf("Title = %q, want %q", updated.Title, "New Title")
	}
}

func TestDeleteMovie(t *testing.T) {
	d := newTestDB(t)

	m := createTestMovie(t, d, "Temp Movie", nil)

	if err := d.DeleteMovie(m.ID); err != nil {
		t.Fatalf("DeleteMovie() error: %v", err)
	}

	_, err := d.GetMovie(m.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteMovie_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteMovie("nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestLikeAndListMovies(t *testing.T) {
	d := newTestDB(t)

	user, _ := d.CreateUser("Pat", "pat@example.com", "pw")
	m1 := createTestMovie(t, d, "Movie A", nil)
	m2 := createTestMovie(t, d, "Movie B", nil)

	if err := d.LikeMovie(user.ID, m1.ID); err != nil {
		t.Fatalf("LikeMovie() error: %v", err)
	}
	if err := d.LikeMovie(user.ID, m2.ID); err != nil {
		t.Fatalf("LikeMovie() error: %v", err)
	}

	movies, err := d.ListMoviesByUser(user.ID)
	if err != nil {
		t.Fatalf("ListMoviesByUser() error: %v", err)
	}
	if len(movies) != 2 {
		t.Errorf("expected 2 movies, got %d", len(movies))
	}
}

func TestLikeMovie_Idempotent(t *testing.T) {
	d := newTestDB(t)

	user, _ := d.CreateUser("Quinn", "quinn@example.com", "pw")
	m := createTestMovie(t, d, "Dupe Movie", nil)

	_ = d.LikeMovie(user.ID, m.ID)
	if err := d.LikeMovie(user.ID, m.ID); err != nil {
		t.Errorf("second LikeMovie() should be idempotent, got error: %v", err)
	}
}

func TestUnlikeMovie(t *testing.T) {
	d := newTestDB(t)

	user, _ := d.CreateUser("Rex", "rex@example.com", "pw")
	m := createTestMovie(t, d, "Liked Movie", nil)

	_ = d.LikeMovie(user.ID, m.ID)
	if err := d.UnlikeMovie(user.ID, m.ID); err != nil {
		t.Fatalf("UnlikeMovie() error: %v", err)
	}

	movies, _ := d.ListMoviesByUser(user.ID)
	if len(movies) != 0 {
		t.Errorf("expected 0 movies after unlike, got %d", len(movies))
	}
}

func TestUnlikeMovie_NotFound(t *testing.T) {
	d := newTestDB(t)

	user, _ := d.CreateUser("Sam", "sam@example.com", "pw")
	m := createTestMovie(t, d, "Not Liked Movie", nil)

	err := d.UnlikeMovie(user.ID, m.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
