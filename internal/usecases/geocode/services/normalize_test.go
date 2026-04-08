package services

import (
	"testing"
)

func TestToHiragana(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"アイウエオ", "あいうえお"},
		{"カキクケコ", "かきくけこ"},
		{"東京都", "東京都"}, // Kanji unchanged
		{"ABC", "ABC"},      // Latin unchanged
		{"あいう", "あいう"}, // Hiragana unchanged
	}

	for _, tt := range tests {
		got := ToHiragana(tt.input)
		if got != tt.expected {
			t.Errorf("ToHiragana(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestToHankakuAlphaNum(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"１２３", "123"},
		{"ＡＢＣ", "ABC"},
		{"ａｂｃ", "abc"},
		{"東京都", "東京都"},
	}

	for _, tt := range tests {
		got := toHankakuAlphaNum(tt.input)
		if got != tt.expected {
			t.Errorf("toHankakuAlphaNum(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNormalizeDashes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1－2", "1-2"},
		{"1ー2", "1-2"},
		{"1‐2", "1-2"},
		{"東京都", "東京都"},
	}

	for _, tt := range tests {
		got := normalizeDashes(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeDashes(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"東京都", "東京府", 1},
	}

	for _, tt := range tests {
		got := LevenshteinDistance(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestNormalizeAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"東京都千代田区１丁目１番地", "東京都千代田区1丁目1番地"},
		{"  東京都  ", "東京都"},
		{"東京都（千代田区）", "東京都"},
	}

	for _, tt := range tests {
		got := NormalizeAddress(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeAddress(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
