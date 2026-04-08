// Package download provides functionality to download ABR datasets.
// Ported from TypeScript: src/usecases/download/download-process.ts
//
// Actual data API:
//   https://data.address-br.digital.go.jp/mt_{type}/{scope}/mt_{type}_{scope}{lgcode}.csv.zip
package download

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/mbasa/abr-geocoder-go/internal/drivers/database"
)

const dataBaseURL = "https://data.address-br.digital.go.jp"

// ProgressCallback is called with download progress updates
type ProgressCallback func(downloaded, total int64, current string)

// DownloadOptions configures the download process
type DownloadOptions struct {
	DataDir          string
	LGCodes          []string // Filter to specific LG codes
	Threads          int
	ProgressCallback ProgressCallback
	KeepData         bool
}

// Downloader manages the dataset download process
type Downloader struct {
	opts      DownloadOptions
	client    *http.Client
	datasetDB *database.DatasetDB
}

// NewDownloader creates a new Downloader
func NewDownloader(opts DownloadOptions) (*Downloader, error) {
	if opts.Threads <= 0 {
		opts.Threads = 5
	}
	if err := os.MkdirAll(opts.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	datasetDB, err := database.OpenDatasetDB(opts.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open dataset database: %w", err)
	}
	return &Downloader{
		opts:      opts,
		client:    &http.Client{},
		datasetDB: datasetDB,
	}, nil
}

// Close releases resources
func (d *Downloader) Close() error {
	return d.datasetDB.Close()
}

// Download downloads data for all specified LG codes
func (d *Downloader) Download() error {
	lgCodes := filterLGCodes(d.opts.LGCodes)
	if len(lgCodes) == 0 {
		return fmt.Errorf("no valid LG codes provided")
	}

	// Step 1: Download national-level pref data (once)
	fmt.Fprintln(os.Stderr, "Downloading prefecture data...")
	if err := d.downloadAndImportPref(); err != nil {
		return fmt.Errorf("pref download failed: %w", err)
	}

	// Step 2: Download per-LG-code data concurrently
	sem := make(chan struct{}, d.opts.Threads)
	var wg sync.WaitGroup
	errs := make(chan error, len(lgCodes))

	total := int64(len(lgCodes))
	var done int64

	for _, lgCode := range lgCodes {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := d.downloadForLGCode(code); err != nil {
				errs <- fmt.Errorf("LG %s: %w", code, err)
				return
			}

			done++
			if d.opts.ProgressCallback != nil {
				d.opts.ProgressCallback(done, total, code)
			}
		}(lgCode)
	}

	wg.Wait()
	close(errs)

	var errMsgs []string
	for err := range errs {
		errMsgs = append(errMsgs, err.Error())
	}
	if len(errMsgs) > 0 {
		return fmt.Errorf("download errors:\n%s", strings.Join(errMsgs, "\n"))
	}
	return nil
}

// downloadForLGCode downloads all data types for one LG code
func (d *Downloader) downloadForLGCode(lgCode string) error {
	prefCode := lgCode[:2] // e.g. "13" for Tokyo

	// city data for this pref (downloaded per-pref, keyed by pref code)
	if err := d.downloadAndImportCity(prefCode); err != nil {
		return err
	}

	// town data for this city
	if err := d.downloadAndImportTown(lgCode); err != nil {
		return err
	}

	// residential block data
	if err := d.downloadAndImportRsdtBlk(lgCode); err != nil {
		// Non-fatal: some cities don't have residential blocks
		fmt.Fprintf(os.Stderr, "  [skip] rsdt_blk for %s: %v\n", lgCode, err)
	}

	// residential display (rsdt) data
	if err := d.downloadAndImportRsdtDsp(lgCode); err != nil {
		fmt.Fprintf(os.Stderr, "  [skip] rsdt_rsdt for %s: %v\n", lgCode, err)
	}

	// parcel data
	if err := d.downloadAndImportParcel(lgCode); err != nil {
		fmt.Fprintf(os.Stderr, "  [skip] parcel for %s: %v\n", lgCode, err)
	}

	return nil
}

// --- Pref ---

