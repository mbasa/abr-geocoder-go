// Package config provides configuration constants for abr-geocoder.
// Ported from TypeScript: src/config/constant-values.ts
package config

// DashSymbols contains dash-like characters used in Japanese addresses
const DashSymbols = "－ー‐‑‒–—―⁻₋−˗⁃·･・\u30FB"

// NumericSymbols contains numeric characters (full-width + half-width)
const NumericSymbols = "０１２３４５６７８９0123456789"

// KanjiNums contains kanji numerals used in addresses
const KanjiNums = "〇一二三四五六七八九十百千"

// AlphaNumericSymbols contains full-width alphanumeric characters
const AlphaNumericSymbols = "ａｂｃｄｅｆｇｈｉｊｋｌｍｎｏｐｑｒｓｔｕｖｗｘｙｚＡＢＣＤＥＦＧＨＩＪＫＬＭＮＯＰＱＲＳＴＵＶＷＸＹＺ０１２３４５６７８９"

// Replacement tokens used in address normalization
const (
	SpaceToken    = "␣"   // Space placeholder
	DashToken     = "@"   // Dash placeholder
	BangaichiToken = "<BG>" // 番外地 (outside numbered area)
	MubanchiToken  = "<MB>" // 無番地 (unnumbered land)
	OazaBanchoToken = "<OB>" // 大字番町
	OazaCenterToken = "<OC>" // 大字センター
)

// MaxConcurrentDownload is the maximum number of concurrent downloads
const MaxConcurrentDownload = 100

// CLIServerPort is the default port for the CLI server
const CLIServerPort = 8143

// DefaultFuzzyChar is the default character for fuzzy matching
const DefaultFuzzyChar = "?"

// AmbiguousRsdtAddrFlg is the flag value for ambiguous residential addresses
const AmbiguousRsdtAddrFlg = -1

// DatasetAPIHostname is the hostname for the Digital Agency dataset API
const DatasetAPIHostname = "https://dataset.address-br.digital.go.jp"

// AppVersion is the current version of the application
const AppVersion = "2.2.1"

// CacheDirName is the default cache directory name
const CacheDirName = ".abr-geocoder"
