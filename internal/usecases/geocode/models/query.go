// Package models provides data models for the geocoding process.
// Ported from TypeScript: src/usecases/geocode/models/query.ts
package models

import (
	"fmt"
	"math"
	"strings"

	"github.com/mbasa/abr-geocoder-go/internal/domain/types"
)

// Query holds the state of a geocoding operation
type Query struct {
	// Input
	Input string

	// Address components
	Pref    string
	County  string
	City    string
	Ward    string
	OazaCho string
	Chome   string
	Koaza   string

	// Keys for database lookups
	PrefKey   string
	CityKey   string
	TownKey   string
	BlkKey    string
	RsdtKey   string
	ParcelKey string

	// Local government code
	LGCode string

	// Coordinates
	Lat float64
	Lon float64

	// Match information
	MatchLevel      types.MatchLevel
	CoordinateLevel types.CoordinateLevel

	// Scores and flags
	Score           float64
	SimilarityScore float64
	MatchedChar     int
	UnmatchedChar   int
	RsdtAddrFlg     int
	KoazaAkaCode    int

	// Residential address info
	BlkNum   string
	RsdtID   string
	RsdtID2  string

	// Parcel info
	PrcNum1 string
	PrcNum2 string
	PrcNum3 string

	// Remaining address after matching
	TempAddress string

	// Ambiguity tracking
	AmbiguousCnt int

	// Debug info
	ProcessingTimestamp int64
}

// GeoCodeResult is the output result of a geocoding operation
type GeoCodeResult struct {
	Input       string  `json:"input"`
	Output      string  `json:"output"`
	Score       float64 `json:"score"`
	MatchLevel  string  `json:"match_level"`
	Lat         *float64 `json:"lat,omitempty"`
	Lon         *float64 `json:"lon,omitempty"`
	Pref        string  `json:"pref,omitempty"`
	County      string  `json:"county,omitempty"`
	City        string  `json:"city,omitempty"`
	Ward        string  `json:"ward,omitempty"`
	OazaCho     string  `json:"oaza_cho,omitempty"`
	Chome       string  `json:"chome,omitempty"`
	Koaza       string  `json:"koaza,omitempty"`
	BlkNum      string  `json:"blk_num,omitempty"`
	RsdtID      string  `json:"rsdt_id,omitempty"`
	RsdtID2     string  `json:"rsdt_id2,omitempty"`
	PrcNum1     string  `json:"prc_num1,omitempty"`
	PrcNum2     string  `json:"prc_num2,omitempty"`
	PrcNum3     string  `json:"prc_num3,omitempty"`
	LGCode      string  `json:"lg_code,omitempty"`
	PrefKey     string  `json:"pref_key,omitempty"`
	CityKey     string  `json:"city_key,omitempty"`
	TownKey     string  `json:"town_key,omitempty"`
	BlkKey      string  `json:"blk_key,omitempty"`
	RsdtKey     string  `json:"rsdt_key,omitempty"`
	ParcelKey   string  `json:"parcel_key,omitempty"`
	RsdtAddrFlg int     `json:"rsdt_addr_flg"`
}

// Copy creates a deep copy of the Query
func (q *Query) Copy() *Query {
	c := *q
	return &c
}

// GetFormattedAddress returns the normalized full address
func (q *Query) GetFormattedAddress() string {
	var parts []string

	if q.Pref != "" {
		parts = append(parts, q.Pref)
	}
	if q.County != "" {
		parts = append(parts, q.County)
	}
	if q.City != "" {
		parts = append(parts, q.City)
	}
	if q.Ward != "" {
		parts = append(parts, q.Ward)
	}
	if q.OazaCho != "" {
		parts = append(parts, q.OazaCho)
	}
	if q.Chome != "" {
		parts = append(parts, q.Chome)
	}
	if q.Koaza != "" {
		parts = append(parts, q.Koaza)
	}
	if q.BlkNum != "" {
		parts = append(parts, q.BlkNum)
	}
	if q.RsdtID != "" {
		if q.RsdtID2 != "" {
			parts = append(parts, fmt.Sprintf("%s-%s", q.RsdtID, q.RsdtID2))
		} else {
			parts = append(parts, q.RsdtID)
		}
	}
	if q.PrcNum1 != "" {
		prc := q.PrcNum1
		if q.PrcNum2 != "" {
			prc += "-" + q.PrcNum2
			if q.PrcNum3 != "" {
				prc += "-" + q.PrcNum3
			}
		}
		parts = append(parts, prc)
	}

	return strings.Join(parts, "")
}

// ToResult converts the query to a GeoCodeResult
func (q *Query) ToResult() *GeoCodeResult {
	result := &GeoCodeResult{
		Input:       q.Input,
		Output:      q.GetFormattedAddress(),
		Score:       math.Round(q.Score*1000) / 1000,
		MatchLevel:  q.MatchLevel.String(),
		Pref:        q.Pref,
		County:      q.County,
		City:        q.City,
		Ward:        q.Ward,
		OazaCho:     q.OazaCho,
		Chome:       q.Chome,
		Koaza:       q.Koaza,
		BlkNum:      q.BlkNum,
		RsdtID:      q.RsdtID,
		RsdtID2:     q.RsdtID2,
		PrcNum1:     q.PrcNum1,
		PrcNum2:     q.PrcNum2,
		PrcNum3:     q.PrcNum3,
		LGCode:      q.LGCode,
		PrefKey:     q.PrefKey,
		CityKey:     q.CityKey,
		TownKey:     q.TownKey,
		BlkKey:      q.BlkKey,
		RsdtKey:     q.RsdtKey,
		ParcelKey:   q.ParcelKey,
		RsdtAddrFlg: q.RsdtAddrFlg,
	}

	if q.Lat != 0 || q.Lon != 0 {
		lat := q.Lat
		lon := q.Lon
		result.Lat = &lat
		result.Lon = &lon
	}

	return result
}

// Create initializes a new Query from an address string
func Create(address string) *Query {
	return &Query{
		Input:       address,
		TempAddress: address,
		MatchLevel:  types.MatchLevelUnmatch,
		Score:       0,
		RsdtAddrFlg: -1,
	}
}
