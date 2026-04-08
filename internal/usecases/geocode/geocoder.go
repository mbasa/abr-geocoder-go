// Package geocode provides the core geocoding functionality.
// Ported from TypeScript: src/usecases/geocode/abr-geocoder.ts
package geocode

import (
	"fmt"
	"strings"

	"github.com/mbasa/abr-geocoder-go/internal/domain/models"
	"github.com/mbasa/abr-geocoder-go/internal/domain/types"
	"github.com/mbasa/abr-geocoder-go/internal/drivers/database"
	geocodemodels "github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/models"
	"github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/services"
	"github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/steps"
)

// Geocoder is the main address geocoding engine
type Geocoder struct {
	db           *database.GeocodeDB
	dataDir      string
	fuzzyChar    string
	searchTarget types.SearchTarget

	// Trie finders loaded at initialization
	prefTrie   *geocodemodels.Trie[*models.PrefInfo]
	cityTrie   *geocodemodels.Trie[*models.CityMatchingInfo]
	townTries  map[string]*geocodemodels.Trie[*models.TownMatchingInfo]
}

// GeocoderOptions configures the geocoder
type GeocoderOptions struct {
	DataDir      string
	FuzzyChar    string
	SearchTarget types.SearchTarget
	Debug        bool
}

// New creates and initializes a new Geocoder
func New(opts GeocoderOptions) (*Geocoder, error) {
	if opts.FuzzyChar == "" {
		opts.FuzzyChar = "?"
	}
	if !opts.SearchTarget.IsValid() {
		opts.SearchTarget = types.SearchTargetAll
	}

	db, err := database.OpenGeocodeDB(opts.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open geocode database: %w", err)
	}

	g := &Geocoder{
		db:           db,
		dataDir:      opts.DataDir,
		fuzzyChar:    opts.FuzzyChar,
		searchTarget: opts.SearchTarget,
		townTries:    make(map[string]*geocodemodels.Trie[*models.TownMatchingInfo]),
	}

	// Build trie data structures
	if err := g.buildPrefTrie(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to build prefecture trie: %w", err)
	}

	if err := g.buildCityTrie(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to build city trie: %w", err)
	}

	return g, nil
}

// Close releases resources
func (g *Geocoder) Close() error {
	return g.db.Close()
}

// buildPrefTrie builds the prefecture prefix tree
func (g *Geocoder) buildPrefTrie() error {
	prefs, err := g.db.GetPrefList()
	if err != nil {
		return err
	}

	g.prefTrie = geocodemodels.NewTrie[*models.PrefInfo]()
	for _, p := range prefs {
		info := &models.PrefInfo{
			PrefKey: p.PrefKey,
			Pref:    p.Pref,
			LGCode:  p.LGCode,
			RepLat:  p.RepLat,
			RepLon:  p.RepLon,
		}
		// Insert normalized key
		key := services.NormalizeForMatching(p.Pref)
		g.prefTrie.Insert(key, info)
	}
	return nil
}

// buildCityTrie builds the city/county prefix tree
func (g *Geocoder) buildCityTrie() error {
	cities, err := g.db.GetAllCities()
	if err != nil {
		return err
	}

	g.cityTrie = geocodemodels.NewTrie[*models.CityMatchingInfo]()
	for _, c := range cities {
		info := &models.CityMatchingInfo{
			LGCode:  c.LGCode,
			PrefKey: c.PrefKey,
			CityKey: c.CityKey,
			Pref:    c.Pref,
			County:  c.County,
			City:    c.City,
			Ward:    c.Ward,
			RepLat:  c.RepLat,
			RepLon:  c.RepLon,
		}

		// Build city key for trie: county + city + ward
		cityStr := c.County + c.City + c.Ward
		if cityStr == "" {
			cityStr = c.City
		}
		key := services.NormalizeForMatching(cityStr)
		if key != "" {
			g.cityTrie.Insert(key, info)
		}
	}
	return nil
}

