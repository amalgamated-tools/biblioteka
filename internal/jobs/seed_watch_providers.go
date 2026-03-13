package jobs

// import (
// 	"context"
// 	"log/slog"

// 	"github.com/amalgamated-tools/biblioteka/internal/db"
// 	"github.com/amalgamated-tools/biblioteka/pkg/tmdb"
// )

// // SeedWatchProviders fetches the master list of movie and TV watch providers
// // from TMDB and upserts them into the watch_providers table. Errors are logged
// // but do not prevent the application from starting.
// func SeedWatchProviders(ctx context.Context, database *db.DB, tmdbClient *tmdb.Client) {
// 	if tmdbClient == nil {
// 		return
// 	}

// 	slog.InfoContext(ctx, "Seeding watch providers from TMDB")

// 	client := tmdbClient.ClientWithResponses()

// 	// Track which provider IDs appear in which list
// 	type providerEntry struct {
// 		ProviderName    string
// 		LogoPath        string
// 		DisplayPriority int
// 		InMovie         bool
// 		InTv            bool
// 	}
// 	providers := make(map[int]*providerEntry)

// 	// Fetch movie providers
// 	movieResp, err := client.WatchProvidersMovieListWithResponse(ctx, &tmdb.WatchProvidersMovieListParams{
// 		WatchRegion: "US",
// 	})
// 	if err != nil {
// 		slog.WarnContext(ctx, "Failed to fetch movie watch providers from TMDB", slog.Any("error", err))
// 	} else if movieResp.JSON200 == nil {
// 		slog.WarnContext(ctx, "Unexpected response fetching movie watch providers", slog.String("status", movieResp.Status()))
// 	} else {
// 		for _, p := range movieResp.JSON200.Results {
// 			entry, ok := providers[p.ProviderId]
// 			if !ok {
// 				logoPath := ""
// 				if p.LogoPath != "" {
// 					logoPath = tmdbLogoBase + p.LogoPath
// 				}
// 				entry = &providerEntry{
// 					ProviderName:    p.ProviderName,
// 					LogoPath:        logoPath,
// 					DisplayPriority: p.DisplayPriority,
// 				}
// 				providers[p.ProviderId] = entry
// 			}
// 			entry.InMovie = true
// 		}
// 		slog.InfoContext(ctx, "Fetched movie watch providers", slog.Int("count", len(movieResp.JSON200.Results)))
// 	}

// 	// Fetch TV providers
// 	tvResp, err := client.WatchProviderTvListWithResponse(ctx, &tmdb.WatchProviderTvListParams{
// 		WatchRegion: "US",
// 	})
// 	if err != nil {
// 		slog.WarnContext(ctx, "Failed to fetch TV watch providers from TMDB", slog.Any("error", err))
// 	} else if tvResp.JSON200 == nil {
// 		slog.WarnContext(ctx, "Unexpected response fetching TV watch providers", slog.String("status", tvResp.Status()))
// 	} else {
// 		for _, p := range tvResp.JSON200.Results {
// 			entry, ok := providers[p.ProviderId]
// 			if !ok {
// 				logoPath := ""
// 				if p.LogoPath != "" {
// 					logoPath = tmdbLogoBase + p.LogoPath
// 				}
// 				entry = &providerEntry{
// 					ProviderName:    p.ProviderName,
// 					LogoPath:        logoPath,
// 					DisplayPriority: p.DisplayPriority,
// 				}
// 				providers[p.ProviderId] = entry
// 			}
// 			entry.InTv = true
// 		}
// 		slog.InfoContext(ctx, "Fetched TV watch providers", slog.Int("count", len(tvResp.JSON200.Results)))
// 	}

// 	if len(providers) == 0 {
// 		slog.WarnContext(ctx, "No watch providers fetched from TMDB")
// 		return
// 	}

// 	// Build the slice for upsert
// 	var dbProviders []db.WatchProvider
// 	for id, entry := range providers {
// 		providerType := "both"
// 		if entry.InMovie && !entry.InTv {
// 			providerType = "movie"
// 		} else if !entry.InMovie && entry.InTv {
// 			providerType = "tv"
// 		}

// 		dbProviders = append(dbProviders, db.WatchProvider{
// 			ProviderID:      id,
// 			ProviderName:    entry.ProviderName,
// 			LogoPath:        entry.LogoPath,
// 			DisplayPriority: entry.DisplayPriority,
// 			ProviderType:    providerType,
// 		})
// 	}

// 	if err := database.UpsertWatchProviders(dbProviders); err != nil {
// 		slog.ErrorContext(ctx, "Failed to save watch providers to database", slog.Any("error", err))
// 		return
// 	}

// 	slog.InfoContext(ctx, "Watch providers seeded successfully", slog.Int("count", len(dbProviders)))
// }
