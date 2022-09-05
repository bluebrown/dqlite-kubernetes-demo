package model

import (
	"context"
	"database/sql"
	"fmt"
)

func Migrate(ctx context.Context, db *sql.DB) error {
	ddl := `CREATE TABLE IF NOT EXISTS authors (
		id   INTEGER PRIMARY KEY AUTOINCREMENT,
		name text    NOT NULL,
		bio  text
	);`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("could not create schema: %w", err)
	}
	return nil
}
