package db

// GetSetting retrieves a setting value by key.
// Returns sql.ErrNoRows if the key does not exist.
func (d *DB) GetSetting(key string) (string, error) {
	var value string
	err := d.QueryRow("SELECT value FROM settings WHERE key = $1", key).Scan(&value)
	return value, err
}

// SetSetting upserts a setting key-value pair.
func (d *DB) SetSetting(key, value string) error {
	_, err := d.Exec(
		`INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, `+d.now()+`)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value,
	)
	return err
}
