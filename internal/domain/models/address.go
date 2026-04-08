// Package models defines core address data models for abr-geocoder.
package models

// PrefRow represents a prefecture record in the database
type PrefRow struct {
	LGCode  string  `db:"lg_code"`
	PrefKey string  `db:"pref_key"`
	Pref    string  `db:"pref"`
	KanaPref string `db:"kana_pref"`
	RomaPref string `db:"roma_pref"`
	RepLat  float64 `db:"rep_lat"`
	RepLon  float64 `db:"rep_lon"`
}

// CityRow represents a city/county record in the database
type CityRow struct {
	LGCode   string  `db:"lg_code"`
	PrefKey  string  `db:"pref_key"`
	CityKey  string  `db:"city_key"`
	Pref     string  `db:"pref"`
	County   string  `db:"county"`
	City     string  `db:"city"`
	Ward     string  `db:"ward"`
	KanaCity string  `db:"kana_city"`
	RomaCity string  `db:"roma_city"`
	RepLat   float64 `db:"rep_lat"`
	RepLon   float64 `db:"rep_lon"`
}

// TownRow represents a town/machiaza record in the database
type TownRow struct {
	LGCode      string  `db:"lg_code"`
	PrefKey     string  `db:"pref_key"`
	CityKey     string  `db:"city_key"`
	TownKey     string  `db:"town_key"`
	MachiazaID  string  `db:"machiaza_id"`
	Pref        string  `db:"pref"`
	County      string  `db:"county"`
	City        string  `db:"city"`
	Ward        string  `db:"ward"`
	OazaCho     string  `db:"oaza_cho"`
	Chome       string  `db:"chome"`
	Koaza       string  `db:"koaza"`
	KoazaAkaCode int    `db:"koaza_aka_code"`
	RsdtAddrFlg int     `db:"rsdt_addr_flg"`
	RepLat      float64 `db:"rep_lat"`
	RepLon      float64 `db:"rep_lon"`
}

// RsdtBlkRow represents a residential block record
type RsdtBlkRow struct {
	LGCode   string  `db:"lg_code"`
	TownKey  string  `db:"town_key"`
	BlkKey   string  `db:"blk_key"`
	BlkNum   string  `db:"blk_num"`
	RepLat   float64 `db:"rep_lat"`
	RepLon   float64 `db:"rep_lon"`
}

// RsdtDspRow represents a residential display record
type RsdtDspRow struct {
	LGCode   string  `db:"lg_code"`
	TownKey  string  `db:"town_key"`
	BlkKey   string  `db:"blk_key"`
	RsdtKey  string  `db:"rsdt_key"`
	RsdtID   string  `db:"rsdt_id"`
	RsdtID2  string  `db:"rsdt_id2"`
	RepLat   float64 `db:"rep_lat"`
	RepLon   float64 `db:"rep_lon"`
}

// ParcelRow represents a land parcel record
type ParcelRow struct {
	LGCode    string  `db:"lg_code"`
	TownKey   string  `db:"town_key"`
	ParcelKey string  `db:"parcel_key"`
	PrcNum1   string  `db:"prc_num1"`
	PrcNum2   string  `db:"prc_num2"`
	PrcNum3   string  `db:"prc_num3"`
	RepLat    float64 `db:"rep_lat"`
	RepLon    float64 `db:"rep_lon"`
}

// PrefInfo holds prefecture matching data for trie lookups
type PrefInfo struct {
	PrefKey string
	Pref    string
	LGCode  string
	RepLat  float64
	RepLon  float64
}

// CityMatchingInfo holds city/county matching data for trie lookups
type CityMatchingInfo struct {
	LGCode  string
	PrefKey string
	CityKey string
	Pref    string
	County  string
	City    string
	Ward    string
	RepLat  float64
	RepLon  float64
}

// TownMatchingInfo holds town matching data for trie lookups
type TownMatchingInfo struct {
	LGCode      string
	PrefKey     string
	CityKey     string
	TownKey     string
	MachiazaID  string
	Pref        string
	County      string
	City        string
	Ward        string
	OazaCho     string
	Chome       string
	Koaza       string
	KoazaAkaCode int
	RsdtAddrFlg  int
	RepLat      float64
	RepLon      float64
}

// UrlCache represents a cached URL response metadata
type UrlCache struct {
	UrlKey        string
	URL           string
	Etag          string
	LastModified  string
	ContentLength int64
	Crc32         uint32
}
