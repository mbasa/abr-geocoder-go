// Package steps provides the geocoding pipeline transformation steps.
// Ported from TypeScript: src/usecases/geocode/steps/
package steps

import (
	geocodemodels "github.com/mbasa/abr-geocoder-go/internal/usecases/geocode/models"
)

// Step represents a single step in the geocoding pipeline
type Step interface {
	Process(q *geocodemodels.Query) ([]*geocodemodels.Query, error)
}
