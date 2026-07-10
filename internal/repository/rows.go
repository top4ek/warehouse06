package repository

import (
	"context"
	"database/sql"
)

// forEachRow runs query, invokes scan for every row, and closes the rows
// before returning. Closing before issuing the next query matters: in-memory
// mode runs on a single pooled connection, which stays busy until the rows
// are closed.
func (r *SQLiteRepository) forEachRow(ctx context.Context, query string, args []any, scan func(rows *sql.Rows) error) error {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return rows.Close()
}
