// Package database provides download database operations.
// Ported from TypeScript: src/drivers/database/sqlite3/download/
package database

import (
	"fmt"
	"path/filepath"
)

// DatasetDB manages URL cache for downloads
type DatasetDB struct {
	db *DB
}

// OpenDatasetDB opens or creates the dataset.sqlite database
func OpenDatasetDB(dataDir string) (*DatasetDB, error) {
	path := filepath.Join(dataDir, "dataset.sqlite")
	db, err := Open(path, false)
	if err != nil {
		return nil, err
	}

	// Create table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS url_cache (
			url_key TEXT PRIMARY KEY,
			url TEXT NOT NULL,
			etag TEXT,
			last_modified TEXT,
			content_length INTEGER,
			crc32 INTEGER
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &DatasetDB{db: db}, nil
}

// Close closes the database
func (d *DatasetDB) Close() error {
	return d.db.Close()
}

// SaveURLCache stores URL cache metadata
func (d *DatasetDB) SaveURLCache(urlKey, url, etag, lastModified string, contentLength int64, crc32 uint32) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO url_cache (url_key, url, etag, last_modified, content_length, crc32)
		VALUES (?, ?, ?, ?, ?, ?)
	`, urlKey, url, etag, lastModified, contentLength, crc32)
	return err
}

// ReadURLCache retrieves URL cache metadata
func (d *DatasetDB) ReadURLCache(urlKey string) (*URLCacheEntry, error) {
	entry := &URLCacheEntry{}
	err := d.db.QueryRow(`
		SELECT url_key, url, etag, last_modified, content_length, crc32
		FROM url_cache
		WHERE url_key = ?
	`, urlKey).Scan(
		&entry.URLKey, &entry.URL, &entry.Etag, &entry.LastModified,
		&entry.ContentLength, &entry.CRC32,
	)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// DeleteURLCache removes a URL cache entry
func (d *DatasetDB) DeleteURLCache(urlKey string) error {
	_, err := d.db.Exec("DELETE FROM url_cache WHERE url_key = ?", urlKey)
	return err
}

// URLCacheEntry holds cached URL metadata
type URLCacheEntry struct {
	URLKey        string
	URL           string
	Etag          string
	LastModified  string
	ContentLength int64
	CRC32         uint32
}

// DownloadCommonDB manages the common database for downloads
type DownloadCommonDB struct {
	db *DB
}

// OpenDownloadCommonDB opens or creates the common.sqlite database for writing
func OpenDownloadCommonDB(dataDir string) (*DownloadCommonDB, error) {
	path := filepath.Join(dataDir, "common.sqlite")
	db, err := Open(path, false)
	if err != nil {
		return nil, err
	}

	d := &DownloadCommonDB{db: db}
	if err := d.createTables(); err != nil {
		db.Close()
		return nil, err
	}

	return d, nil
}

// Close closes the database
func (d *DownloadCommonDB) Close() error {
	return d.db.Close()
}

// createTables creates the necessary tables if they don't exist
func (d *DownloadCommonDB) createTables() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS pref (
			lg_code TEXT NOT NULL,
			pref_key TEXT NOT NULL,
			pref TEXT NOT NULL,
			kana_pref TEXT NOT NULL DEFAULT '',
			roma_pref TEXT NOT NULL DEFAULT '',
			rep_lat REAL NOT NULL DEFAULT 0,
			rep_lon REAL NOT NULL DEFAULT 0,
			PRIMARY KEY (pref_key)
		)`,
		`CREATE TABLE IF NOT EXISTS city (
			lg_code TEXT NOT NULL,
			pref_key TEXT NOT NULL,
			city_key TEXT NOT NULL,
			pref TEXT NOT NULL DEFAULT '',
			county TEXT NOT NULL DEFAULT '',
			city TEXT NOT NULL DEFAULT '',
			ward TEXT NOT NULL DEFAULT '',
			kana_city TEXT NOT NULL DEFAULT '',
			roma_city TEXT NOT NULL DEFAULT '',
			rep_lat REAL NOT NULL DEFAULT 0,
			rep_lon REAL NOT NULL DEFAULT 0,
			PRIMARY KEY (city_key)
		)`,
		`CREATE TABLE IF NOT EXISTS town (
			lg_code TEXT NOT NULL,
			pref_key TEXT NOT NULL,
			city_key TEXT NOT NULL,
			town_key TEXT NOT NULL,
			machiaza_id TEXT NOT NULL DEFAULT '',
			pref TEXT NOT NULL DEFAULT '',
			county TEXT NOT NULL DEFAULT '',
			city TEXT NOT NULL DEFAULT '',
			ward TEXT NOT NULL DEFAULT '',
			oaza_cho TEXT NOT NULL DEFAULT '',
			chome TEXT NOT NULL DEFAULT '',
			koaza TEXT NOT NULL DEFAULT '',
			koaza_aka_code INTEGER NOT NULL DEFAULT 0,
			rsdt_addr_flg INTEGER NOT NULL DEFAULT 0,
			rep_lat REAL NOT NULL DEFAULT 0,
			rep_lon REAL NOT NULL DEFAULT 0,
			PRIMARY KEY (town_key)
		)`,
	}

	for _, stmt := range statements {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// UpsertPref inserts or updates a prefecture record
func (d *DownloadCommonDB) UpsertPref(lgCode, prefKey, pref, kanaPref, romaPref string, repLat, repLon float64) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO pref (lg_code, pref_key, pref, kana_pref, roma_pref, rep_lat, rep_lon)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, lgCode, prefKey, pref, kanaPref, romaPref, repLat, repLon)
	return err
}

