package entrysvc

import (
	"context"
	"database/sql"
	"fmt"
)

type EntryRepository struct {
	db *sql.DB
}

func NewEntryRepository(db *sql.DB) *EntryRepository {
	return &EntryRepository{db: db}
}

func (r *EntryRepository) Migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS entry (
	id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	value TEXT NOT NULL
);`)
	return err
}

func (r *EntryRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *EntryRepository) Add(ctx context.Context, entry *Entry) error {
	if entry.Value == "" {
		return ErrNoValue
	}
	result, err := r.db.ExecContext(ctx, "INSERT INTO entry (value) VALUES (?)", entry.Value)
	if err != nil {
		return fmt.Errorf("error inserting entry: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error getting last insert id: %w", err)
	}
	entry.ID = id
	return nil
}

func (r *EntryRepository) List(ctx context.Context) ([]*Entry, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, value FROM entry`)
	if err != nil {
		return nil, fmt.Errorf("error getting entries: %w", err)
	}
	defer rows.Close()
	var entries = make([]*Entry, 0, 10)
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Value); err != nil {
			return nil, fmt.Errorf("error scanning entry: %w", err)
		}
		entries = append(entries, &entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return entries, err
}
