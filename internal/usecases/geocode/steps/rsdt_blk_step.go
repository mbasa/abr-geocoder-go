// Package steps provides the residential block matching step.
// Ported from TypeScript: src/usecases/geocode/steps/rsdt-blk-transform.ts
package steps

import (
	"regexp"
	"strings"

	"github.com/mbasa/abr-geocoder-go/internal/domain/types"
	"github.com/mbasa/abr-geocoder-go/internal/drivers/database"
	geocodemodels "github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/models"
)

// RsdtBlkStep matches residential block numbers
type RsdtBlkStep struct {
	dataDir string
}

// NewRsdtBlkStep creates a new residential block matching step
func NewRsdtBlkStep(dataDir string) *RsdtBlkStep {
	return &RsdtBlkStep{dataDir: dataDir}
}

// blockNumRe matches a block number pattern like "1" or "12" at start of string
var blockNumRe = regexp.MustCompile(`^(\d+)`)

// Process attempts to match the residential block number
func (s *RsdtBlkStep) Process(q *geocodemodels.Query) ([]*geocodemodels.Query, error) {
	if q.TempAddress == "" {
		return []*geocodemodels.Query{q}, nil
	}

	// Skip if already at residential detail level
	if q.MatchLevel >= types.MatchLevelResidentialDetail {
		return []*geocodemodels.Query{q}, nil
	}

	// Skip if not at town level
	if q.MatchLevel < types.MatchLevelTown {
		return []*geocodemodels.Query{q}, nil
	}

	// Skip Kyoto City (koaza_aka_code == 2) - Kyoto uses street name system
	if q.KoazaAkaCode == 2 {
		return []*geocodemodels.Query{q}, nil
	}

	// Skip if this is a parcel address
	if q.RsdtAddrFlg == 0 {
		return []*geocodemodels.Query{q}, nil
	}

	// Clean the temp address
	addr := strings.TrimSpace(q.TempAddress)
	addr = strings.TrimLeft(addr, "-−")
	addr = strings.TrimSpace(addr)

	// Extract the block number from the beginning of the remaining address
	match := blockNumRe.FindStringSubmatch(addr)
	if match == nil {
		return []*geocodemodels.Query{q}, nil
	}

	blkNum := match[1]

	// Open the residential block database
	rsdtDB, err := database.OpenRsdtBlkDB(s.dataDir, q.LGCode)
	if err != nil || rsdtDB == nil {
		return []*geocodemodels.Query{q}, nil
	}
	defer rsdtDB.Close()

	// Query for matching block numbers
	rows, err := rsdtDB.GetBlockNumRows(q.TownKey, blkNum)
	if err != nil || len(rows) == 0 {
		return []*geocodemodels.Query{q}, nil
	}

	var results []*geocodemodels.Query
	blkNumLen := len(blkNum)

	for _, row := range rows {
		newQ := q.Copy()
		newQ.BlkNum = row.BlkNum
		newQ.BlkKey = row.BlkKey
		newQ.Lat = row.RepLat
		newQ.Lon = row.RepLon
		newQ.MatchLevel = types.MatchLevelResidentialBlock
		newQ.CoordinateLevel = types.CoordinateLevelBlock

		// Advance the temp address past the block number
		remaining := addr[blkNumLen:]
		remaining = strings.TrimLeft(remaining, "-−")
		newQ.TempAddress = remaining
		newQ.MatchedChar += blkNumLen

		results = append(results, newQ)
	}

	if len(results) == 0 {
		return []*geocodemodels.Query{q}, nil
	}

	return results, nil
}
