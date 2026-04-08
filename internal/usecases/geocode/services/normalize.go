// Package services provides text processing utilities for geocoding.
// Ported from TypeScript: src/usecases/geocode/services/ and steps/normalize-transform.ts
package services

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// NormalizeAddress normalizes a Japanese address string for geocoding
// This is the main normalization pipeline
func NormalizeAddress(address string) string {
	// Remove leading/trailing whitespace
	address = strings.TrimSpace(address)

	// Normalize Unicode to NFC form
	address = norm.NFC.String(address)

	// Remove comments (content in parentheses)
	address = removeParenthetical(address)

	// Convert full-width alphanumeric to half-width
	address = toHankakuAlphaNum(address)

	// Convert katakana to hiragana (for matching)
	// Note: kept as-is in output, only normalized for matching

	// Consolidate whitespace
	address = consolidateWhitespace(address)

	// Normalize various dash characters to standard dash
	address = normalizeDashes(address)

	// Normalize kanji numbers
	address = Kan2Num(address)

	return address
}

// toHankakuAlphaNum converts full-width alphanumeric characters to half-width
func toHankakuAlphaNum(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'Ａ' && r <= 'Ｚ' {
			b.WriteRune(r - 'Ａ' + 'A')
		} else if r >= 'ａ' && r <= 'ｚ' {
			b.WriteRune(r - 'ａ' + 'a')
		} else if r >= '０' && r <= '９' {
			b.WriteRune(r - '０' + '0')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ToHiragana converts katakana characters to hiragana
func ToHiragana(s string) string {
	var b strings.Builder
	for _, r := range s {
		// Full-width katakana range: U+30A1 to U+30F6
		if r >= 'ァ' && r <= 'ヶ' {
			// Convert to hiragana: subtract 0x60 (96)
			b.WriteRune(r - 0x60)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ToKatakana converts hiragana characters to katakana
func ToKatakana(s string) string {
	var b strings.Builder
	for _, r := range s {
		// Hiragana range: U+3041 to U+3096
		if r >= 'ぁ' && r <= 'ゖ' {
			b.WriteRune(r + 0x60)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normalizeDashes replaces various dash-like characters with a standard hyphen
func normalizeDashes(s string) string {
	dashChars := []rune{'－', 'ー', '‐', '‑', '‒', '–', '—', '―', '⁻', '₋', '−', '˗', '⁃', '·', '･', '・', '\u30FB'}
	var b strings.Builder
	for _, r := range s {
		isDash := false
		for _, d := range dashChars {
			if r == d {
				isDash = true
				break
			}
		}
		if isDash {
			b.WriteRune('-')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// consolidateWhitespace replaces multiple whitespace with single space
func consolidateWhitespace(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) || r == '　' { // include full-width space
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return b.String()
}

// removeParenthetical removes content in parentheses
var parenthesisRe = regexp.MustCompile(`[（(][^）)]*[）)]`)

func removeParenthetical(s string) string {
	return parenthesisRe.ReplaceAllString(s, "")
}

// NormalizeForMatching prepares an address for trie matching.
// Both trie keys and input queries must go through this before comparison.
func NormalizeForMatching(address string) string {
	// Full-width alphanumeric → half-width
	address = toHankakuAlphaNum(address)
	// Katakana → hiragana
	address = ToHiragana(address)
	// Remove spaces
	address = strings.ReplaceAll(address, " ", "")
	address = strings.ReplaceAll(address, "　", "")
	return address
}

// TrimDashAndSpace trims leading dashes and spaces from a string
func TrimDashAndSpace(s string) string {
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == '-' || r == '−' || r == '－' || unicode.IsSpace(r) {
			s = s[size:]
		} else {
			break
		}
	}
	return s
}

// IsNumber checks if a string represents a number
func IsNumber(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// LevenshteinDistance computes the Levenshtein distance between two strings
func LevenshteinDistance(a, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	la, lb := len(ra), len(rb)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Create a 2D slice
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
		dp[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			dp[i][j] = min3(dp[i-1][j]+1, dp[i][j-1]+1, dp[i-1][j-1]+cost)
		}
	}

	return dp[la][lb]
}

// LevenshteinRatio computes the similarity ratio (0.0 to 1.0)
func LevenshteinRatio(a, b string) float64 {
	if a == "" && b == "" {
		return 1.0
	}
	maxLen := len([]rune(a))
	if lb := len([]rune(b)); lb > maxLen {
		maxLen = lb
	}
	if maxLen == 0 {
		return 1.0
	}
	dist := LevenshteinDistance(a, b)
	return 1.0 - float64(dist)/float64(maxLen)
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// InsertSpaceBeforeRoomOrFacility adds a space before room/facility numbers
// e.g., "東京都渋谷区1-2-3101" -> "東京都渋谷区1-2-3 101"
var roomRe = regexp.MustCompile(`(\d)([ぁ-ん]|号室|部屋)`)

func InsertSpaceBeforeRoomOrFacility(s string) string {
	return roomRe.ReplaceAllString(s, "$1 $2")
}

// NormalizeJisKanji performs JIS kanji normalization (old forms to new forms)
// This maps commonly used variant kanji to their standard forms
func NormalizeJisKanji(s string) string {
	var b strings.Builder
	for _, r := range s {
		if normalized, ok := jisKanjiMap[r]; ok {
			b.WriteRune(normalized)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// jisKanjiMap maps JIS Level 2 kanji variants to Level 1 standard forms
// This is a subset of the full normalization table from jis-kanji.ts
var jisKanjiMap = map[rune]rune{
	// Common kanji variants in Japanese addresses
	'亙': '亘',
	'亞': '亜',
	'亰': '京',
	'亳': '亳',
	'仔': '仔',
	'伊': '伊',
	'來': '来',
	'俤': '俤',
	'偉': '偉',
	'傭': '傭',
	'僊': '仙',
	'僞': '偽',
	'兒': '児',
	'兩': '両',
	'冨': '富',
	'冱': '凍',
	'凜': '凛',
	'凞': '熙',
	'刕': '刕',
	'劒': '剣',
	'劔': '剣',
	'勢': '勢',
	'勸': '勧',
	'勹': '勹',
	'匤': '匤',
	'卽': '即',
	'厰': '廠',
	'叢': '叢',
	'吞': '呑',
	'喆': '喆',
	'嚙': '噛',
	'囎': '囎',
	'囿': '囿',
	'圀': '国',
	'圄': '圄',
	'塚': '塚',
	'墻': '墻',
	'壽': '寿',
	'夘': '卯',
	'姸': '妍',
	'媛': '媛',
	'嫺': '嫺',
	'孃': '嬢',
	'學': '学',
	'寬': '寛',
	'寳': '宝',
	'寵': '寵',
	'峯': '峰',
	'崎': '崎',
	'嶋': '島',
	'嶌': '島',
	'廣': '広',
	'廰': '廳',
	'彌': '弥',
	'悳': '悳',
	'惠': '恵',
	'戲': '戯',
	'拔': '抜',
	'搜': '捜',
	'攜': '携',
	'晚': '晩',
	'暑': '暑',
	'曆': '暦',
	'樂': '楽',
	'樋': '樋',
	'橋': '橋',
	'歸': '帰',
	'歷': '歴',
	'溫': '温',
	'漢': '漢',
	'澤': '沢',
	'濕': '湿',
	'瀨': '瀬',
	'炙': '炙',
	'熈': '熙',
	'燈': '灯',
	'爲': '為',
	'獸': '獣',
	'琢': '琢',
	'瑩': '瑩',
	'瓮': '瓮',
	'甕': '甕',
	'畠': '畠',
	'疉': '畳',
	'眞': '真',
	'祥': '祥',
	'禎': '禎',
	'穗': '穂',
	'竹': '竹',
	'籠': '籠',
	'縣': '県',
	'繁': '繁',
	'纊': '纊',
	'繪': '絵',
	'羣': '群',
	'羽': '羽',
	'翠': '翠',
	'聰': '聡',
	'聲': '声',
	'聽': '聴',
	'脛': '脛',
	'脣': '唇',
	'臺': '台',
	'舘': '館',
	'藏': '蔵',
	'藤': '藤',
	'蠖': '蠖',
	'蟬': '蝉',
	'衞': '衛',
	'裝': '装',
	'褒': '褒',
	'覺': '覚',
	'覽': '覧',
	'諸': '諸',
	'謙': '謙',
	'讓': '譲',
	'豐': '豊',
	'賴': '頼',
	'踊': '踊',
	'蹤': '蹤',
	'輕': '軽',
	'轉': '転',
	'辯': '弁',
	'邊': '辺',
	'邨': '村',
	'郞': '郎',
	'醫': '医',
	'醬': '醤',
	'醱': '醗',
	'鈩': '鈩',
	'鎭': '鎮',
	'鑄': '鋳',
	'間': '間',
	'關': '関',
	'隆': '隆',
	'隨': '随',
	'險': '険',
	'雄': '雄',
	'雲': '雲',
	'靑': '青',
	'頰': '頬',
	'顯': '顕',
	'飮': '飲',
	'餘': '余',
	'鬆': '鬆',
	'鷗': '鴎',
	'麴': '麹',
	'黑': '黒',
	'齊': '斉',
}
