package entrysvc

import (
	"context"
	"fmt"
)

var (
	ErrNoValue = fmt.Errorf("no value")
)

type Entry struct {
	ID    int64  `db:"id"`
	Value string `db:"value"`
}

type EntryService struct {
	repo *EntryRepository
}

func NewEntryService(repo *EntryRepository) *EntryService {
	return &EntryService{repo: repo}
}

func (e *EntryService) Ready(ctx context.Context) error {
	return e.repo.Ping(ctx)
}

func (e *EntryService) Add(ctx context.Context, entry *Entry) error {
	return e.repo.Add(ctx, entry)
}

func (e *EntryService) List(ctx context.Context) ([]*Entry, error) {
	return e.repo.List(ctx)
}
