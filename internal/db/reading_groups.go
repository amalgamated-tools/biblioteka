package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// Sentinel errors for reading group operations.
var (
	ErrInvalidGroupName      = errors.New("invalid group name")
	ErrGroupNameExists       = errors.New("group name already exists")
	ErrGroupNotFound         = errors.New("group not found")
	ErrMemberUserNotFound    = errors.New("member user not found")
	ErrNotGroupMember        = errors.New("not a group member")
	ErrAlreadyGroupMember    = errors.New("already a group member")
	ErrOwnerCannotLeaveGroup = errors.New("owner cannot leave their own group")
)

// NormalizeGroupName normalizes a group name.
func NormalizeGroupName(name string) string { return normalizeName(name) }

// ReadingGroup represents a row in the reading_groups table.
type ReadingGroup struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	MemberCount int       `json:"member_count"`
	CreatedAt   Timestamp `json:"created_at"`
	UpdatedAt   Timestamp `json:"updated_at"`
}

// ReadingGroupMember represents a member of a reading group.
type ReadingGroupMember struct {
	GroupID  string    `json:"group_id"`
	UserID   string    `json:"user_id"`
	UserName string    `json:"user_name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	JoinedAt Timestamp `json:"joined_at"`
}

func scanReadingGroup(row interface{ Scan(...any) error }) (*ReadingGroup, error) {
	return scanRow(row, func(g *ReadingGroup) []any {
		return []any{&g.ID, &g.OwnerID, &g.Name, &g.Description, &g.MemberCount, &g.CreatedAt, &g.UpdatedAt}
	})
}

func scanReadingGroupMember(row interface{ Scan(...any) error }) (*ReadingGroupMember, error) {
	return scanRow(row, func(m *ReadingGroupMember) []any {
		return []any{&m.GroupID, &m.UserID, &m.UserName, &m.Email, &m.Role, &m.JoinedAt}
	})
}

// CreateGroup creates a new reading group and inserts the owner as a member with role="owner".
func (d *DB) CreateGroup(ctx context.Context, ownerID, name string, description *string) (*ReadingGroup, error) {
	name = NormalizeGroupName(name)
	if name == "" {
		slog.WarnContext(ctx, "db: rejecting group with blank name after normalization",
			slog.String(otelkeys.UserID, ownerID),
		)
		return nil, ErrInvalidGroupName
	}
	slog.DebugContext(ctx, "db: creating reading group",
		slog.String(otelkeys.UserID, ownerID),
		slog.String(otelkeys.GroupName, name),
	)

	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer deferRollback(ctx, tx)

	g, err := scanReadingGroup(tx.QueryRowContext(ctx,
		`INSERT INTO reading_groups (owner_id, name, description) VALUES ($1, $2, $3)
		 RETURNING id, owner_id, name, description, 0, created_at, updated_at`,
		ownerID, name, description,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrGroupNameExists
		}
		return nil, err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO reading_group_members (group_id, user_id, role) VALUES ($1, $2, 'owner')`,
		g.ID, ownerID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	g.MemberCount = 1
	return g, nil
}

// GetGroup retrieves a reading group by ID. The requesting user must be a member.
func (d *DB) GetGroup(ctx context.Context, id, userID string) (*ReadingGroup, error) {
	slog.DebugContext(ctx, "db: fetching reading group",
		slog.String(otelkeys.GroupID, id),
		slog.String(otelkeys.UserID, userID),
	)
	return scanReadingGroup(d.QueryRowContext(ctx,
		`SELECT g.id, g.owner_id, g.name, g.description, COUNT(m.user_id), g.created_at, g.updated_at
		 FROM reading_groups g
		 LEFT JOIN reading_group_members m ON m.group_id = g.id
		 WHERE g.id = $1
		   AND EXISTS (SELECT 1 FROM reading_group_members WHERE group_id = $1 AND user_id = $2)
		 GROUP BY g.id, g.owner_id, g.name, g.description, g.created_at, g.updated_at`,
		id, userID,
	))
}