// buildTownTrie builds the town prefix tree for a specific city
func (g *Geocoder) buildTownTrie(cityKey string) (*geocodemodels.Trie[*models.TownMatchingInfo], error) {
	if t, ok := g.townTries[cityKey]; ok {
		return t, nil
	}

	towns, err := g.db.GetOazaChomes(cityKey)
	if err != nil {
		return nil, err
	}

	trie := geocodemodels.NewTrie[*models.TownMatchingInfo]()
	for _, town := range towns {
		info := &models.TownMatchingInfo{
			LGCode:       town.LGCode,
			PrefKey:      town.PrefKey,
			CityKey:      town.CityKey,
			TownKey:      town.TownKey,
			MachiazaID:   town.MachiazaID,
			Pref:         town.Pref,
			County:       town.County,
			City:         town.City,
			Ward:         town.Ward,
			OazaCho:      town.OazaCho,
			Chome:        town.Chome,
			Koaza:        town.Koaza,
			KoazaAkaCode: town.KoazaAkaCode,
			RsdtAddrFlg:  town.RsdtAddrFlg,
			RepLat:       town.RepLat,
			RepLon:       town.RepLon,
		}

		// Build town key: oaza_cho + chome + koaza
		townStr := town.OazaCho + town.Chome + town.Koaza
		key := services.NormalizeForMatching(townStr)
		if key != "" {
			trie.Insert(key, info)
		}

		// Also insert a "short" key: oaza_cho + chome_number (without 丁目 suffix).
		// Japanese addresses often omit 丁目, writing "清水1-3" instead of "清水1丁目3番".
		if town.Chome != "" {
			chomeNum := stripChomeSuffix(services.NormalizeForMatching(town.Chome))
			if chomeNum != "" {
				shortKey := services.NormalizeForMatching(town.OazaCho) + chomeNum
				if shortKey != key {
					trie.Insert(shortKey, info)
				}
			}
		}
	}

	g.townTries[cityKey] = trie
	return trie, nil
}

// Geocode geocodes a single address string
func (g *Geocoder) Geocode(address string) (*geocodemodels.GeoCodeResult, error) {
	// Normalize input
	normalized := services.NormalizeAddress(address)

	query := geocodemodels.Create(normalized)
	query.Input = address // Keep original input

	// Run through the pipeline
	pipeline := []steps.Step{
		steps.NewPrefStep(g.prefTrie, g.fuzzyChar),
		steps.NewCityStep(g.cityTrie, g.fuzzyChar),
	}

	queries := []*geocodemodels.Query{query}

	for _, step := range pipeline {
		var nextQueries []*geocodemodels.Query
		for _, q := range queries {
			results, err := step.Process(q)
			if err != nil {
				return nil, err
			}
			nextQueries = append(nextQueries, results...)
		}
		if len(nextQueries) > 0 {
			queries = nextQueries
		}
	}

	// Town step requires city key
	var townQueries []*geocodemodels.Query
	for _, q := range queries {
		if q.CityKey != "" {
			townTrie, err := g.buildTownTrie(q.CityKey)
			if err != nil {
				return nil, err
			}

			townStep := steps.NewTownStep(townTrie, g.fuzzyChar)
			results, err := townStep.Process(q)
			if err != nil {
				return nil, err
			}
			townQueries = append(townQueries, results...)
		} else {
			townQueries = append(townQueries, q)
		}
	}
	if len(townQueries) > 0 {
		queries = townQueries
	}

	// Residential block and parcel steps
	var detailQueries []*geocodemodels.Query
	for _, q := range queries {
		var foundDetail bool

		if q.TownKey != "" && q.LGCode != "" {
			// Try residential block matching
			if g.searchTarget != types.SearchTargetParcel {
				rsdtStep := steps.NewRsdtBlkStep(g.dataDir)
				results, err := rsdtStep.Process(q)
				if err != nil {
					return nil, err
				}
				if len(results) > 0 && results[0].MatchLevel > q.MatchLevel {
					detailQueries = append(detailQueries, results...)
					foundDetail = true
				}
			}

			// Try parcel matching
			if g.searchTarget != types.SearchTargetResidential {
				parcelStep := steps.NewParcelStep(g.dataDir)
				results, err := parcelStep.Process(q)
				if err != nil {
					return nil, err
				}
				if len(results) > 0 && results[0].MatchLevel > q.MatchLevel {
					detailQueries = append(detailQueries, results...)
					foundDetail = true
				}
			}
		}

		// If no detail found, keep the original town-level result
		if !foundDetail {
			detailQueries = append(detailQueries, q)
		}
	}
	if len(detailQueries) > 0 {
		queries = detailQueries
	}

	// Score and select the best result
	best := selectBestResult(queries, g.searchTarget)
	return best.ToResult(), nil
}

