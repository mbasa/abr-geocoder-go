// Package steps provides the parcel matching step.
// Ported from TypeScript: src/usecases/geocode/steps/parcel-transform.ts
package steps

import (
	"regexp"
	"strings"

	"github.com/mbasa/abr-geocoder-go/internal/domain/types"
	"github.com/mbasa/abr-geocoder-go/internal/drivers/database"
	geocodemodels "github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/models"
)

// ParcelStep matches land parcel numbers
type ParcelStep struct {
	dataDir string
}

// NewParcelStep creates a new parcel matching step
func NewParcelStep(dataDir string) *ParcelStep {
	return &ParcelStep{dataDir: dataDir}
}

// parcelRe matches parcel number patterns like "123", "123-45", "123-45-6"
var parcelRe = regexp.MustCompile(`^(\d+)(?:-(\d+)(?:-(\d+))?)?`)

// Process attempts to match parcel numbers in the address
func (s *ParcelStep) Process(q *geocodemodels.Query) ([]*geocodemodels.Query, error) {
	if q.TempAddress == "" {
		return []*geocodemodels.Query{q}, nil
	}

	// Skip if not at town level
	if q.MatchLevel < types.MatchLevelTown {
		return []*geocodemodels.Query{q}, nil
	}

	// Skip parcel search for residential address towns (use rsdt_blk path instead)
	if q.RsdtAddrFlg == 1 {
		return []*geocodemodels.Query{q}, nil
	}

	// Clean the temp address
	addr := strings.TrimSpace(q.TempAddress)
	addr = strings.TrimLeft(addr, "-−")
	addr = strings.TrimSpace(addr)

	// Extract parcel number pattern
	match := parcelRe.FindStringSubmatch(addr)
	if match == nil {
		return []*geocodemodels.Query{q}, nil
	}

	prcNum1 := match[1]
	prcNum2 := ""
	prcNum3 := ""
	if len(match) > 2 {
		prcNum2 = match[2]
	}
	if len(match) > 3 {
		prcNum3 = match[3]
	}

	// Open the parcel database
	parcelDB, err := database.OpenParcelDB(s.dataDir, q.LGCode)
	if err != nil || parcelDB == nil {
		return []*geocodemodels.Query{q}, nil
	}
	defer parcelDB.Close()

	// Try matching with different levels of specificity
	var n1, n2, n3 *string
	n1 = &prcNum1
	if prcNum2 != "" {
		n2 = &prcNum2
	}
	if prcNum3 != "" {
		n3 = &prcNum3
	}

	rows, err := parcelDB.GetParcelRows(q.TownKey, n1, n2, n3)
	if err != nil || len(rows) == 0 {
		// Try without prc_num3
		if n3 != nil {
			rows, err = parcelDB.GetParcelRows(q.TownKey, n1, n2, nil)
		}
	}
	if err != nil || len(rows) == 0 {
		// Try without prc_num2
		rows, err = parcelDB.GetParcelRows(q.TownKey, n1, nil, nil)
	}
	if err != nil || len(rows) == 0 {
		return []*geocodemodels.Query{q}, nil
	}

	var results []*geocodemodels.Query
	matchedLen := len(match[0])

	for _, row := range rows {
		newQ := q.Copy()
		newQ.PrcNum1 = row.PrcNum1
		newQ.PrcNum2 = row.PrcNum2
		newQ.PrcNum3 = row.PrcNum3
		newQ.ParcelKey = row.ParcelKey
		newQ.RsdtAddrFlg = 0 // Parcel address
		newQ.Lat = row.RepLat
		newQ.Lon = row.RepLon
		newQ.MatchLevel = types.MatchLevelParcel

		// Advance temp address
		remaining := addr[matchedLen:]
		newQ.TempAddress = remaining
		newQ.MatchedChar += matchedLen

		results = append(results, newQ)
	}

	if len(results) == 0 {
		return []*geocodemodels.Query{q}, nil
	}

	return results, nil
}
