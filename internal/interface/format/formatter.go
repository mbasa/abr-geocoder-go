// Package format provides output formatting for geocoding results.
// Ported from TypeScript: src/interface/format/
package format

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mbasa/abr-geocoder-go/internal/domain/types"
	geocodemodels "github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/models"
)

// Formatter writes geocoding results in a specific format
type Formatter interface {
	WriteHeader(w io.Writer) error
	WriteResult(w io.Writer, result *geocodemodels.GeoCodeResult) error
	WriteFooter(w io.Writer) error
	MimeType() string
}

// NewFormatter creates a formatter for the given output format
func NewFormatter(format types.OutputFormat, debug bool) (Formatter, error) {
	switch format {
	case types.OutputFormatJSON:
		return &JSONFormatter{debug: debug, first: true}, nil
	case types.OutputFormatNDJSON:
		return &NDJSONFormatter{debug: debug}, nil
	case types.OutputFormatCSV:
		return &CSVFormatter{debug: debug, first: true}, nil
	case types.OutputFormatSimplified:
		return &SimplifiedCSVFormatter{debug: debug, first: true}, nil
	case types.OutputFormatGeoJSON:
		return &GeoJSONFormatter{debug: debug, first: true}, nil
	case types.OutputFormatNDGeoJSON:
		return &NDGeoJSONFormatter{debug: debug}, nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", format)
	}
}

// JSONFormatter outputs results as a JSON array
type JSONFormatter struct {
	debug bool
	first bool
}

func (f *JSONFormatter) MimeType() string { return "application/json" }

func (f *JSONFormatter) WriteHeader(w io.Writer) error {
	_, err := io.WriteString(w, "[\n")
	return err
}

