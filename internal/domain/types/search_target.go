// Ported from TypeScript: src/domain/types/search-target.ts
package types

// SearchTarget defines what type of address to search for
type SearchTarget string

const (
	SearchTargetAll         SearchTarget = "all"
	SearchTargetResidential SearchTarget = "residential"
	SearchTargetParcel      SearchTarget = "parcel"
)

// IsValid returns true if the search target is valid
func (t SearchTarget) IsValid() bool {
	switch t {
	case SearchTargetAll, SearchTargetResidential, SearchTargetParcel:
		return true
	}
	return false
}