func (d *Downloader) downloadAndImportPref() error {
	mainURL := dataBaseURL + "/mt_pref/mt_pref_all.csv.zip"
	posURL := dataBaseURL + "/mt_pref_pos/mt_pref_pos_all.csv.zip"

	mainData, err := d.downloadCSV(mainURL)
	if err != nil {
		return err
	}
	posData, err := d.downloadCSV(posURL)
	if err != nil {
		return err
	}

	// Build position map: lg_code → {lon, lat}
	posMap := buildPosMap2(posData) // col 0=lg_code, 1=rep_lon, 2=rep_lat

	db, err := database.OpenDownloadCommonDB(d.opts.DataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	reader := csv.NewReader(bytes.NewReader(mainData))
	reader.LazyQuotes = true
	if _, err := reader.Read(); err != nil { // skip header
		return err
	}

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 4 {
			continue
		}
		lgCode := rec[0]
		pref := rec[1]
		kanaPref := rec[2]
		romaPref := rec[3]

		prefKey := lgCode
		pos := posMap[lgCode]

		if err := db.UpsertPref(lgCode, prefKey, pref, kanaPref, romaPref, pos[1], pos[0]); err != nil {
			return err
		}
	}
	return nil
}

// --- City ---

var downloadedCityPrefs = sync.Map{} // track which pref city files were already downloaded

func (d *Downloader) downloadAndImportCity(prefCode string) error {
	if _, ok := downloadedCityPrefs.LoadOrStore(prefCode, true); ok {
		return nil // already downloaded
	}

	mainURL := fmt.Sprintf("%s/mt_city/pref/mt_city_pref%s.csv.zip", dataBaseURL, prefCode)
	posURL := fmt.Sprintf("%s/mt_city_pos/pref/mt_city_pos_pref%s.csv.zip", dataBaseURL, prefCode)

	mainData, err := d.downloadCSV(mainURL)
	if err != nil {
		return err
	}
	posData, err := d.downloadCSV(posURL)
	if err != nil {
		return err
	}

	posMap := buildPosMap2(posData) // col 0=lg_code, 1=rep_lon, 2=rep_lat

	// Build pref_key map: need pref name → pref_key mapping
	prefKeyMap, err := d.getPrefKeyMap()
	if err != nil {
		return err
	}

	db, err := database.OpenDownloadCommonDB(d.opts.DataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	reader := csv.NewReader(bytes.NewReader(mainData))
	reader.LazyQuotes = true
	if _, err := reader.Read(); err != nil {
		return err
	}

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 13 {
			continue
		}
		// lg_code,pref,pref_kana,pref_roma,county,county_kana,county_roma,city,city_kana,city_roma,ward,ward_kana,ward_roma,...
		lgCode := rec[0]
		prefName := rec[1]
		county := rec[4]
		city := rec[7]
		kanaCity := rec[8]
		romaCity := rec[9]
		ward := rec[10]

		prefKey := prefKeyMap[prefName]
		if prefKey == "" {
			prefKey = prefCode + "0001"
		}
		cityKey := lgCode
		pos := posMap[lgCode]

		if err := db.UpsertCity(lgCode, prefKey, cityKey, prefName, county, city, ward, kanaCity, romaCity, pos[1], pos[0]); err != nil {
			return err
		}
	}
	return nil
}

// getPrefKeyMap returns a map of pref name → pref_key from the database
func (d *Downloader) getPrefKeyMap() (map[string]string, error) {
	db, err := database.OpenDownloadCommonDB(d.opts.DataDir)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return db.GetPrefKeyMap()
}

// --- Town ---

