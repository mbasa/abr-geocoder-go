// Ported from TypeScript address match level types
package types

// MatchLevel represents how precisely an address was matched
type MatchLevel int

const (
	MatchLevelUnmatch          MatchLevel = 0
	MatchLevelPrefecture       MatchLevel = 1
	MatchLevelCity             MatchLevel = 2
	MatchLevelTown             MatchLevel = 3
	MatchLevelResidentialBlock MatchLevel = 4
	MatchLevelResidentialDetail MatchLevel = 5
	MatchLevelParcel           MatchLevel = 6
)

// String returns a human-readable name for the match level
func (m MatchLevel) String() string {
	switch m {
	case MatchLevelUnmatch:
		return "unmatch"
	case MatchLevelPrefecture:
		return "prefecture"
	case MatchLevelCity:
		return "city"
	case MatchLevelTown:
		return "town"
	case MatchLevelResidentialBlock:
		return "residential_block"
	case MatchLevelResidentialDetail:
		return "residential_detail"
	case MatchLevelParcel:
		return "parcel"
	}
	return "unknown"
}

// CoordinateLevel represents the precision of coordinates
type CoordinateLevel int

const (
	CoordinateLevelUnknown    CoordinateLevel = 0
	CoordinateLevelPrefecture CoordinateLevel = 1
	CoordinateLevelCity       CoordinateLevel = 2
	CoordinateLevelTown       CoordinateLevel = 3
	CoordinateLevelBlock      CoordinateLevel = 8
)
