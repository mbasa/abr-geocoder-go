// Package steps provides the city/county matching step.
// Ported from TypeScript: src/usecases/geocode/steps/county-and-city-transform.ts
package steps

import (
	"github.com/mbasa/abr-geocoder-go/internal/domain/models"
	"github.com/mbasa/abr-geocoder-go/internal/domain/types"
	geocodemodels "github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/models"
	"github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/services"
)

// CityStep matches the city/county portion of an address
type CityStep struct {
	trie      *geocodemodels.Trie[*models.CityMatchingInfo]
	fuzzyChar rune
}

// NewCityStep creates a new city/county matching step
func NewCityStep(trie *geocodemodels.Trie[*models.CityMatchingInfo], fuzzyChar string) *CityStep {
	fc := '?'
	if len([]rune(fuzzyChar)) > 0 {
		fc = []rune(fuzzyChar)[0]
	}
	return &CityStep{trie: trie, fuzzyChar: fc}
}

// Process attempts to match the city/county in the query's address
func (s *CityStep) Process(q *geocodemodels.Query) ([]*geocodemodels.Query, error) {
	if q.TempAddress == "" {
		return []*geocodemodels.Query{q}, nil
	}

	// Skip if already at city level or higher
	if q.MatchLevel >= types.MatchLevelCity {
		return []*geocodemodels.Query{q}, nil
	}

	normalized := services.NormalizeForMatching(q.TempAddress)

	matches := s.trie.FindAll(normalized)
	if len(matches) == 0 {
		return []*geocodemodels.Query{q}, nil
	}

	var results []*geocodemodels.Query
	seen := make(map[string]bool)

	for _, match := range matches {
		for _, cityInfo := range match.Values {
			// If we have a prefecture, validate consistency
			if q.PrefKey != "" && cityInfo.PrefKey != q.PrefKey {
				continue
			}

			key := cityInfo.CityKey
			if seen[key] {
				continue
			}
			seen[key] = true

			newQ := q.Copy()
			newQ.County = cityInfo.County
			newQ.City = cityInfo.City
			newQ.Ward = cityInfo.Ward
			newQ.PrefKey = cityInfo.PrefKey
			newQ.CityKey = cityInfo.CityKey
			newQ.LGCode = cityInfo.LGCode
			newQ.Lat = cityInfo.RepLat
			newQ.Lon = cityInfo.RepLon
			newQ.MatchLevel = types.MatchLevelCity
			newQ.CoordinateLevel = types.CoordinateLevelCity

			// Calculate remaining address
			inputRunes := []rune(q.TempAddress)
			matchedLen := len([]rune(match.Matched))
			if matchedLen <= len(inputRunes) {
				newQ.TempAddress = string(inputRunes[matchedLen:])
			} else {
				newQ.TempAddress = ""
			}

			newQ.MatchedChar += len([]rune(match.Matched))
			results = append(results, newQ)
		}
	}

	if len(results) == 0 {
		return []*geocodemodels.Query{q}, nil
	}

	return results, nil
}
