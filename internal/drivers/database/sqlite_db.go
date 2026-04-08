// Package database provides SQLite3 database access for abr-geocoder.
// Ported from TypeScript: src/drivers/database/
package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // SQLite driver
)

// DB wraps a SQLite database connection
type DB struct {
	db       *sql.DB
	path     string
	readOnly bool
}

// Open opens a SQLite database at the given path
func Open(path string, readOnly bool) (*DB, error) {
	var dsn string
	if readOnly {
		dsn = fmt.Sprintf("file:%s?mode=ro&_journal_mode=OFF&_synchronous=OFF&cache=shared", path)
	} else {
		dsn = fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL&cache=shared", path)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", path, err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1)

	if readOnly {
		// Read-only optimizations
		if _, err := db.Exec("PRAGMA cache_size=20000"); err != nil {
			db.Close()
			return nil, err
		}
	}

	return &DB{db: db, path: path, readOnly: readOnly}, nil
}

// OpenIfExists opens a database only if the file exists, returns nil if not found
func OpenIfExists(path string, readOnly bool) (*DB, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	return Open(path, readOnly)
}

// Close closes the database connection
func (d *DB) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// HasTable returns true if the named table exists
func (d *DB) HasTable(tableName string) (bool, error) {
	var count int
	err := d.db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",
		tableName,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Exec executes a SQL statement
func (d *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return d.db.Exec(query, args...)
}

// QueryRow executes a query that returns a single row
func (d *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.db.QueryRow(query, args...)
}

// Query executes a query that returns multiple rows
func (d *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.Query(query, args...)
}

// Begin starts a transaction
func (d *DB) Begin() (*sql.Tx, error) {
	return d.db.Begin()
}

// CommonDB provides access to the common SQLite database
type CommonDB struct {
	db *DB
}

// OpenCommonDB opens the common.sqlite database
func OpenCommonDB(dataDir string, readOnly bool) (*CommonDB, error) {
	path := filepath.Join(dataDir, "common.sqlite")
	db, err := Open(path, readOnly)
	if err != nil {
		return nil, err
	}

	// Verify the database has the expected tables
	for _, table := range []string{"pref", "city", "town"} {
		has, err := db.HasTable(table)
		if err != nil {
			db.Close()
			return nil, err
		}
		if !has {
			db.Close()
			return nil, fmt.Errorf("database %s is missing table %s", path, table)
		}
	}

	return &CommonDB{db: db}, nil
}

// OpenCommonDBIfExists opens the common.sqlite database if it exists
func OpenCommonDBIfExists(dataDir string, readOnly bool) (*CommonDB, error) {
	path := filepath.Join(dataDir, "common.sqlite")
	db, err := OpenIfExists(path, readOnly)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}
	return &CommonDB{db: db}, nil
}

// Close closes the common database
func (c *CommonDB) Close() error {
	return c.db.Close()
}
