package db

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// searchLikeReplacer escapes special LIKE pattern characters for use in SQL LIKE/ILIKE queries.
var searchLikeReplacer = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (d *DB) execBooksPaginated(ctx context.Context, offset int, mainQuery, countQuery string, mainArgs, countArgs []any) ([]Book, int, error) {
	rows, err := d.QueryContext(ctx, mainQuery, mainArgs...)
	if err != nil {
		return nil, 0, err
	}
	books, total, err := collectRowsAndTotal(rows, scanBookAndTotal)
	if err != nil {
		return nil, 0, err
	}
	if err := countFallback(ctx, d, &total, len(books), offset, countQuery, countArgs...); err != nil {
		return nil, 0, err
	}
	return books, total, nil
}

// ListBooksPaginated returns books ordered by title with pagination and total count.
func (d *DB) ListBooksPaginated(ctx context.Context, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing books paginated",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	orderBy := d.dialectOrderBy("title", "ASC")
	return d.execBooksPaginated(
		ctx,
		offset,
		`SELECT `+bookColumns+`, COUNT(*) OVER() FROM books `+orderBy+` LIMIT $1 OFFSET $2`,
		`SELECT COUNT(*) FROM books`,
		[]any{limit, offset},
		nil,
	)
}

// ListRecentBooks returns books ordered by creation time (newest first) with pagination and total count.
func (d *DB) ListRecentBooks(ctx context.Context, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing recent books",
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	return d.execBooksPaginated(
		ctx,
		offset,
		`SELECT `+bookColumns+`, COUNT(*) OVER() FROM books ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`,
		`SELECT COUNT(*) FROM books`,
		[]any{limit, offset},
		nil,
	)
}

// ListBooksByAuthor returns all books for a specific author.
func (d *DB) ListBooksByAuthor(ctx context.Context, authorID string) ([]Book, error) {
	slog.DebugContext(ctx, "db: listing books by author", slog.String(otelkeys.AuthorID, authorID))
	orderBy := d.dialectOrderBy("b.title", "ASC")
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumnsWithPrefix("b.")+` FROM books b INNER JOIN book_authors ba ON ba.book_id = b.id WHERE ba.author_id = $1 `+orderBy,
		authorID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanBook)
}

// ListBooksByAuthorPaginated returns books for a specific author with pagination and total count.
func (d *DB) ListBooksByAuthorPaginated(ctx context.Context, authorID string, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing books by author paginated",
		slog.String(otelkeys.AuthorID, authorID),
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	orderBy := d.dialectOrderBy("b.title", "ASC")
	return d.execBooksPaginated(
		ctx,
		offset,
		`SELECT `+bookColumnsWithPrefix("b.")+`, COUNT(*) OVER() FROM books b INNER JOIN book_authors ba ON ba.book_id = b.id WHERE ba.author_id = $1 `+orderBy+` LIMIT $2 OFFSET $3`,
		`SELECT COUNT(*) FROM books b INNER JOIN book_authors ba ON ba.book_id = b.id WHERE ba.author_id = $1`,
		[]any{authorID, limit, offset},
		[]any{authorID},
	)
}

// seriesPositionOrderBy returns a dialect-appropriate ORDER BY clause for series books,
// using NULLS LAST on PostgreSQL so unpositioned books sort after positioned ones.
func (d *DB) seriesPositionOrderBy() string {
	if d.Dialect == DialectPostgres {
		return "ORDER BY bs.position ASC NULLS LAST, b.title ASC"
	}
	return "ORDER BY bs.position ASC, b.title ASC"
}

// ListBooksBySeries returns all books in a specific series, ordered by position.
func (d *DB) ListBooksBySeries(ctx context.Context, seriesID string) ([]Book, error) {
	slog.DebugContext(ctx, "db: listing books by series", slog.String(otelkeys.SeriesID, seriesID))
	rows, err := d.QueryContext(ctx,
		`SELECT `+bookColumnsWithPrefix("b.")+` FROM books b INNER JOIN book_series bs ON bs.book_id = b.id WHERE bs.series_id = $1 `+d.seriesPositionOrderBy(),
		seriesID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanBook)
}