func (d *Downloader) downloadAndImportTown(lgCode string) error {
	mainURL := fmt.Sprintf("%s/mt_town/city/mt_town_city%s.csv.zip", dataBaseURL, lgCode)
	posURL := fmt.Sprintf("%s/mt_town_pos/city/mt_town_pos_city%s.csv.zip", dataBaseURL, lgCode)

	mainData, err := d.downloadCSV(mainURL)
	if err != nil {
		return err
	}
	posData, err := d.downloadCSV(posURL)
	if err != nil {
		return err
	}

	// town_pos: lg_code, machiaza_id, rsdt_addr_flg, rep_lon, rep_lat, ...
	posMap := buildPosMap3(posData) // key = lg_code+machiaza_id

	prefKeyMap, err := d.getPrefKeyMap()
	if err != nil {
		return err
	}

	db, err := database.OpenDownloadCommonDB(d.opts.DataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	reader := csv.NewReader(bytes.NewReader(mainData))
	reader.LazyQuotes = true
	if _, err := reader.Read(); err != nil {
		return err
	}

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 29 {
			continue
		}
		// lg_code,machiaza_id,machiaza_type,pref,pref_kana,pref_roma,
		// county,county_kana,county_roma,city,city_kana,city_roma,
		// ward,ward_kana,ward_roma,oaza_cho,oaza_cho_kana,oaza_cho_roma,
		// chome,chome_kana,chome_number,koaza,koaza_kana,koaza_roma,
		// machiaza_dist,rsdt_addr_flg,rsdt_addr_mtd_code,oaza_cho_aka_flg,koaza_aka_code,...
		lgC := rec[0]
		machiazaID := rec[1]
		prefName := rec[3]
		county := rec[6]
		city := rec[9]
		ward := rec[12]
		oazaCho := rec[15]
		chome := rec[18]
		koaza := rec[21]
		rsdtAddrFlg, _ := strconv.Atoi(rec[25])
		koazaAkaCode, _ := strconv.Atoi(rec[28])

		prefKey := prefKeyMap[prefName]
		if prefKey == "" {
			prefKey = lgC[:2] + "0001"
		}
		cityKey := lgC
		townKey := lgC + machiazaID

		posKey := lgC + machiazaID
		pos := posMap[posKey]

		if err := db.UpsertTown(lgC, prefKey, cityKey, townKey, machiazaID,
			prefName, county, city, ward, oazaCho, chome, koaza,
			koazaAkaCode, rsdtAddrFlg, pos[1], pos[0]); err != nil {
			return err
		}
	}
	return nil
}

// --- Residential Block ---

func (d *Downloader) downloadAndImportRsdtBlk(lgCode string) error {
	mainURL := fmt.Sprintf("%s/mt_rsdtdsp_blk/city/mt_rsdtdsp_blk_city%s.csv.zip", dataBaseURL, lgCode)
	posURL := fmt.Sprintf("%s/mt_rsdtdsp_blk_pos/city/mt_rsdtdsp_blk_pos_city%s.csv.zip", dataBaseURL, lgCode)

	mainData, err := d.downloadCSV(mainURL)
	if err != nil {
		return err
	}
	posData, err := d.downloadCSV(posURL)
	if err != nil {
		return err
	}

	// blk_pos: lg_code,machiaza_id,blk_id,rsdt_addr_flg,rsdt_addr_mtd_code,rep_lon,rep_lat,...
	posMap := buildPosMap4(posData) // key = lg_code+machiaza_id+blk_id

	lgDB, err := database.OpenDownloadLGDB(d.opts.DataDir, lgCode)
	if err != nil {
		return err
	}
	defer lgDB.Close()

	reader := csv.NewReader(bytes.NewReader(mainData))
	reader.LazyQuotes = true
	if _, err := reader.Read(); err != nil {
		return err
	}

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 10 {
			continue
		}
		// lg_code,machiaza_id,blk_id,city,ward,oaza_cho,chome,koaza,machiaza_dist,blk_num,rsdt_addr_flg,...
		lgC := rec[0]
		machiazaID := rec[1]
		blkID := rec[2]
		blkNum := rec[9]

		townKey := lgC + machiazaID
		blkKey := lgC + machiazaID + blkID

		posKey := lgC + machiazaID + blkID
		pos := posMap[posKey]

		if err := lgDB.UpsertRsdtBlk(lgC, townKey, blkKey, blkNum, pos[1], pos[0]); err != nil {
			return err
		}
	}
	return nil
}

// --- Residential Display ---

