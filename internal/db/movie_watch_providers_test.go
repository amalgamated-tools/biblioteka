package db

import (
	"testing"
)

func TestSaveAndGetMovieWatchProviders(t *testing.T) {
	d := newTestDB(t)

	m := createTestMovie(t, d, "The Matrix", intPtr(603))

	providers := []MovieWatchProvider{
		{MovieID: m.ID, ProviderName: "Netflix", ProviderID: 8, LogoPath: "/netflix.jpg", ProviderType: "stream", DisplayPriority: 1},
		{MovieID: m.ID, ProviderName: "Amazon Prime Video", ProviderID: 9, LogoPath: "/prime.jpg", ProviderType: "stream", DisplayPriority: 2},
		{MovieID: m.ID, ProviderName: "Vudu", ProviderID: 7, LogoPath: "/vudu.jpg", ProviderType: "rent", DisplayPriority: 1},
	}

	if err := d.SaveMovieWatchProviders(m.ID, providers); err != nil {
		t.Fatalf("SaveMovieWatchProviders() error: %v", err)
	}

	got, err := d.GetMovieWatchProviders(m.ID)
	if err != nil {
		t.Fatalf("GetMovieWatchProviders() error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(got))
	}
	for _, p := range got {
		if p.ID == "" {
			t.Error("provider ID should not be empty")
		}
		if p.MovieID != m.ID {
			t.Errorf("MovieID = %q, want %q", p.MovieID, m.ID)
		}
	}
}

func TestSaveMovieWatchProviders_ReplacesExisting(t *testing.T) {
	d := newTestDB(t)

	m := createTestMovie(t, d, "Inception", intPtr(27205))

	initial := []MovieWatchProvider{
		{MovieID: m.ID, ProviderName: "Netflix", ProviderID: 8, LogoPath: "/netflix.jpg", ProviderType: "stream", DisplayPriority: 1},
		{MovieID: m.ID, ProviderName: "Hulu", ProviderID: 15, LogoPath: "/hulu.jpg", ProviderType: "stream", DisplayPriority: 2},
	}
	if err := d.SaveMovieWatchProviders(m.ID, initial); err != nil {
		t.Fatalf("initial SaveMovieWatchProviders() error: %v", err)
	}

	replacement := []MovieWatchProvider{
		{MovieID: m.ID, ProviderName: "Disney Plus", ProviderID: 337, LogoPath: "/disney.jpg", ProviderType: "stream", DisplayPriority: 1},
	}
	if err := d.SaveMovieWatchProviders(m.ID, replacement); err != nil {
		t.Fatalf("replacement SaveMovieWatchProviders() error: %v", err)
	}

	got, err := d.GetMovieWatchProviders(m.ID)
	if err != nil {
		t.Fatalf("GetMovieWatchProviders() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 provider after replacement, got %d", len(got))
	}
	if got[0].ProviderName != "Disney Plus" {
		t.Errorf("ProviderName = %q, want %q", got[0].ProviderName, "Disney Plus")
	}
}

func TestSaveMovieWatchProviders_EmptyClearsAll(t *testing.T) {
	d := newTestDB(t)

	m := createTestMovie(t, d, "Interstellar", intPtr(157336))

	providers := []MovieWatchProvider{
		{MovieID: m.ID, ProviderName: "Netflix", ProviderID: 8, LogoPath: "/netflix.jpg", ProviderType: "stream", DisplayPriority: 1},
	}
	if err := d.SaveMovieWatchProviders(m.ID, providers); err != nil {
		t.Fatalf("SaveMovieWatchProviders() error: %v", err)
	}

	if err := d.SaveMovieWatchProviders(m.ID, nil); err != nil {
		t.Fatalf("SaveMovieWatchProviders(nil) error: %v", err)
	}

	got, err := d.GetMovieWatchProviders(m.ID)
	if err != nil {
		t.Fatalf("GetMovieWatchProviders() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 providers after clear, got %d", len(got))
	}
}

func TestGetMovieWatchProviders_Empty(t *testing.T) {
	d := newTestDB(t)

	m := createTestMovie(t, d, "Fight Club", nil)

	got, err := d.GetMovieWatchProviders(m.ID)
	if err != nil {
		t.Fatalf("GetMovieWatchProviders() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 providers, got %d", len(got))
	}
}

func TestGetMovieWatchProvidersByTmdbID(t *testing.T) {
	d := newTestDB(t)

	m := createTestMovie(t, d, "The Shawshank Redemption", intPtr(278))

	providers := []MovieWatchProvider{
		{MovieID: m.ID, ProviderName: "Netflix", ProviderID: 8, LogoPath: "/netflix.jpg", ProviderType: "stream", DisplayPriority: 1},
		{MovieID: m.ID, ProviderName: "Amazon Prime Video", ProviderID: 9, LogoPath: "/prime.jpg", ProviderType: "rent", DisplayPriority: 1},
	}
	if err := d.SaveMovieWatchProviders(m.ID, providers); err != nil {
		t.Fatalf("SaveMovieWatchProviders() error: %v", err)
	}

	got, err := d.GetMovieWatchProvidersByTmdbID(278)
	if err != nil {
		t.Fatalf("GetMovieWatchProvidersByTmdbID() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(got))
	}
}

func TestGetMovieWatchProvidersByTmdbID_NoMatch(t *testing.T) {
	d := newTestDB(t)

	got, err := d.GetMovieWatchProvidersByTmdbID(99999)
	if err != nil {
		t.Fatalf("GetMovieWatchProvidersByTmdbID() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 providers, got %d", len(got))
	}
}

func TestIsProvidersFetched_NotFetched(t *testing.T) {
	d := newTestDB(t)

	createTestMovie(t, d, "Unfetched Movie", intPtr(11111))

	fetched, err := d.IsProvidersFetched(11111)
	if err != nil {
		t.Fatalf("IsProvidersFetched() error: %v", err)
	}
	if fetched {
		t.Error("expected false before providers are fetched")
	}
}

func TestIsProvidersFetched_AfterSave(t *testing.T) {
	d := newTestDB(t)

	m := createTestMovie(t, d, "Fetched Movie", intPtr(22222))

	if err := d.SaveMovieWatchProviders(m.ID, nil); err != nil {
		t.Fatalf("SaveMovieWatchProviders() error: %v", err)
	}

	fetched, err := d.IsProvidersFetched(22222)
	if err != nil {
		t.Fatalf("IsProvidersFetched() error: %v", err)
	}
	if !fetched {
		t.Error("expected true after SaveMovieWatchProviders sets providers_fetched_at")
	}
}

func TestIsProvidersFetched_MovieNotInDB(t *testing.T) {
	d := newTestDB(t)

	fetched, err := d.IsProvidersFetched(99999)
	if err != nil {
		t.Fatalf("IsProvidersFetched() on missing movie error: %v", err)
	}
	if fetched {
		t.Error("expected false when movie is not in the database")
	}
}