// ListBooksBySeriesPaginated returns books in a specific series with pagination and total count.
func (d *DB) ListBooksBySeriesPaginated(ctx context.Context, seriesID string, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: listing books by series paginated",
		slog.String(otelkeys.SeriesID, seriesID),
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	return d.execBooksPaginated(
		ctx,
		offset,
		`SELECT `+bookColumnsWithPrefix("b.")+`, COUNT(*) OVER() FROM books b INNER JOIN book_series bs ON bs.book_id = b.id WHERE bs.series_id = $1 `+d.seriesPositionOrderBy()+` LIMIT $2 OFFSET $3`,
		`SELECT COUNT(*) FROM books b INNER JOIN book_series bs ON bs.book_id = b.id WHERE bs.series_id = $1`,
		[]any{seriesID, limit, offset},
		[]any{seriesID},
	)
}

// buildILIKESearchWhere builds a WHERE clause and args for a per-token ILIKE
// search on the books table. Each whitespace-delimited token in query becomes
// an independent AND condition that checks both title and description, giving
// the same "all tokens must appear somewhere" semantics as the FTS5 path used
// on SQLite.
//
// Positional placeholders start at startIdx. Returns ("", nil) when query
// contains no tokens.
func buildILIKESearchWhere(query string, startIdx int) (string, []any) {
	if startIdx < 1 {
		startIdx = 1
	}
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return "", nil
	}
	conditions := make([]string, 0, len(tokens))
	args := make([]any, 0, len(tokens))
	idx := startIdx
	for _, token := range tokens {
		escaped := searchLikeReplacer.Replace(token)
		conditions = append(conditions, fmt.Sprintf(
			`(title ILIKE $%d ESCAPE '\' OR description ILIKE $%d ESCAPE '\')`,
			idx, idx,
		))
		args = append(args, "%"+escaped+"%")
		idx++
	}
	if len(conditions) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// SearchBooks searches books by title or description with pagination and total count.
//
// On SQLite the search is backed by a FTS5 virtual table (books_fts) for
// index-accelerated full-text matching. Multi-token queries are evaluated as an
// implicit AND across the combined FTS document (title + description), so
// different tokens may match in different columns (e.g., one in title and one
// in description).
//
// On PostgreSQL each whitespace-delimited token produces its own ILIKE
// condition (accelerated by the pg_trgm GIN indexes added in migration
// 20260412000000_add_books_trgm), and all token conditions are joined with AND.
// This mirrors FTS5 token-AND semantics: every token must appear somewhere in
// either title or description.
//
// An empty or whitespace-only query returns zero results on all dialects.
func (d *DB) SearchBooks(ctx context.Context, query string, limit, offset int) ([]Book, int, error) {
	slog.DebugContext(ctx, "db: searching books",
		slog.String(otelkeys.Query, query),
		slog.Int(otelkeys.Limit, limit),
		slog.Int(otelkeys.Offset, offset),
	)

	query = strings.TrimSpace(query)
	if query == "" {
		return []Book{}, 0, nil
	}

	var (
		whereClause string
		searchArgs  []any
	)

	if d.Dialect == DialectSQLite {
		ftsQuery := sanitizeFTS5Query(query)
		if ftsQuery == "" {
			return []Book{}, 0, nil
		}
		whereClause = `WHERE rowid IN (SELECT rowid FROM books_fts WHERE books_fts MATCH $1)`
		searchArgs = []any{ftsQuery}
	} else {
		whereClause, searchArgs = buildILIKESearchWhere(query, 1)
		if whereClause == "" {
			return []Book{}, 0, nil
		}
	}

	limitPos := dollarN(len(searchArgs) + 1)
	offsetPos := dollarN(len(searchArgs) + 2)
	mainArgs := make([]any, 0, len(searchArgs)+2)
	mainArgs = append(mainArgs, searchArgs...)
	mainArgs = append(mainArgs, limit, offset)

	orderBy := d.dialectOrderBy("title", "ASC")
	return d.execBooksPaginated(
		ctx,
		offset,
		`SELECT `+bookColumns+`, COUNT(*) OVER() FROM books `+whereClause+` `+orderBy+` LIMIT `+limitPos+` OFFSET `+offsetPos,
		`SELECT COUNT(*) FROM books `+whereClause,
		mainArgs,
		searchArgs,
	)
}