func (d *Downloader) downloadAndImportRsdtDsp(lgCode string) error {
	mainURL := fmt.Sprintf("%s/mt_rsdtdsp_rsdt/city/mt_rsdtdsp_rsdt_city%s.csv.zip", dataBaseURL, lgCode)
	posURL := fmt.Sprintf("%s/mt_rsdtdsp_rsdt_pos/city/mt_rsdtdsp_rsdt_pos_city%s.csv.zip", dataBaseURL, lgCode)

	mainData, err := d.downloadCSV(mainURL)
	if err != nil {
		return err
	}
	posData, err := d.downloadCSV(posURL)
	if err != nil {
		return err
	}

	// rsdt_pos: lg_code,machiaza_id,blk_id,rsdt_id,rsdt2_id,rsdt_addr_flg,...,rep_lon,rep_lat,...
	posMap := buildPosMap5(posData) // key = lg_code+machiaza_id+blk_id+rsdt_id+rsdt2_id

	lgDB, err := database.OpenDownloadLGDB(d.opts.DataDir, lgCode)
	if err != nil {
		return err
	}
	defer lgDB.Close()

	reader := csv.NewReader(bytes.NewReader(mainData))
	reader.LazyQuotes = true
	if _, err := reader.Read(); err != nil {
		return err
	}

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 5 {
			continue
		}
		// lg_code,machiaza_id,blk_id,rsdt_id,rsdt2_id,city,ward,oaza_cho,chome,koaza,...
		lgC := rec[0]
		machiazaID := rec[1]
		blkID := rec[2]
		rsdtID := rec[3]
		rsdtID2 := rec[4]

		townKey := lgC + machiazaID
		blkKey := lgC + machiazaID + blkID
		rsdtKey := lgC + machiazaID + blkID + rsdtID + rsdtID2

		posKey := lgC + machiazaID + blkID + rsdtID + rsdtID2
		pos := posMap[posKey]

		var rID, rID2 *string
		if rsdtID != "" {
			s := rsdtID
			rID = &s
		}
		if rsdtID2 != "" {
			s := rsdtID2
			rID2 = &s
		}

		if err := lgDB.UpsertRsdtDsp(lgC, townKey, blkKey, rsdtKey, rID, rID2, pos[1], pos[0]); err != nil {
			return err
		}
	}
	return nil
}

// --- Parcel ---

func (d *Downloader) downloadAndImportParcel(lgCode string) error {
	mainURL := fmt.Sprintf("%s/mt_parcel/city/mt_parcel_city%s.csv.zip", dataBaseURL, lgCode)
	posURL := fmt.Sprintf("%s/mt_parcel_pos/city/mt_parcel_pos_city%s.csv.zip", dataBaseURL, lgCode)

	mainData, err := d.downloadCSV(mainURL)
	if err != nil {
		return err
	}
	posData, err := d.downloadCSV(posURL)
	if err != nil {
		return err
	}

	// parcel_pos: lg_code,machiaza_id,prc_id,rsdt_addr_flg,...,rep_lon,rep_lat,...
	type parcelPosKey struct{ lgCode, machiazaID, prcID string }
	posMapRaw := make(map[parcelPosKey][2]float64)
	pr := csv.NewReader(bytes.NewReader(posData))
	pr.LazyQuotes = true
	pr.Read() // skip header
	for {
		rec, err := pr.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 8 {
			continue
		}
		lon, _ := strconv.ParseFloat(rec[5], 64)
		lat, _ := strconv.ParseFloat(rec[6], 64)
		posMapRaw[parcelPosKey{rec[0], rec[1], rec[2]}] = [2]float64{lon, lat}
	}

	lgDB, err := database.OpenDownloadLGDB(d.opts.DataDir, lgCode)
	if err != nil {
		return err
	}
	defer lgDB.Close()

	reader := csv.NewReader(bytes.NewReader(mainData))
	reader.LazyQuotes = true
	if _, err := reader.Read(); err != nil {
		return err
	}

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 8 {
			continue
		}
		// lg_code,machiaza_id,prc_id,city,ward,oaza_cho,chome,koaza,...,prc_num1,prc_num2,prc_num3,...
		lgC := rec[0]
		machiazaID := rec[1]
		prcID := rec[2]

		// prc_num fields vary by CSV version; find them
		prcNum1, prcNum2, prcNum3 := "", "", ""
		if len(rec) > 12 {
			prcNum1 = rec[9]
			prcNum2 = rec[10]
			prcNum3 = rec[11]
		}

		townKey := lgC + machiazaID
		parcelKey := lgC + machiazaID + prcID

		pos := posMapRaw[parcelPosKey{lgC, machiazaID, prcID}]

		if err := lgDB.UpsertParcel(lgC, townKey, parcelKey, prcNum1, prcNum2, prcNum3, pos[1], pos[0]); err != nil {
			return err
		}
	}
	return nil
}

