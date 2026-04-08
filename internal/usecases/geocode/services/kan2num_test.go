package services

import (
	"testing"
)

func TestKan2Num(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"三田二丁目", "三田2丁目"},
		{"一丁目", "1丁目"},
		{"十二番地", "12番地"},
		{"二十三番", "23番"},
		{"三百番地", "300番地"},
		{"東京都", "東京都"},       // No kanji numbers
	}

	for _, tt := range tests {
		got := Kan2Num(tt.input)
		if got != tt.expected {
			t.Errorf("Kan2Num(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestIsKanjiNums(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"一二三", true},
		{"123", true},
		{"一2三", true},
		{"東京都", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsKanjiNums(tt.input)
		if got != tt.expected {
			t.Errorf("IsKanjiNums(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
