package db

// protocolCredentialColumns is the shared SELECT column list for all
// protocol credential tables (opds_credentials, kosync_credentials, …).
// Every table must expose these six columns in this exact order.
const protocolCredentialColumns = `id, user_id, username, password_hash, created_at, updated_at`

// scanProtocolCredentialFields scans the six common protocol-credential
// columns into the provided pointer fields.  Call it from each protocol's
// own scan wrapper to avoid duplicating the Scan call.
func scanProtocolCredentialFields(
	row interface{ Scan(...any) error },
	id, userID, username, passwordHash *string,
	createdAt, updatedAt *Timestamp,
) error {
	return row.Scan(id, userID, username, passwordHash, createdAt, updatedAt)
}
