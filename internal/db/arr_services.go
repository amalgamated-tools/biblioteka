package db

import "database/sql"

// ArrServiceType represents the type of *arr service.
type ArrServiceType string

const (
	ArrServiceTypeRadarr   ArrServiceType = "radarr"
	ArrServiceTypeSonarr   ArrServiceType = "sonarr"
	ArrServiceTypeProwlarr ArrServiceType = "prowlarr"
	ArrServiceTypeSeerr    ArrServiceType = "seerr"
)

// ArrService represents a row in the media_services table.
type ArrService struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      ArrServiceType `json:"type"`
	URL       string         `json:"url"`
	APIKey    string         `json:"api_key"`
	CreatedAt Timestamp      `json:"created_at"`
	UpdatedAt Timestamp      `json:"updated_at"`
}

// CreateArrService inserts a new *arr service and returns it.
func (d *DB) CreateArrService(name string, serviceType ArrServiceType, url, apiKey string) (*ArrService, error) {
	var s ArrService
	err := d.QueryRow(
		`INSERT INTO media_services (name, type, url, api_key)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, type, url, api_key, created_at, updated_at`,
		name, string(serviceType), url, apiKey,
	).Scan(&s.ID, &s.Name, &s.Type, &s.URL, &s.APIKey, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListArrServices returns all *arr services.
func (d *DB) ListArrServices() ([]ArrService, error) {
	rows, err := d.Query(
		`SELECT id, name, type, url, api_key, created_at, updated_at
		 FROM media_services
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var services []ArrService
	for rows.Next() {
		var s ArrService
		if err := rows.Scan(&s.ID, &s.Name, &s.Type, &s.URL, &s.APIKey, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, rows.Err()
}

// GetArrService returns a single *arr service by ID.
func (d *DB) GetArrService(id string) (*ArrService, error) {
	var s ArrService
	err := d.QueryRow(
		`SELECT id, name, type, url, api_key, created_at, updated_at
		 FROM media_services
		 WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.Name, &s.Type, &s.URL, &s.APIKey, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UpdateArrService updates an existing *arr service.
func (d *DB) UpdateArrService(id, name string, serviceType ArrServiceType, url, apiKey string) (*ArrService, error) {
	var s ArrService
	err := d.QueryRow(
		`UPDATE media_services
		 SET name = $1, type = $2, url = $3, api_key = $4, updated_at = `+d.now()+`
		 WHERE id = $5
		 RETURNING id, name, type, url, api_key, created_at, updated_at`,
		name, string(serviceType), url, apiKey, id,
	).Scan(&s.ID, &s.Name, &s.Type, &s.URL, &s.APIKey, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteArrService deletes an *arr service by ID.
func (d *DB) DeleteArrService(id string) error {
	res, err := d.Exec(`DELETE FROM media_services WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
