// Package steps provides the prefecture matching step.
// Ported from TypeScript: src/usecases/geocode/steps/pref-transform.ts
package steps

import (
	"github.com/mbasa/abr-geocoder-go/internal/domain/models"
	"github.com/mbasa/abr-geocoder-go/internal/domain/types"
	geocodemodels "github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/models"
	"github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/services"
)

// PrefStep matches the prefecture portion of an address
type PrefStep struct {
	trie      *geocodemodels.Trie[*models.PrefInfo]
	fuzzyChar rune
}

// NewPrefStep creates a new prefecture matching step
func NewPrefStep(trie *geocodemodels.Trie[*models.PrefInfo], fuzzyChar string) *PrefStep {
	fc := '?'
	if len([]rune(fuzzyChar)) > 0 {
		fc = []rune(fuzzyChar)[0]
	}
	return &PrefStep{trie: trie, fuzzyChar: fc}
}

// Process attempts to match the prefecture in the query's address
func (s *PrefStep) Process(q *geocodemodels.Query) ([]*geocodemodels.Query, error) {
	if q.TempAddress == "" {
		return []*geocodemodels.Query{q}, nil
	}

	// Already matched city or higher - skip
	if q.MatchLevel >= types.MatchLevelCity {
		return []*geocodemodels.Query{q}, nil
	}

	// Normalize address for matching
	normalized := services.NormalizeForMatching(q.TempAddress)

	// Try to find prefecture prefix
	// Include extra challenge characters that follow prefecture names
	extraChallengeChars := []rune{'道', '都', '府', '県'}

	var results []*geocodemodels.Query

	matches := s.trie.FindAll(normalized)
	for _, match := range matches {
		for _, prefInfo := range match.Values {
			// Verify the match ends with a valid pref suffix
			matched := []rune(match.Matched)
			lastChar := matched[len(matched)-1]
			isValidSuffix := false
			for _, c := range extraChallengeChars {
				if lastChar == c {
					isValidSuffix = true
					break
				}
			}

			if !isValidSuffix {
				// The pref name itself might not end with these chars - check pref name
				prefRunes := []rune(prefInfo.Pref)
				if len(prefRunes) > 0 {
					lastPrefChar := prefRunes[len(prefRunes)-1]
					for _, c := range extraChallengeChars {
						if lastPrefChar == c {
							isValidSuffix = true
							break
						}
					}
				}
			}

			newQ := q.Copy()
			newQ.Pref = prefInfo.Pref
			newQ.PrefKey = prefInfo.PrefKey
			newQ.LGCode = prefInfo.LGCode
			newQ.Lat = prefInfo.RepLat
			newQ.Lon = prefInfo.RepLon
			newQ.MatchLevel = types.MatchLevelPrefecture
			newQ.CoordinateLevel = types.CoordinateLevelPrefecture

			// Restore the original remaining address
			inputRunes := []rune(q.TempAddress)
			matchedLen := len([]rune(match.Matched))
			if matchedLen <= len(inputRunes) {
				newQ.TempAddress = string(inputRunes[matchedLen:])
			} else {
				newQ.TempAddress = ""
			}

			newQ.MatchedChar = len(matched)
			results = append(results, newQ)
		}
	}

	if len(results) == 0 {
		// No prefecture found - return original
		return []*geocodemodels.Query{q}, nil
	}

	return results, nil
}
