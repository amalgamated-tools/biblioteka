package db

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// ShareListWithGroup shares a reading list with a group.
// The user must own the list and be a group member.
// Returns (true, nil) if newly shared; (false, nil) if already shared.
func (d *DB) ShareListWithGroup(ctx context.Context, groupID, listID, userID string) (bool, error) {
	slog.DebugContext(ctx, "db: sharing list with group",
		slog.String(otelkeys.GroupID, groupID),
		slog.String(otelkeys.ReadingListID, listID),
		slog.String(otelkeys.UserID, userID),
	)
	if err := d.verifyReadingListOwnership(ctx, listID, userID); err != nil {
		return false, err
	}
	isMember, err := d.IsMember(ctx, groupID, userID)
	if err != nil {
		return false, err
	}
	if !isMember {
		return false, sql.ErrNoRows
	}

	result, err := d.ExecContext(ctx,
		`INSERT INTO reading_group_lists (group_id, list_id, shared_by) VALUES ($1, $2, $3)
		 ON CONFLICT (group_id, list_id) DO NOTHING`,
		groupID, listID, userID,
	)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// UnshareListFromGroup removes a reading list from a group. The user must own the list.
func (d *DB) UnshareListFromGroup(ctx context.Context, groupID, listID, userID string) error {
	slog.DebugContext(ctx, "db: unsharing list from group",
		slog.String(otelkeys.GroupID, groupID),
		slog.String(otelkeys.ReadingListID, listID),
		slog.String(otelkeys.UserID, userID),
	)
	if err := d.verifyReadingListOwnership(ctx, listID, userID); err != nil {
		return err
	}
	return d.execAffected(ctx,
		`DELETE FROM reading_group_lists WHERE group_id = $1 AND list_id = $2`,
		groupID, listID,
	)
}

// ListGroupReadingLists returns all reading lists shared with a group.
// The requester must be a member.
func (d *DB) ListGroupReadingLists(ctx context.Context, groupID, requesterID string) ([]ReadingList, error) {
	slog.DebugContext(ctx, "db: listing group reading lists",
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
		`SELECT `+readingListColumns+` `+readingListBaseFrom+`
		 INNER JOIN reading_group_lists rgl ON rgl.list_id = rl.id
		 WHERE rgl.group_id = $1
		 GROUP BY rl.id, rl.user_id, rl.name, rl.description, rl.created_at, rl.updated_at
		 ORDER BY rl.name ASC, rl.id ASC`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanReadingList)
}