// --- HTTP helpers ---

// downloadCSV downloads a zip file and returns the CSV content inside
func (d *Downloader) downloadCSV(url string) ([]byte, error) {
	resp, err := d.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("404: %s", url)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, fmt.Errorf("bad zip %s: %w", url, err)
	}

	for _, f := range zr.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			return data, err
		}
	}
	return nil, fmt.Errorf("no CSV in zip: %s", url)
}

// --- Position map builders ---

// buildPosMap2 builds key=col0 → [lon,lat] (for pref/city_pos: lg_code,rep_lon,rep_lat,...)
func buildPosMap2(data []byte) map[string][2]float64 {
	m := make(map[string][2]float64)
	r := csv.NewReader(bytes.NewReader(data))
	r.LazyQuotes = true
	r.Read() // skip header
	for {
		rec, err := r.Read()
		if err != nil || len(rec) < 3 {
			if err == io.EOF {
				break
			}
			continue
		}
		lon, _ := strconv.ParseFloat(rec[1], 64)
		lat, _ := strconv.ParseFloat(rec[2], 64)
		m[rec[0]] = [2]float64{lon, lat}
	}
	return m
}

// buildPosMap3 builds key=col0+col1 → [lon,lat] (for town_pos: lg_code,machiaza_id,_,rep_lon,rep_lat,...)
func buildPosMap3(data []byte) map[string][2]float64 {
	m := make(map[string][2]float64)
	r := csv.NewReader(bytes.NewReader(data))
	r.LazyQuotes = true
	r.Read()
	for {
		rec, err := r.Read()
		if err != nil || len(rec) < 5 {
			if err == io.EOF {
				break
			}
			continue
		}
		lon, _ := strconv.ParseFloat(rec[3], 64)
		lat, _ := strconv.ParseFloat(rec[4], 64)
		m[rec[0]+rec[1]] = [2]float64{lon, lat}
	}
	return m
}

// buildPosMap4 builds key=col0+col1+col2 → [lon,lat] (for blk_pos: lg_code,machiaza_id,blk_id,_,_,rep_lon,rep_lat,...)
func buildPosMap4(data []byte) map[string][2]float64 {
	m := make(map[string][2]float64)
	r := csv.NewReader(bytes.NewReader(data))
	r.LazyQuotes = true
	r.Read()
	for {
		rec, err := r.Read()
		if err != nil || len(rec) < 7 {
			if err == io.EOF {
				break
			}
			continue
		}
		lon, _ := strconv.ParseFloat(rec[5], 64)
		lat, _ := strconv.ParseFloat(rec[6], 64)
		m[rec[0]+rec[1]+rec[2]] = [2]float64{lon, lat}
	}
	return m
}

// buildPosMap5 builds key=col0+col1+col2+col3+col4 → [lon,lat] (for rsdt_pos)
func buildPosMap5(data []byte) map[string][2]float64 {
	m := make(map[string][2]float64)
	r := csv.NewReader(bytes.NewReader(data))
	r.LazyQuotes = true
	r.Read()
	for {
		rec, err := r.Read()
		if err != nil || len(rec) < 9 {
			if err == io.EOF {
				break
			}
			continue
		}
		lon, _ := strconv.ParseFloat(rec[7], 64)
		lat, _ := strconv.ParseFloat(rec[8], 64)
		m[rec[0]+rec[1]+rec[2]+rec[3]+rec[4]] = [2]float64{lon, lat}
	}
	return m
}

// filterLGCodes validates and deduplicates LG codes
func filterLGCodes(codes []string) []string {
	var result []string
	seen := make(map[string]bool)
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if len(code) != 6 {
			continue
		}
		valid := true
		for _, c := range code {
			if c < '0' || c > '9' {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		prefCode, _ := strconv.Atoi(code[:2])
		if prefCode < 1 || prefCode > 47 {
			continue
		}
		if !seen[code] {
			seen[code] = true
			result = append(result, code)
		}
	}
	return result
}

// GetDownloadPath returns the path for a specific LG code database
func GetDownloadPath(dataDir, lgCode string) string {
	return filepath.Join(dataDir, fmt.Sprintf("abrg-%s.sqlite", lgCode))
}

// Ensure sql is used (for nullable fields)
var _ = sql.NullString{}
