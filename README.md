# ABR Geocoder (Go)

A Go port of the [ABR Geocoder](https://github.com/digital-go-jp/abr-geocoder) by Japan's Digital Agency (デジタル庁) that normalizes Japanese domestic addresses using the Address Base Registry (ABR).

## Overview

This tool geocodes Japanese addresses by:
- Normalizing address strings (kanji numbers, full-width characters, etc.)
- Looking up hierarchical address components (prefecture → city → town → block/parcel)
- Returning geographic coordinates (latitude/longitude) and structured address data

## Installation

```bash
go install github.com/mbasa/abr-geocoder-go/cmd/abrg@latest
```

Or build from source:

```bash
git clone https://github.com/mbasa/abr-geocoder-go
cd abr-geocoder-go
go build -o abrg ./cmd/abrg
```

## Requirements

- Go 1.20 or later
- Dataset downloaded from Japan's Digital Agency (see `abrg download`)

## Usage

### 1. Download the dataset

```bash
# Download data for Tokyo's Chiyoda Ward (LG code 131016)
abrg download --lgCode 131016

# Download multiple municipalities
abrg download --lgCode 131016 --lgCode 131024

# Specify custom data directory
abrg download --lgCode 131016 --abrgDir /path/to/data
```

### 2. Geocode addresses

```bash
# Geocode from a file
abrg geocode input.txt

# Geocode from stdin
echo "東京都千代田区1-1" | abrg geocode -

# Specify output format
abrg geocode input.txt --format csv
abrg geocode input.txt --format geojson
abrg geocode input.txt -o output.json

# Control what to search for
abrg geocode input.txt --target residential
abrg geocode input.txt --target parcel
abrg geocode input.txt --target all
```

### 3. Start the API server

```bash
# Start server on default port 8143
abrg serve

# Start on custom port
abrg serve --port 8080
```

API endpoint:
```
GET /geocode?address=東京都千代田区1-1&format=json
```

## Output Formats

| Format | Description |
|--------|-------------|
| `json` | JSON array (default) |
| `csv` | CSV with all fields |
| `geojson` | GeoJSON FeatureCollection |
| `ndjson` | Newline-delimited JSON |
| `ndgeojson` | Newline-delimited GeoJSON |
| `simplified` | Minimal CSV (input, output, score, match_level) |

## Output Fields

```json
{
  "input": "東京都千代田区1-1",
  "output": "東京都千代田区霞が関一丁目1番",
  "score": 0.85,
  "match_level": "residential_block",
  "lat": 35.675888,
  "lon": 139.744408,
  "pref": "東京都",
  "city": "千代田区",
  "oaza_cho": "霞が関",
  "chome": "一丁目",
  "blk_num": "1",
  "lg_code": "131016",
  "rsdt_addr_flg": 1
}
```

### Match Levels

| Level | Description |
|-------|-------------|
| `unmatch` | No match found |
| `prefecture` | Matched to prefecture level |
| `city` | Matched to city/ward level |
| `town` | Matched to town/oaza level |
| `residential_block` | Matched to block number |
| `residential_detail` | Matched to residential display address |
| `parcel` | Matched to land parcel |

## Architecture

```
abr-geocoder-go/
├── cmd/abrg/          # CLI entry point
├── internal/
│   ├── config/        # Constants and configuration
│   ├── domain/
│   │   ├── models/    # Data structures (PrefRow, CityRow, TownRow, etc.)
│   │   └── types/     # Core types (OutputFormat, SearchTarget, MatchLevel)
│   ├── drivers/
│   │   └── database/  # SQLite3 database access layer
│   ├── interface/
│   │   ├── cli/       # CLI commands (geocode, download, serve, update-check)
│   │   ├── format/    # Output formatters (JSON, CSV, GeoJSON, etc.)
│   │   └── server/    # REST API server
│   └── usecases/
│       ├── download/  # Dataset download logic
│       └── geocode/   # Core geocoding engine
│           ├── models/   # Query model and Trie data structure
│           ├── services/ # Text normalization (kan2num, normalize, etc.)
│           └── steps/    # Pipeline steps (pref, city, town, rsdt_blk, parcel)
```

## Geocoding Pipeline

The geocoding pipeline processes addresses through these stages:

1. **Normalize** — Converts full-width chars, kanji numbers, katakana→hiragana
2. **Prefecture** — Trie-based prefix match (e.g., "東京都")
3. **City/County** — Trie-based match (e.g., "千代田区")
4. **Town/Oaza** — Trie-based match (e.g., "霞が関一丁目")
5. **Residential Block** — SQLite LIKE query (e.g., block "1")
6. **Parcel** — SQLite query for land parcel numbers
7. **Score & Select** — Picks the best candidate result

## Data Sources

Dataset is provided by Japan's Digital Agency:
- Address Base Registry: https://www.digital.go.jp/policies/base_registry_address

## Differences from the TypeScript Version

- **No worker threads**: Go's goroutines provide concurrency natively
- **No streaming transforms**: Uses synchronous processing with channels for batch mode
- **Pure Go SQLite**: Uses `modernc.org/sqlite` (no CGO required)
- **Simplified trie**: Generic trie implementation vs. custom binary format

## License

MIT License — © 2024 デジタル庁 (Digital Agency of Japan)
