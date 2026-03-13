package db

import (
	"testing"
)

func TestSaveAndGetTvSeriesWatchProviders(t *testing.T) {
	d := newTestDB(t)

	s := createTestTvSeries(t, d, "Breaking Bad", intPtr(1396))

	providers := []TvSeriesWatchProvider{
		{TvSeriesID: s.ID, ProviderName: "Netflix", ProviderID: 8, LogoPath: "/netflix.jpg", ProviderType: "stream", DisplayPriority: 1},
		{TvSeriesID: s.ID, ProviderName: "Amazon Prime Video", ProviderID: 9, LogoPath: "/prime.jpg", ProviderType: "stream", DisplayPriority: 2},
		{TvSeriesID: s.ID, ProviderName: "Apple TV Plus", ProviderID: 350, LogoPath: "/apple.jpg", ProviderType: "buy", DisplayPriority: 1},
	}

	if err := d.SaveTvSeriesWatchProviders(s.ID, providers); err != nil {
		t.Fatalf("SaveTvSeriesWatchProviders() error: %v", err)
	}

	got, err := d.GetTvSeriesWatchProviders(s.ID)
	if err != nil {
		t.Fatalf("GetTvSeriesWatchProviders() error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(got))
	}
	for _, p := range got {
		if p.ID == "" {
			t.Error("provider ID should not be empty")
		}
		if p.TvSeriesID != s.ID {
			t.Errorf("TvSeriesID = %q, want %q", p.TvSeriesID, s.ID)
		}
	}
}

func TestSaveTvSeriesWatchProviders_ReplacesExisting(t *testing.T) {
	d := newTestDB(t)

	s := createTestTvSeries(t, d, "The Wire", intPtr(1438))

	initial := []TvSeriesWatchProvider{
		{TvSeriesID: s.ID, ProviderName: "Netflix", ProviderID: 8, LogoPath: "/netflix.jpg", ProviderType: "stream", DisplayPriority: 1},
		{TvSeriesID: s.ID, ProviderName: "Hulu", ProviderID: 15, LogoPath: "/hulu.jpg", ProviderType: "stream", DisplayPriority: 2},
	}
	if err := d.SaveTvSeriesWatchProviders(s.ID, initial); err != nil {
		t.Fatalf("initial SaveTvSeriesWatchProviders() error: %v", err)
	}

	replacement := []TvSeriesWatchProvider{
		{TvSeriesID: s.ID, ProviderName: "Disney Plus", ProviderID: 337, LogoPath: "/disney.jpg", ProviderType: "stream", DisplayPriority: 1},
	}
	if err := d.SaveTvSeriesWatchProviders(s.ID, replacement); err != nil {
		t.Fatalf("replacement SaveTvSeriesWatchProviders() error: %v", err)
	}

	got, err := d.GetTvSeriesWatchProviders(s.ID)
	if err != nil {
		t.Fatalf("GetTvSeriesWatchProviders() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 provider after replacement, got %d", len(got))
	}
	if got[0].ProviderName != "Disney Plus" {
		t.Errorf("ProviderName = %q, want %q", got[0].ProviderName, "Disney Plus")
	}
}

func TestSaveTvSeriesWatchProviders_EmptyClearsAll(t *testing.T) {
	d := newTestDB(t)

	s := createTestTvSeries(t, d, "Succession", intPtr(63333))

	providers := []TvSeriesWatchProvider{
		{TvSeriesID: s.ID, ProviderName: "Netflix", ProviderID: 8, LogoPath: "/netflix.jpg", ProviderType: "stream", DisplayPriority: 1},
	}
	if err := d.SaveTvSeriesWatchProviders(s.ID, providers); err != nil {
		t.Fatalf("SaveTvSeriesWatchProviders() error: %v", err)
	}

	if err := d.SaveTvSeriesWatchProviders(s.ID, nil); err != nil {
		t.Fatalf("SaveTvSeriesWatchProviders(nil) error: %v", err)
	}

	got, err := d.GetTvSeriesWatchProviders(s.ID)
	if err != nil {
		t.Fatalf("GetTvSeriesWatchProviders() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 providers after clear, got %d", len(got))
	}
}

func TestGetTvSeriesWatchProviders_Empty(t *testing.T) {
	d := newTestDB(t)

	s := createTestTvSeries(t, d, "Unfetched Series", nil)

	got, err := d.GetTvSeriesWatchProviders(s.ID)
	if err != nil {
		t.Fatalf("GetTvSeriesWatchProviders() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 providers, got %d", len(got))
	}
}

func TestGetTvSeriesWatchProvidersByTmdbID(t *testing.T) {
	d := newTestDB(t)

	s := createTestTvSeries(t, d, "Chernobyl", intPtr(87108))

	providers := []TvSeriesWatchProvider{
		{TvSeriesID: s.ID, ProviderName: "HBO Max", ProviderID: 384, LogoPath: "/hbo.jpg", ProviderType: "stream", DisplayPriority: 1},
		{TvSeriesID: s.ID, ProviderName: "Apple TV Plus", ProviderID: 350, LogoPath: "/apple.jpg", ProviderType: "buy", DisplayPriority: 1},
	}
	if err := d.SaveTvSeriesWatchProviders(s.ID, providers); err != nil {
		t.Fatalf("SaveTvSeriesWatchProviders() error: %v", err)
	}

	got, err := d.GetTvSeriesWatchProvidersByTmdbID(87108)
	if err != nil {
		t.Fatalf("GetTvSeriesWatchProvidersByTmdbID() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(got))
	}
}

func TestGetTvSeriesWatchProvidersByTmdbID_NoMatch(t *testing.T) {
	d := newTestDB(t)

	got, err := d.GetTvSeriesWatchProvidersByTmdbID(99999)
	if err != nil {
		t.Fatalf("GetTvSeriesWatchProvidersByTmdbID() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 providers, got %d", len(got))
	}
}

func TestIsTvSeriesProvidersFetched_NotFetched(t *testing.T) {
	d := newTestDB(t)

	createTestTvSeries(t, d, "Unfetched Series", intPtr(11111))

	fetched, err := d.IsTvSeriesProvidersFetched(11111)
	if err != nil {
		t.Fatalf("IsTvSeriesProvidersFetched() error: %v", err)
	}
	if fetched {
		t.Error("expected false before providers are fetched")
	}
}

func TestIsTvSeriesProvidersFetched_AfterSave(t *testing.T) {
	d := newTestDB(t)

	s := createTestTvSeries(t, d, "Fetched Series", intPtr(22222))

	if err := d.SaveTvSeriesWatchProviders(s.ID, nil); err != nil {
		t.Fatalf("SaveTvSeriesWatchProviders() error: %v", err)
	}

	fetched, err := d.IsTvSeriesProvidersFetched(22222)
	if err != nil {
		t.Fatalf("IsTvSeriesProvidersFetched() error: %v", err)
	}
	if !fetched {
		t.Error("expected true after SaveTvSeriesWatchProviders sets providers_fetched_at")
	}
}

func TestIsTvSeriesProvidersFetched_SeriesNotInDB(t *testing.T) {
	d := newTestDB(t)

	fetched, err := d.IsTvSeriesProvidersFetched(99999)
	if err != nil {
		t.Fatalf("IsTvSeriesProvidersFetched() on missing series error: %v", err)
	}
	if fetched {
		t.Error("expected false when series is not in the database")
	}
}