// UpsertCity inserts or updates a city record
func (d *DownloadCommonDB) UpsertCity(lgCode, prefKey, cityKey, pref, county, city, ward, kanaCity, romaCity string, repLat, repLon float64) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO city (lg_code, pref_key, city_key, pref, county, city, ward, kana_city, roma_city, rep_lat, rep_lon)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, lgCode, prefKey, cityKey, pref, county, city, ward, kanaCity, romaCity, repLat, repLon)
	return err
}

// GetPrefKeyMap returns a map of pref_name → pref_key from the pref table
func (d *DownloadCommonDB) GetPrefKeyMap() (map[string]string, error) {
	rows, err := d.db.Query("SELECT pref, pref_key FROM pref")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var pref, prefKey string
		if err := rows.Scan(&pref, &prefKey); err != nil {
			return nil, err
		}
		m[pref] = prefKey
	}
	return m, rows.Err()
}

// UpsertTown inserts or updates a town record
func (d *DownloadCommonDB) UpsertTown(lgCode, prefKey, cityKey, townKey, machiazaID, pref, county, city, ward, oazaCho, chome, koaza string, koazaAkaCode, rsdtAddrFlg int, repLat, repLon float64) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO town (
			lg_code, pref_key, city_key, town_key, machiaza_id,
			pref, county, city, ward, oaza_cho, chome, koaza,
			koaza_aka_code, rsdt_addr_flg, rep_lat, rep_lon
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, lgCode, prefKey, cityKey, townKey, machiazaID,
		pref, county, city, ward, oazaCho, chome, koaza,
		koazaAkaCode, rsdtAddrFlg, repLat, repLon)
	return err
}

// DownloadLGDB manages a local government specific database for downloads
type DownloadLGDB struct {
	db     *DB
	lgCode string
}

// OpenDownloadLGDB opens or creates an LG-specific database
func OpenDownloadLGDB(dataDir, lgCode string) (*DownloadLGDB, error) {
	path := filepath.Join(dataDir, fmt.Sprintf("abrg-%s.sqlite", lgCode))
	db, err := Open(path, false)
	if err != nil {
		return nil, err
	}

	d := &DownloadLGDB{db: db, lgCode: lgCode}
	if err := d.createTables(); err != nil {
		db.Close()
		return nil, err
	}

	return d, nil
}

// Close closes the database
func (d *DownloadLGDB) Close() error {
	return d.db.Close()
}

// createTables creates the tables in the LG database
func (d *DownloadLGDB) createTables() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS rsdt_blk (
			lg_code TEXT NOT NULL,
			town_key TEXT NOT NULL,
			blk_key TEXT NOT NULL,
			blk_num TEXT NOT NULL DEFAULT '',
			rep_lat REAL NOT NULL DEFAULT 0,
			rep_lon REAL NOT NULL DEFAULT 0,
			PRIMARY KEY (blk_key)
		)`,
		`CREATE TABLE IF NOT EXISTS rsdt_dsp (
			lg_code TEXT NOT NULL,
			town_key TEXT NOT NULL,
			blk_key TEXT NOT NULL,
			rsdt_key TEXT NOT NULL,
			rsdt_id TEXT,
			rsdt_id2 TEXT,
			rep_lat REAL NOT NULL DEFAULT 0,
			rep_lon REAL NOT NULL DEFAULT 0,
			PRIMARY KEY (rsdt_key)
		)`,
		`CREATE TABLE IF NOT EXISTS parcel (
			lg_code TEXT NOT NULL,
			town_key TEXT NOT NULL,
			parcel_key TEXT NOT NULL,
			prc_num1 TEXT NOT NULL DEFAULT '',
			prc_num2 TEXT NOT NULL DEFAULT '',
			prc_num3 TEXT NOT NULL DEFAULT '',
			rep_lat REAL NOT NULL DEFAULT 0,
			rep_lon REAL NOT NULL DEFAULT 0,
			PRIMARY KEY (parcel_key)
		)`,
	}

	for _, stmt := range statements {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// UpsertRsdtBlk inserts or updates a residential block record
func (d *DownloadLGDB) UpsertRsdtBlk(lgCode, townKey, blkKey, blkNum string, repLat, repLon float64) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO rsdt_blk (lg_code, town_key, blk_key, blk_num, rep_lat, rep_lon)
		VALUES (?, ?, ?, ?, ?, ?)
	`, lgCode, townKey, blkKey, blkNum, repLat, repLon)
	return err
}

// UpsertRsdtDsp inserts or updates a residential display record
func (d *DownloadLGDB) UpsertRsdtDsp(lgCode, townKey, blkKey, rsdtKey string, rsdtID, rsdtID2 *string, repLat, repLon float64) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO rsdt_dsp (lg_code, town_key, blk_key, rsdt_key, rsdt_id, rsdt_id2, rep_lat, rep_lon)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, lgCode, townKey, blkKey, rsdtKey, rsdtID, rsdtID2, repLat, repLon)
	return err
}

// UpsertParcel inserts or updates a parcel record
func (d *DownloadLGDB) UpsertParcel(lgCode, townKey, parcelKey, prcNum1, prcNum2, prcNum3 string, repLat, repLon float64) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO parcel (lg_code, town_key, parcel_key, prc_num1, prc_num2, prc_num3, rep_lat, rep_lon)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, lgCode, townKey, parcelKey, prcNum1, prcNum2, prcNum3, repLat, repLon)
	return err
}
