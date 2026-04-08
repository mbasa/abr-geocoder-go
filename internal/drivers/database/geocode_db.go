// Package database provides geocoding database operations.
// Ported from TypeScript: src/drivers/database/sqlite3/geocode/common-db-geocode-sqlite3.ts
package database

import (
	"fmt"
	"path/filepath"

	"github.com/mbasa/abr-geocoder-go/internal/domain/models"
)

// GeocodeDB provides geocoding queries against the common SQLite database
type GeocodeDB struct {
	db *DB
}

// OpenGeocodeDB opens the common database for geocoding queries
func OpenGeocodeDB(dataDir string) (*GeocodeDB, error) {
	path := filepath.Join(dataDir, "common.sqlite")
	db, err := Open(path, true)
	if err != nil {
		return nil, err
	}

	for _, table := range []string{"pref", "city", "town"} {
		has, err := db.HasTable(table)
		if err != nil {
			db.Close()
			return nil, err
		}
		if !has {
			db.Close()
			return nil, fmt.Errorf("database %s is missing required table: %s", path, table)
		}
	}

	return &GeocodeDB{db: db}, nil
}

// Close closes the database
func (g *GeocodeDB) Close() error {
	return g.db.Close()
}

// GetPrefList retrieves all prefectures
func (g *GeocodeDB) GetPrefList() ([]*models.PrefRow, error) {
	rows, err := g.db.Query(`
		SELECT lg_code, pref_key, pref, kana_pref, roma_pref, rep_lat, rep_lon
		FROM pref
		ORDER BY lg_code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.PrefRow
	for rows.Next() {
		row := &models.PrefRow{}
		if err := rows.Scan(
			&row.LGCode, &row.PrefKey, &row.Pref, &row.KanaPref, &row.RomaPref,
			&row.RepLat, &row.RepLon,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// GetCityList retrieves all cities for a given prefecture
func (g *GeocodeDB) GetCityList(prefKey string) ([]*models.CityRow, error) {
	rows, err := g.db.Query(`
		SELECT lg_code, pref_key, city_key, pref, county, city, ward,
		       kana_city, roma_city, rep_lat, rep_lon
		FROM city
		WHERE pref_key = ?
		ORDER BY lg_code
	`, prefKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.CityRow
	for rows.Next() {
		row := &models.CityRow{}
		if err := rows.Scan(
			&row.LGCode, &row.PrefKey, &row.CityKey, &row.Pref, &row.County,
			&row.City, &row.Ward, &row.KanaCity, &row.RomaCity,
			&row.RepLat, &row.RepLon,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// GetAllCities retrieves all cities
func (g *GeocodeDB) GetAllCities() ([]*models.CityRow, error) {
	rows, err := g.db.Query(`
		SELECT lg_code, pref_key, city_key, pref, county, city, ward,
		       kana_city, roma_city, rep_lat, rep_lon
		FROM city
		ORDER BY lg_code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.CityRow
	for rows.Next() {
		row := &models.CityRow{}
		if err := rows.Scan(
			&row.LGCode, &row.PrefKey, &row.CityKey, &row.Pref, &row.County,
			&row.City, &row.Ward, &row.KanaCity, &row.RomaCity,
			&row.RepLat, &row.RepLon,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// GetTownList retrieves all towns for a given city
func (g *GeocodeDB) GetTownList(cityKey string) ([]*models.TownRow, error) {
	rows, err := g.db.Query(`
		SELECT lg_code, pref_key, city_key, town_key, machiaza_id,
		       pref, county, city, ward, oaza_cho, chome, koaza,
		       koaza_aka_code, rsdt_addr_flg, rep_lat, rep_lon
		FROM town
		WHERE city_key = ?
		ORDER BY town_key
	`, cityKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTownRows(rows)
}

// GetOazaChomes retrieves towns for matching oaza/chome portions
func (g *GeocodeDB) GetOazaChomes(cityKey string) ([]*models.TownRow, error) {
	rows, err := g.db.Query(`
		SELECT lg_code, pref_key, city_key, town_key, machiaza_id,
		       pref, county, city, ward, oaza_cho, chome, koaza,
		       koaza_aka_code, rsdt_addr_flg, rep_lat, rep_lon
		FROM town
		WHERE city_key = ?
		  AND (oaza_cho != '' OR chome != '' OR koaza != '')
		ORDER BY town_key
	`, cityKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTownRows(rows)
}

func scanTownRows(rows interface{ Next() bool; Scan(...interface{}) error; Err() error }) ([]*models.TownRow, error) {
	var result []*models.TownRow
	for rows.Next() {
		row := &models.TownRow{}
		if err := rows.Scan(
			&row.LGCode, &row.PrefKey, &row.CityKey, &row.TownKey, &row.MachiazaID,
			&row.Pref, &row.County, &row.City, &row.Ward, &row.OazaCho, &row.Chome, &row.Koaza,
			&row.KoazaAkaCode, &row.RsdtAddrFlg, &row.RepLat, &row.RepLon,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// RsdtBlkDB provides residential block geocoding queries
type RsdtBlkDB struct {
	db *DB
}

// OpenRsdtBlkDB opens a residential block database for a specific LG code
func OpenRsdtBlkDB(dataDir, lgCode string) (*RsdtBlkDB, error) {
	path := filepath.Join(dataDir, fmt.Sprintf("abrg-%s.sqlite", lgCode))
	db, err := OpenIfExists(path, true)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}

	has, err := db.HasTable("rsdt_blk")
	if err != nil {
		db.Close()
		return nil, err
	}
	if !has {
		db.Close()
		return nil, nil
	}

	return &RsdtBlkDB{db: db}, nil
}

// Close closes the database
func (r *RsdtBlkDB) Close() error {
	return r.db.Close()
}

// GetBlockNumRows retrieves residential block numbers matching the given pattern
func (r *RsdtBlkDB) GetBlockNumRows(townKey, blkNumPattern string) ([]*models.RsdtBlkRow, error) {
	rows, err := r.db.Query(`
		SELECT lg_code, town_key, blk_key, blk_num, rep_lat, rep_lon
		FROM rsdt_blk
		WHERE town_key = ? AND blk_num LIKE ?
	`, townKey, blkNumPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.RsdtBlkRow
	for rows.Next() {
		row := &models.RsdtBlkRow{}
		if err := rows.Scan(
			&row.LGCode, &row.TownKey, &row.BlkKey, &row.BlkNum,
			&row.RepLat, &row.RepLon,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// RsdtDspDB provides residential display geocoding queries
type RsdtDspDB struct {
	db *DB
}

// OpenRsdtDspDB opens a residential display database for a specific LG code
func OpenRsdtDspDB(dataDir, lgCode string) (*RsdtDspDB, error) {
	path := filepath.Join(dataDir, fmt.Sprintf("abrg-%s.sqlite", lgCode))
	db, err := OpenIfExists(path, true)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}

	has, err := db.HasTable("rsdt_dsp")
	if err != nil {
		db.Close()
		return nil, err
	}
	if !has {
		db.Close()
		return nil, nil
	}

	return &RsdtDspDB{db: db}, nil
}

// Close closes the database
func (r *RsdtDspDB) Close() error {
	return r.db.Close()
}

// GetRsdtDspRows retrieves residential display records
func (r *RsdtDspDB) GetRsdtDspRows(townKey string, blkKey *string) ([]*models.RsdtDspRow, error) {
	var rows interface{ Next() bool; Scan(...interface{}) error; Err() error; Close() error }
	var err error

	if blkKey != nil {
		rows, err = r.db.Query(`
			SELECT lg_code, town_key, blk_key, rsdt_key,
			       IIF(rsdt_id IS NULL, '', rsdt_id) as rsdt_id,
			       IIF(rsdt_id2 IS NULL, '', rsdt_id2) as rsdt_id2,
			       rep_lat, rep_lon
			FROM rsdt_dsp
			WHERE town_key = ? AND blk_key = ?
		`, townKey, *blkKey)
	} else {
		rows, err = r.db.Query(`
			SELECT lg_code, town_key, blk_key, rsdt_key,
			       IIF(rsdt_id IS NULL, '', rsdt_id) as rsdt_id,
			       IIF(rsdt_id2 IS NULL, '', rsdt_id2) as rsdt_id2,
			       rep_lat, rep_lon
			FROM rsdt_dsp
			WHERE town_key = ?
		`, townKey)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.RsdtDspRow
	for rows.Next() {
		row := &models.RsdtDspRow{}
		if err := rows.Scan(
			&row.LGCode, &row.TownKey, &row.BlkKey, &row.RsdtKey,
			&row.RsdtID, &row.RsdtID2, &row.RepLat, &row.RepLon,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// ParcelDB provides parcel geocoding queries
type ParcelDB struct {
	db *DB
}

// OpenParcelDB opens a parcel database for a specific LG code
func OpenParcelDB(dataDir, lgCode string) (*ParcelDB, error) {
	path := filepath.Join(dataDir, fmt.Sprintf("abrg-%s.sqlite", lgCode))
	db, err := OpenIfExists(path, true)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}

	has, err := db.HasTable("parcel")
	if err != nil {
		db.Close()
		return nil, err
	}
	if !has {
		db.Close()
		return nil, nil
	}

	return &ParcelDB{db: db}, nil
}

// Close closes the database
func (p *ParcelDB) Close() error {
	return p.db.Close()
}

// GetParcelRows retrieves parcel records matching the given identifiers
func (p *ParcelDB) GetParcelRows(townKey string, prcNum1, prcNum2, prcNum3 *string) ([]*models.ParcelRow, error) {
	query := `
		SELECT lg_code, town_key, parcel_key, prc_num1, prc_num2, prc_num3, rep_lat, rep_lon
		FROM parcel
		WHERE town_key = ?
	`
	args := []interface{}{townKey}

	if prcNum1 != nil {
		query += " AND prc_num1 = ?"
		args = append(args, *prcNum1)
	}
	if prcNum2 != nil {
		query += " AND prc_num2 = ?"
		args = append(args, *prcNum2)
	}
	if prcNum3 != nil {
		query += " AND prc_num3 = ?"
		args = append(args, *prcNum3)
	}

	rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.ParcelRow
	for rows.Next() {
		row := &models.ParcelRow{}
		if err := rows.Scan(
			&row.LGCode, &row.TownKey, &row.ParcelKey,
			&row.PrcNum1, &row.PrcNum2, &row.PrcNum3,
			&row.RepLat, &row.RepLon,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
