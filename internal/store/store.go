package store

import (
	"database/sql"
)

// Store is the concrete implementation of the Store interface of the mysql package
type Store struct {
	db *sql.DB
}

// New creates a new Store instance
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetDB returns the underlying sql.DB instance
func (s *Store) GetDB() *sql.DB {
	return s.db
}