func (f *JSONFormatter) WriteResult(w io.Writer, r *geocodemodels.GeoCodeResult) error {
	if !f.first {
		if _, err := io.WriteString(w, ",\n"); err != nil {
			return err
		}
	}
	f.first = false

	data := resultToMap(r, f.debug)
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func (f *JSONFormatter) WriteFooter(w io.Writer) error {
	_, err := io.WriteString(w, "\n]\n")
	return err
}

// NDJSONFormatter outputs results as newline-delimited JSON
type NDJSONFormatter struct {
	debug bool
}

func (f *NDJSONFormatter) MimeType() string { return "application/x-ndjson" }

func (f *NDJSONFormatter) WriteHeader(w io.Writer) error { return nil }

func (f *NDJSONFormatter) WriteResult(w io.Writer, r *geocodemodels.GeoCodeResult) error {
	data := resultToMap(r, f.debug)
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", b)
	return err
}

func (f *NDJSONFormatter) WriteFooter(w io.Writer) error { return nil }

// CSVFormatter outputs results as CSV
type CSVFormatter struct {
	debug bool
	first bool
}

func (f *CSVFormatter) MimeType() string { return "text/csv" }

func (f *CSVFormatter) WriteHeader(w io.Writer) error {
	headers := csvHeaders(f.debug)
	cw := csv.NewWriter(w)
	if err := cw.Write(headers); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

func (f *CSVFormatter) WriteResult(w io.Writer, r *geocodemodels.GeoCodeResult) error {
	row := resultToCSVRow(r, f.debug)
	cw := csv.NewWriter(w)
	if err := cw.Write(row); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

func (f *CSVFormatter) WriteFooter(w io.Writer) error { return nil }

// SimplifiedCSVFormatter outputs minimal CSV with input, output, score, match_level
type SimplifiedCSVFormatter struct {
	debug bool
	first bool
}

func (f *SimplifiedCSVFormatter) MimeType() string { return "text/csv" }

func (f *SimplifiedCSVFormatter) WriteHeader(w io.Writer) error {
	headers := []string{"input", "output", "score", "match_level"}
	if f.debug {
		headers = append(headers, "pref_key", "city_key", "town_key", "parcel_key", "rsdtblk_key", "rsdtdsp_key")
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(headers); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

func (f *SimplifiedCSVFormatter) WriteResult(w io.Writer, r *geocodemodels.GeoCodeResult) error {
	row := []string{
		r.Input,
		r.Output,
		strconv.FormatFloat(r.Score, 'f', 4, 64),
		r.MatchLevel,
	}
	if f.debug {
		row = append(row, r.PrefKey, r.CityKey, r.TownKey, r.ParcelKey, r.BlkKey, r.RsdtKey)
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(row); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

func (f *SimplifiedCSVFormatter) WriteFooter(w io.Writer) error { return nil }

// GeoJSONFormatter outputs results as a GeoJSON FeatureCollection
type GeoJSONFormatter struct {
	debug bool
	first bool
}

func (f *GeoJSONFormatter) MimeType() string { return "application/geo+json" }

func (f *GeoJSONFormatter) WriteHeader(w io.Writer) error {
	_, err := io.WriteString(w, `{"type":"FeatureCollection","features":[`)
	return err
}

func (f *GeoJSONFormatter) WriteResult(w io.Writer, r *geocodemodels.GeoCodeResult) error {
	if !f.first {
		if _, err := io.WriteString(w, ","); err != nil {
			return err
		}
	}
	f.first = false

	b, err := json.Marshal(toGeoJSONFeature(r))
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func (f *GeoJSONFormatter) WriteFooter(w io.Writer) error {
	_, err := io.WriteString(w, "]}\n")
	return err
}

// NDGeoJSONFormatter outputs results as newline-delimited GeoJSON features
type NDGeoJSONFormatter struct {
	debug bool
}

func (f *NDGeoJSONFormatter) MimeType() string { return "application/geo+json" }

func (f *NDGeoJSONFormatter) WriteHeader(w io.Writer) error { return nil }

func (f *NDGeoJSONFormatter) WriteResult(w io.Writer, r *geocodemodels.GeoCodeResult) error {
	b, err := json.Marshal(toGeoJSONFeature(r))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", b)
	return err
}

func (f *NDGeoJSONFormatter) WriteFooter(w io.Writer) error { return nil }

// Helper functions

func resultToMap(r *geocodemodels.GeoCodeResult, debug bool) map[string]interface{} {
	m := map[string]interface{}{
		"input":        r.Input,
		"output":       r.Output,
		"score":        r.Score,
		"match_level":  r.MatchLevel,
		"pref":         r.Pref,
		"county":       r.County,
		"city":         r.City,
		"ward":         r.Ward,
		"lg_code":      r.LGCode,
		"rsdt_addr_flg": r.RsdtAddrFlg,
	}

	if r.Lat != nil {
		m["lat"] = *r.Lat
		m["lon"] = *r.Lon
	} else {
		m["lat"] = nil
		m["lon"] = nil
	}

	if r.OazaCho != "" {
		m["oaza_cho"] = r.OazaCho
	}
	if r.Chome != "" {
		m["chome"] = r.Chome
	}
	if r.Koaza != "" {
		m["koaza"] = r.Koaza
	}
	if r.BlkNum != "" {
		m["blk_num"] = r.BlkNum
	}
	if r.RsdtID != "" {
		m["rsdt_id"] = r.RsdtID
	}
	if r.RsdtID2 != "" {
		m["rsdt_id2"] = r.RsdtID2
	}
	if r.PrcNum1 != "" {
		m["prc_num1"] = r.PrcNum1
	}
	if r.PrcNum2 != "" {
		m["prc_num2"] = r.PrcNum2
	}
	if r.PrcNum3 != "" {
		m["prc_num3"] = r.PrcNum3
	}

	if debug {
		m["pref_key"] = r.PrefKey
		m["city_key"] = r.CityKey
		m["town_key"] = r.TownKey
		m["blk_key"] = r.BlkKey
		m["rsdt_key"] = r.RsdtKey
		m["parcel_key"] = r.ParcelKey
	}

	return m
}

func csvHeaders(debug bool) []string {
	headers := []string{
		"input", "output", "score", "match_level",
		"pref", "county", "city", "ward", "lg_code",
		"oaza_cho", "chome", "koaza",
		"blk_num", "rsdt_id", "rsdt_id2",
		"prc_num1", "prc_num2", "prc_num3",
		"lat", "lon", "rsdt_addr_flg",
	}
	if debug {
		headers = append(headers, "pref_key", "city_key", "town_key", "blk_key", "rsdt_key", "parcel_key")
	}
	return headers
}

func resultToCSVRow(r *geocodemodels.GeoCodeResult, debug bool) []string {
	lat, lon := "", ""
	if r.Lat != nil {
		lat = strconv.FormatFloat(*r.Lat, 'f', 8, 64)
		lon = strconv.FormatFloat(*r.Lon, 'f', 8, 64)
	}

	row := []string{
		r.Input,
		r.Output,
		strconv.FormatFloat(r.Score, 'f', 4, 64),
		r.MatchLevel,
		r.Pref,
		r.County,
		r.City,
		r.Ward,
		r.LGCode,
		r.OazaCho,
		r.Chome,
		r.Koaza,
		r.BlkNum,
		r.RsdtID,
		r.RsdtID2,
		r.PrcNum1,
		r.PrcNum2,
		r.PrcNum3,
		lat,
		lon,
		strconv.Itoa(r.RsdtAddrFlg),
	}

	if debug {
		row = append(row, r.PrefKey, r.CityKey, r.TownKey, r.BlkKey, r.RsdtKey, r.ParcelKey)
	}

	return row
}

type geoJSONFeature struct {
	Type       string                 `json:"type"`
	Geometry   *geoJSONPoint          `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

type geoJSONPoint struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

func toGeoJSONFeature(r *geocodemodels.GeoCodeResult) *geoJSONFeature {
	f := &geoJSONFeature{
		Type: "Feature",
		Properties: map[string]interface{}{
			"input":        r.Input,
			"output":       r.Output,
			"score":        r.Score,
			"match_level":  r.MatchLevel,
			"pref":         r.Pref,
			"county":       r.County,
			"city":         r.City,
			"ward":         r.Ward,
			"oaza_cho":     r.OazaCho,
			"chome":        r.Chome,
			"koaza":        r.Koaza,
			"blk_num":      r.BlkNum,
			"rsdt_id":      r.RsdtID,
			"rsdt_id2":     r.RsdtID2,
			"prc_num1":     r.PrcNum1,
			"prc_num2":     r.PrcNum2,
			"prc_num3":     r.PrcNum3,
			"lg_code":      r.LGCode,
			"rsdt_addr_flg": r.RsdtAddrFlg,
		},
	}

	if r.Lat != nil {
		f.Geometry = &geoJSONPoint{
			Type:        "Point",
			Coordinates: []float64{*r.Lon, *r.Lat}, // GeoJSON is [lon, lat]
		}
	}

	return f
}

// WriteResults writes multiple results using a formatter
func WriteResults(w io.Writer, results []*geocodemodels.GeoCodeResult, format types.OutputFormat, debug bool) error {
	formatter, err := NewFormatter(format, debug)
	if err != nil {
		return err
	}

	if err := formatter.WriteHeader(w); err != nil {
		return err
	}

	for _, r := range results {
		if err := formatter.WriteResult(w, r); err != nil {
			return err
		}
	}

	return formatter.WriteFooter(w)
}

// FormatResult formats a single result as a string
func FormatResult(result *geocodemodels.GeoCodeResult, format types.OutputFormat, debug bool) (string, error) {
	var sb strings.Builder
	if err := WriteResults(&sb, []*geocodemodels.GeoCodeResult{result}, format, debug); err != nil {
		return "", err
	}
	return sb.String(), nil
}
