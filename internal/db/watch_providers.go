package db

// WatchProvider represents a row in the watch_providers table.
// This is a master list of all available streaming/rental/purchase providers
// from TMDB, not tied to any specific movie or TV series.
type WatchProvider struct {
	ProviderID      int       `json:"provider_id"`
	ProviderName    string    `json:"provider_name"`
	LogoPath        string    `json:"logo_path"`
	DisplayPriority int       `json:"display_priority"`
	ProviderType    string    `json:"provider_type"` // "movie", "tv", or "both"
	CreatedAt       Timestamp `json:"created_at"`
	UpdatedAt       Timestamp `json:"updated_at"`
}

// UpsertWatchProviders bulk upserts watch providers into the watch_providers table.
func (d *DB) UpsertWatchProviders(providers []WatchProvider) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.Prepare(
		`INSERT INTO watch_providers (provider_id, provider_name, logo_path, display_priority, provider_type)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT(provider_id) DO UPDATE SET
			provider_name = excluded.provider_name,
			logo_path = excluded.logo_path,
			display_priority = excluded.display_priority,
			provider_type = excluded.provider_type,
			updated_at = ` + d.now(),
	)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for _, p := range providers {
		if _, err := stmt.Exec(p.ProviderID, p.ProviderName, p.LogoPath, p.DisplayPriority, p.ProviderType); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetWatchProviders returns all watch providers ordered by provider name and then display priority.
func (d *DB) GetWatchProviders() ([]WatchProvider, error) {
	rows, err := d.Query(
		`SELECT provider_id, provider_name, logo_path, display_priority, provider_type, created_at, updated_at
		 FROM watch_providers
		 ORDER BY provider_name, display_priority`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var providers []WatchProvider
	for rows.Next() {
		var p WatchProvider
		if err := rows.Scan(&p.ProviderID, &p.ProviderName, &p.LogoPath, &p.DisplayPriority, &p.ProviderType, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

// GetUserWatchProviders returns the watch providers selected by a user.
func (d *DB) GetUserWatchProviders(userID string) ([]WatchProvider, error) {
	rows, err := d.Query(
		`SELECT wp.provider_id, wp.provider_name, wp.logo_path, wp.display_priority, wp.provider_type, wp.created_at, wp.updated_at
		 FROM watch_providers wp
		 INNER JOIN user_watch_providers uwp ON wp.provider_id = uwp.provider_id
		 WHERE uwp.user_id = $1
		 ORDER BY wp.display_priority, wp.provider_name`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var providers []WatchProvider
	for rows.Next() {
		var p WatchProvider
		if err := rows.Scan(&p.ProviderID, &p.ProviderName, &p.LogoPath, &p.DisplayPriority, &p.ProviderType, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

// ValidateProviderIDs checks which of the given IDs exist in watch_providers
// and returns any that do not.
func (d *DB) ValidateProviderIDs(ids []int) (invalid []int, err error) {
	if len(ids) == 0 {
		return nil, nil
	}

	existing := make(map[int]bool)
	rows, err := d.Query(`SELECT provider_id FROM watch_providers`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		existing[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, id := range ids {
		if !existing[id] {
			invalid = append(invalid, id)
		}
	}
	return invalid, nil
}

// SetUserWatchProviders replaces a user's selected watch providers.
func (d *DB) SetUserWatchProviders(userID string, providerIDs []int) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(`DELETE FROM user_watch_providers WHERE user_id = $1`, userID); err != nil {
		return err
	}

	if len(providerIDs) > 0 {
		query := `INSERT OR IGNORE INTO user_watch_providers (user_id, provider_id) VALUES ($1, $2)`
		if d.Dialect == DialectPostgres {
			query = `INSERT INTO user_watch_providers (user_id, provider_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
		}
		stmt, err := tx.Prepare(query)
		if err != nil {
			return err
		}
		defer func() { _ = stmt.Close() }()

		for _, pid := range providerIDs {
			if _, err := stmt.Exec(userID, pid); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
