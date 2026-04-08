// Package steps provides the town/oaza/chome matching step.
// Ported from TypeScript: src/usecases/geocode/steps/oaza-chome-transform.ts
package steps

import (
	"github.com/mbasa/abr-geocoder-go/internal/domain/models"
	"github.com/mbasa/abr-geocoder-go/internal/domain/types"
	geocodemodels "github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/models"
	"github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/services"
)

// TownStep matches the town/oaza/chome portion of an address
type TownStep struct {
	trie      *geocodemodels.Trie[*models.TownMatchingInfo]
	fuzzyChar rune
}

// NewTownStep creates a new town matching step
func NewTownStep(trie *geocodemodels.Trie[*models.TownMatchingInfo], fuzzyChar string) *TownStep {
	fc := '?'
	if len([]rune(fuzzyChar)) > 0 {
		fc = []rune(fuzzyChar)[0]
	}
	return &TownStep{trie: trie, fuzzyChar: fc}
}

// Process attempts to match the town/oaza in the query's address
func (s *TownStep) Process(q *geocodemodels.Query) ([]*geocodemodels.Query, error) {
	if q.TempAddress == "" {
		return []*geocodemodels.Query{q}, nil
	}

	if q.MatchLevel < types.MatchLevelCity {
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
		for _, townInfo := range match.Values {
			key := townInfo.TownKey
			if seen[key] {
				continue
			}
			seen[key] = true

			newQ := q.Copy()
			newQ.OazaCho = townInfo.OazaCho
			newQ.Chome = townInfo.Chome
			newQ.Koaza = townInfo.Koaza
			newQ.TownKey = townInfo.TownKey
			newQ.KoazaAkaCode = townInfo.KoazaAkaCode
			newQ.RsdtAddrFlg = townInfo.RsdtAddrFlg
			newQ.Lat = townInfo.RepLat
			newQ.Lon = townInfo.RepLon
			newQ.MatchLevel = types.MatchLevelTown
			newQ.CoordinateLevel = types.CoordinateLevelTown

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