// selectBestResult chooses the best geocoding result from a set of candidates
func selectBestResult(queries []*geocodemodels.Query, target types.SearchTarget) *geocodemodels.Query {
	if len(queries) == 0 {
		return geocodemodels.Create("")
	}

	var best *geocodemodels.Query
	bestScore := -1.0

	for _, q := range queries {
		score := computeScore(q, target)
		if score > bestScore {
			bestScore = score
			best = q
		}
	}

	if best == nil {
		return queries[0]
	}

	best.Score = bestScore
	return best
}

// computeScore computes a relevance score for a query result
func computeScore(q *geocodemodels.Query, target types.SearchTarget) float64 {
	// Base score from match level
	levelScore := float64(q.MatchLevel) * 10.0

	// Character coverage score
	inputLen := float64(len([]rune(q.Input)))
	if inputLen == 0 {
		return 0
	}
	matched := float64(q.MatchedChar)
	coverageScore := matched / inputLen

	// Penalty for unmatched characters
	unmatchedPenalty := float64(q.UnmatchedChar) * 0.1

	// Similarity score
	similarityBonus := q.SimilarityScore * 5.0

	// Target type bonus
	targetBonus := 0.0
	switch target {
	case types.SearchTargetResidential:
		if q.RsdtAddrFlg == 1 {
			targetBonus = 5.0
		}
	case types.SearchTargetParcel:
		if q.RsdtAddrFlg == 0 {
			targetBonus = 5.0
		}
	}

	total := levelScore + coverageScore*20 - unmatchedPenalty + similarityBonus + targetBonus

	// Normalize to 0-1 range approximately
	return total / 100.0
}

// stripChomeSuffix removes 丁目/ちょうめ from a normalized chome string, returning just the number.
// e.g. "1丁目" → "1", "2ちょうめ" → "2"
func stripChomeSuffix(chome string) string {
	suffixes := []string{"丁目", "ちょうめ", "chome"}
	for _, suf := range suffixes {
		if strings.HasSuffix(chome, suf) {
			return strings.TrimSuffix(chome, suf)
		}
	}
	return chome
}

// GeocodeMany geocodes multiple addresses
func (g *Geocoder) GeocodeMany(addresses []string) ([]*geocodemodels.GeoCodeResult, error) {
	results := make([]*geocodemodels.GeoCodeResult, len(addresses))
	for i, addr := range addresses {
		result, err := g.Geocode(addr)
		if err != nil {
			return nil, fmt.Errorf("failed to geocode address %q: %w", addr, err)
		}
		results[i] = result
	}
	return results, nil
}

// GeocodeStream processes addresses from a channel and sends results to output
func (g *Geocoder) GeocodeStream(input <-chan string, output chan<- *geocodemodels.GeoCodeResult) error {
	for addr := range input {
		addr = strings.TrimSpace(addr)
		if addr == "" || strings.HasPrefix(addr, "#") {
			continue
		}

		result, err := g.Geocode(addr)
		if err != nil {
			return err
		}
		output <- result
	}
	return nil
}
