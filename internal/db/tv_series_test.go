package db

import (
	"database/sql"
	"testing"
)

func createTestTvSeries(t *testing.T, d *DB, title string, tmdbID *int) *TvSeries {
	t.Helper()
	s, err := d.CreateTvSeries(title, nil, nil, nil, nil, nil, nil, nil, "continuing", "standard", nil, tmdbID, nil, nil, nil)
	if err != nil {
		t.Fatalf("createTestTvSeries(%q): %v", title, err)
	}
	return s
}

func TestCreateTvSeries(t *testing.T) {
	d := newTestDB(t)

	tmdbID := 1396
	s, err := d.CreateTvSeries(
		"Breaking Bad", strPtr("Breaking Bad"), strPtr("A meth drama"),
		intPtr(2008), intPtr(45), strPtr("TV-MA"), strPtr("AMC"), strPtr("crime,drama"),
		"ended", "standard", strPtr("tt0903747"), &tmdbID, intPtr(81189),
		strPtr("http://poster.jpg"), strPtr("2008-01-20"),
	)
	if err != nil {
		t.Fatalf("CreateTvSeries() error: %v", err)
	}
	if s.ID == "" {
		t.Error("CreateTvSeries() returned empty ID")
	}
	if s.Title != "Breaking Bad" {
		t.Errorf("Title = %q, want %q", s.Title, "Breaking Bad")
	}
	if s.TmdbID == nil || *s.TmdbID != tmdbID {
		t.Errorf("TmdbID = %v, want %d", s.TmdbID, tmdbID)
	}
}

func TestGetTvSeries(t *testing.T) {
	d := newTestDB(t)

	created := createTestTvSeries(t, d, "Succession", intPtr(63333))

	found, err := d.GetTvSeries(created.ID)
	if err != nil {
		t.Fatalf("GetTvSeries() error: %v", err)
	}
	if found.Title != "Succession" {
		t.Errorf("Title = %q, want %q", found.Title, "Succession")
	}
}

func TestGetTvSeries_NotFound(t *testing.T) {
	d := newTestDB(t)

	_, err := d.GetTvSeries("nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestGetTvSeriesByTmdbID(t *testing.T) {
	d := newTestDB(t)

	createTestTvSeries(t, d, "The Wire", intPtr(1438))

	found, err := d.GetTvSeriesByTmdbID(1438)
	if err != nil {
		t.Fatalf("GetTvSeriesByTmdbID() error: %v", err)
	}
	if found.Title != "The Wire" {
		t.Errorf("Title = %q, want %q", found.Title, "The Wire")
	}
}

func TestGetTvSeriesByImdbID(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateTvSeries("Sopranos", nil, nil, nil, nil, nil, nil, nil,
		"ended", "standard", strPtr("tt0141842"), nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateTvSeries() error: %v", err)
	}

	found, err := d.GetTvSeriesByImdbID("tt0141842")
	if err != nil {
		t.Fatalf("GetTvSeriesByImdbID() error: %v", err)
	}
	if found.Title != "Sopranos" {
		t.Errorf("Title = %q, want %q", found.Title, "Sopranos")
	}
}

func TestGetTvSeriesByTvdbID(t *testing.T) {
	d := newTestDB(t)

	_, err := d.CreateTvSeries("Game of Thrones", nil, nil, nil, nil, nil, nil, nil,
		"ended", "standard", nil, nil, intPtr(121361), nil, nil)
	if err != nil {
		t.Fatalf("CreateTvSeries() error: %v", err)
	}

	found, err := d.GetTvSeriesByTvdbID(121361)
	if err != nil {
		t.Fatalf("GetTvSeriesByTvdbID() error: %v", err)
	}
	if found.Title != "Game of Thrones" {
		t.Errorf("Title = %q, want %q", found.Title, "Game of Thrones")
	}
}

func TestUpdateTvSeries(t *testing.T) {
	d := newTestDB(t)

	s := createTestTvSeries(t, d, "Old Series", nil)

	updated, err := d.UpdateTvSeries(s.ID, "New Series", nil, nil, nil, nil, nil, nil, nil, "ended", "standard", nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateTvSeries() error: %v", err)
	}
	if updated.Title != "New Series" {
		t.Errorf("Title = %q, want %q", updated.Title, "New Series")
	}
	if updated.Status != "ended" {
		t.Errorf("Status = %q, want %q", updated.Status, "ended")
	}
}

func TestDeleteTvSeries(t *testing.T) {
	d := newTestDB(t)

	s := createTestTvSeries(t, d, "Temp Series", nil)

	if err := d.DeleteTvSeries(s.ID); err != nil {
		t.Fatalf("DeleteTvSeries() error: %v", err)
	}

	_, err := d.GetTvSeries(s.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteTvSeries_NotFound(t *testing.T) {
	d := newTestDB(t)

	err := d.DeleteTvSeries("nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestLikeAndListTvSeries(t *testing.T) {
	d := newTestDB(t)

	user, _ := d.CreateUser("Tina", "tina@example.com", "pw")
	s1 := createTestTvSeries(t, d, "Series A", nil)
	s2 := createTestTvSeries(t, d, "Series B", nil)

	if err := d.LikeTvSeries(user.ID, s1.ID); err != nil {
		t.Fatalf("LikeTvSeries() error: %v", err)
	}
	if err := d.LikeTvSeries(user.ID, s2.ID); err != nil {
		t.Fatalf("LikeTvSeries() error: %v", err)
	}

	series, err := d.ListTvSeriesByUser(user.ID)
	if err != nil {
		t.Fatalf("ListTvSeriesByUser() error: %v", err)
	}
	if len(series) != 2 {
		t.Errorf("expected 2 TV series, got %d", len(series))
	}
}

func TestLikeTvSeries_Idempotent(t *testing.T) {
	d := newTestDB(t)

	user, _ := d.CreateUser("Uma", "uma@example.com", "pw")
	s := createTestTvSeries(t, d, "Dupe Series", nil)

	_ = d.LikeTvSeries(user.ID, s.ID)
	if err := d.LikeTvSeries(user.ID, s.ID); err != nil {
		t.Errorf("second LikeTvSeries() should be idempotent, got error: %v", err)
	}
}

func TestUnlikeTvSeries(t *testing.T) {
	d := newTestDB(t)

	user, _ := d.CreateUser("Vera", "vera@example.com", "pw")
	s := createTestTvSeries(t, d, "Liked Series", nil)

	_ = d.LikeTvSeries(user.ID, s.ID)
	if err := d.UnlikeTvSeries(user.ID, s.ID); err != nil {
		t.Fatalf("UnlikeTvSeries() error: %v", err)
	}

	series, _ := d.ListTvSeriesByUser(user.ID)
	if len(series) != 0 {
		t.Errorf("expected 0 series after unlike, got %d", len(series))
	}
}

func TestUnlikeTvSeries_NotFound(t *testing.T) {
	d := newTestDB(t)

	user, _ := d.CreateUser("Will", "will@example.com", "pw")
	s := createTestTvSeries(t, d, "Not Liked Series", nil)

	err := d.UnlikeTvSeries(user.ID, s.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}
