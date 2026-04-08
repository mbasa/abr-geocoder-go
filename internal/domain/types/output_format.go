// Package types defines core domain types for abr-geocoder.
// Ported from TypeScript: src/domain/types/output-format.ts
package types

// OutputFormat defines the supported output formats
type OutputFormat string

const (
	OutputFormatJSON       OutputFormat = "json"
	OutputFormatCSV        OutputFormat = "csv"
	OutputFormatGeoJSON    OutputFormat = "geojson"
	OutputFormatNDJSON     OutputFormat = "ndjson"
	OutputFormatNDGeoJSON  OutputFormat = "ndgeojson"
	OutputFormatSimplified OutputFormat = "simplified"
)

// IsValid returns true if the output format is supported
func (f OutputFormat) IsValid() bool {
	switch f {
	case OutputFormatJSON, OutputFormatCSV, OutputFormatGeoJSON,
		OutputFormatNDJSON, OutputFormatNDGeoJSON, OutputFormatSimplified:
		return true
	}
	return false
}

// MimeType returns the MIME type for the output format
func (f OutputFormat) MimeType() string {
	switch f {
	case OutputFormatJSON, OutputFormatNDJSON:
		return "application/json"
	case OutputFormatGeoJSON, OutputFormatNDGeoJSON:
		return "application/geo+json"
	case OutputFormatCSV, OutputFormatSimplified:
		return "text/csv"
	}
	return "application/json"
}