// ListGroups returns all reading groups where the user is a member.
func (d *DB) ListGroups(ctx context.Context, userID string) ([]ReadingGroup, error) {
	slog.DebugContext(ctx, "db: listing reading groups", slog.String(otelkeys.UserID, userID))
	rows, err := d.QueryContext(ctx,
		`SELECT g.id, g.owner_id, g.name, g.description, COUNT(m.user_id), g.created_at, g.updated_at
		 FROM reading_groups g
		 LEFT JOIN reading_group_members m ON m.group_id = g.id
		 WHERE g.id IN (SELECT group_id FROM reading_group_members WHERE user_id = $1)
		 GROUP BY g.id, g.owner_id, g.name, g.description, g.created_at, g.updated_at
		 ORDER BY g.name ASC, g.id ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanReadingGroup)
}

// UpdateGroup updates the name and description. Only the owner can update.
func (d *DB) UpdateGroup(ctx context.Context, id, ownerID, name string, description *string) (*ReadingGroup, error) {
	name = NormalizeGroupName(name)
	if name == "" {
		slog.WarnContext(ctx, "db: rejecting group update with blank name",
			slog.String(otelkeys.GroupID, id),
		)
		return nil, ErrInvalidGroupName
	}
	slog.DebugContext(ctx, "db: updating reading group",
		slog.String(otelkeys.GroupID, id),
		slog.String(otelkeys.GroupName, name),
	)
	g, err := scanReadingGroup(d.QueryRowContext(ctx,
		`UPDATE reading_groups SET name = $1, description = $2, updated_at = `+d.now()+`
		 WHERE id = $3 AND owner_id = $4
		 RETURNING id, owner_id, name, description,
		   (SELECT COUNT(*) FROM reading_group_members WHERE group_id = reading_groups.id),
		   created_at, updated_at`,
		name, description, id, ownerID,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrGroupNameExists
		}
		return nil, err
	}
	return g, nil
}

// DeleteGroup deletes a reading group. Only the owner can delete.
func (d *DB) DeleteGroup(ctx context.Context, id, ownerID string) error {
	slog.DebugContext(ctx, "db: deleting reading group",
		slog.String(otelkeys.GroupID, id),
		slog.String(otelkeys.UserID, ownerID),
	)
	return d.execAffected(ctx,
		`DELETE FROM reading_groups WHERE id = $1 AND owner_id = $2`,
		id, ownerID,
	)
}

// ListGroupMembers returns all members of a group. The requester must be a member.
func (d *DB) ListGroupMembers(ctx context.Context, groupID, requesterID string) ([]ReadingGroupMember, error) {
	slog.DebugContext(ctx, "db: listing group members",
		slog.String(otelkeys.GroupID, groupID),
		slog.String(otelkeys.UserID, requesterID),
	)
	isMember, err := d.IsMember(ctx, groupID, requesterID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, sql.ErrNoRows
	}
	rows, err := d.QueryContext(ctx,
		`SELECT m.group_id, m.user_id, u.name, u.email, m.role, m.joined_at
		 FROM reading_group_members m
		 JOIN users u ON u.id = m.user_id
		 WHERE m.group_id = $1
		 ORDER BY m.joined_at ASC`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanReadingGroupMember)
}

// AddGroupMember adds a user to a group. Only the owner can add members.
// Returns (true, nil) when the member was newly added and (false, nil) when
// the user is already a member (idempotent — ON CONFLICT DO NOTHING).
func (d *DB) AddGroupMember(ctx context.Context, groupID, ownerID, memberUserID string) (bool, error) {
	slog.DebugContext(ctx, "db: adding group member",
		slog.String(otelkeys.GroupID, groupID),
		slog.String(otelkeys.UserID, memberUserID),
	)
	var existingOwnerID string
	err := d.QueryRowContext(ctx, `SELECT owner_id FROM reading_groups WHERE id = $1`, groupID).Scan(&existingOwnerID)
	if err != nil {
		return false, err
	}
	if existingOwnerID != ownerID {
		return false, sql.ErrNoRows
	}

	res, err := d.ExecContext(ctx,
		`INSERT INTO reading_group_members (group_id, user_id, role) VALUES ($1, $2, 'member')
		 ON CONFLICT (group_id, user_id) DO NOTHING`,
		groupID, memberUserID,
	)
	if err != nil {
		if isForeignKeyViolation(err) {
			return false, ErrMemberUserNotFound
		}
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// RemoveGroupMember removes a member from a group. Owner can remove anyone; members can remove themselves.
func (d *DB) RemoveGroupMember(ctx context.Context, groupID, requesterID, targetUserID string) error {
	slog.DebugContext(ctx, "db: removing group member",
		slog.String(otelkeys.GroupID, groupID),
		slog.String(otelkeys.UserID, targetUserID),
	)
	if requesterID != targetUserID {
		var ownerID string
		err := d.QueryRowContext(ctx, `SELECT owner_id FROM reading_groups WHERE id = $1`, groupID).Scan(&ownerID)
		if err != nil {
			return err
		}
		if ownerID != requesterID {
			return sql.ErrNoRows
		}
	} else {
		// Prevent the owner from abandoning their own group.
		var ownerID string
		if err := d.QueryRowContext(ctx, `SELECT owner_id FROM reading_groups WHERE id = $1`, groupID).Scan(&ownerID); err != nil {
			return err
		}
		if ownerID == targetUserID {
			return ErrOwnerCannotLeaveGroup
		}
	}
	return d.execAffected(ctx,
		`DELETE FROM reading_group_members WHERE group_id = $1 AND user_id = $2`,
		groupID, targetUserID,
	)
}

// IsMember returns true if the user is a member of the group.
func (d *DB) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	var exists bool
	err := d.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM reading_group_members WHERE group_id = $1 AND user_id = $2)`,
		groupID, userID,
	).Scan(&exists)
	return exists, err
}

// GroupMemberProgress holds reading progress info for a group member on a specific book.
type GroupMemberProgress struct {
	UserID     string     `json:"user_id"`
	UserName   string     `json:"user_name"`
	Percentage float64    `json:"percentage"`
	UpdatedAt  *Timestamp `json:"updated_at"`
}

func scanGroupMemberProgress(row interface{ Scan(...any) error }) (*GroupMemberProgress, error) {
	return scanRow(row, func(p *GroupMemberProgress) []any {
		return []any{&p.UserID, &p.UserName, &p.Percentage, &p.UpdatedAt}
	})
}

// ListGroupMemberProgress returns reading progress for all group members on a specific book.
func (d *DB) ListGroupMemberProgress(ctx context.Context, groupID, bookID, requesterID string) ([]GroupMemberProgress, error) {
	slog.DebugContext(ctx, "db: listing group member progress",
		slog.String(otelkeys.GroupID, groupID),
		slog.String(otelkeys.BookID, bookID),
		slog.String(otelkeys.UserID, requesterID),
	)
	isMember, err := d.IsMember(ctx, groupID, requesterID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, sql.ErrNoRows
	}
	rows, err := d.QueryContext(ctx,
		`SELECT m.user_id, u.name, COALESCE(k.percent_read, 0), k.updated_at
		 FROM reading_group_members m
		 JOIN users u ON u.id = m.user_id
		 LEFT JOIN kobo_reading_states k ON k.user_id = m.user_id AND k.book_id = $2
		 WHERE m.group_id = $1
		 ORDER BY (k.updated_at IS NULL), k.updated_at DESC`,
		groupID, bookID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanGroupMemberProgress)
}
