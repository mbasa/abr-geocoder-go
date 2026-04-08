// Package services provides kanji-to-number conversion.
// Ported from TypeScript: src/usecases/geocode/services/kan2num.ts
package services

import (
	"strings"
	"unicode"
)

// kanjiDigitMap maps kanji numeral characters to their integer values
var kanjiDigitMap = map[rune]int{
	'〇': 0,
	'零': 0,
	'一': 1,
	'壱': 1,
	'壹': 1,
	'１': 1,
	'二': 2,
	'弐': 2,
	'貳': 2,
	'２': 2,
	'三': 3,
	'参': 3,
	'參': 3,
	'３': 3,
	'四': 4,
	'４': 4,
	'五': 5,
	'５': 5,
	'六': 6,
	'６': 6,
	'七': 7,
	'７': 7,
	'八': 8,
	'８': 8,
	'九': 9,
	'９': 9,
	'十': 10,
	'拾': 10,
	'百': 100,
	'千': 1000,
}

// targetChars are the characters that trigger kanji number conversion
// e.g., 三田二丁目 -> 三田2丁目
var targetChars = map[rune]bool{
	'軒': true,
	'丁': true,
	'番': true,
	'条': true,
	'線': true,
	'号': true,
	'地': true,
	'の': true,
}

// Kan2Num converts kanji numerals in an address to Arabic numerals
// Example: "東京都港区三田二丁目２番１８号" -> "東京都港区三田2丁目2番18号"
func Kan2Num(address string) string {
	runes := []rune(address)
	result := make([]rune, 0, len(runes))

	i := 0
	for i < len(runes) {
		r := runes[i]

		// Check if this is a kanji number position
		if isKanjiNum(r) {
			// Look ahead to see if this is followed by a target character
			num, end := parseKanjiNumber(runes, i)
			if end > i {
				// Check if the character after the number is a target char
				if end < len(runes) && isTargetChar(runes[end]) {
					// Convert to Arabic numeral
					numStr := []rune(intToString(num))
					result = append(result, numStr...)
					i = end
					continue
				}
			}
		}

		result = append(result, r)
		i++
	}

	return string(result)
}

// isKanjiNum returns true if the rune is a kanji numeral
func isKanjiNum(r rune) bool {
	_, ok := kanjiDigitMap[r]
	return ok
}

// isTargetChar returns true if the rune is a target character
func isTargetChar(r rune) bool {
	return targetChars[r]
}

// parseKanjiNumber parses a kanji number starting at index i
// Returns the numeric value and the index after the number
func parseKanjiNumber(runes []rune, start int) (int, int) {
	i := start
	total := 0
	current := 0
	hasValue := false

	for i < len(runes) {
		r := runes[i]
		val, ok := kanjiDigitMap[r]
		if !ok {
			break
		}

		hasValue = true

		switch {
		case val == 1000:
			if current == 0 {
				current = 1
			}
			total += current * 1000
			current = 0
		case val == 100:
			if current == 0 {
				current = 1
			}
			total += current * 100
			current = 0
		case val == 10:
			if current == 0 {
				current = 1
			}
			total += current * 10
			current = 0
		default:
			if current > 0 {
				// This means we have something like 十二 -> 12
				total += current
				current = val
			} else {
				current = val
			}
		}
		i++
	}

	if hasValue {
		total += current
		return total, i
	}
	return 0, start
}

// intToString converts an integer to its string representation
func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

// IsKanjiNums checks if a string consists entirely of kanji numerals
func IsKanjiNums(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !isKanjiNum(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// NormalizeNumberChars normalizes number representations
// Converts various numeric representations to standard Arabic numerals
func NormalizeNumberChars(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '０' && r <= '９':
			b.WriteRune(r - '０' + '0')
		default:
			b.WriteRune(r)
		}
	}
	return Kan2Num(b.String())
}
