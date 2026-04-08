// Package models provides trie data structure for efficient address matching.
// Ported from TypeScript trie-based finders.
package models

import "strings"

// TrieNode represents a node in the trie data structure
type TrieNode[T any] struct {
	children map[rune]*TrieNode[T]
	values   []T
	isEnd    bool
}

// Trie is a prefix tree for efficient string matching
type Trie[T any] struct {
	root *TrieNode[T]
}

// NewTrie creates a new empty trie
func NewTrie[T any]() *Trie[T] {
	return &Trie[T]{
		root: &TrieNode[T]{
			children: make(map[rune]*TrieNode[T]),
		},
	}
}

// Insert adds a key-value pair to the trie
func (t *Trie[T]) Insert(key string, value T) {
	node := t.root
	for _, ch := range key {
		if node.children == nil {
			node.children = make(map[rune]*TrieNode[T])
		}
		if _, ok := node.children[ch]; !ok {
			node.children[ch] = &TrieNode[T]{
				children: make(map[rune]*TrieNode[T]),
			}
		}
		node = node.children[ch]
	}
	node.isEnd = true
	node.values = append(node.values, value)
}

// MatchResult holds the result of a trie match
type MatchResult[T any] struct {
	Matched string
	Values  []T
	Rest    string
}

// FindLongest finds the longest matching prefix in the trie
func (t *Trie[T]) FindLongest(input string) *MatchResult[T] {
	node := t.root
	runes := []rune(input)
	lastMatch := -1
	var lastMatchValues []T

	for i, ch := range runes {
		next, ok := node.children[ch]
		if !ok {
			break
		}
		node = next
		if node.isEnd {
			lastMatch = i
			lastMatchValues = node.values
		}
	}

	if lastMatch < 0 {
		return nil
	}

	matched := string(runes[:lastMatch+1])
	rest := string(runes[lastMatch+1:])
	return &MatchResult[T]{
		Matched: matched,
		Values:  lastMatchValues,
		Rest:    rest,
	}
}

// FindAll finds all prefix matches in the trie (partial matches)
func (t *Trie[T]) FindAll(input string) []*MatchResult[T] {
	var results []*MatchResult[T]
	node := t.root
	runes := []rune(input)

	for i, ch := range runes {
		next, ok := node.children[ch]
		if !ok {
			break
		}
		node = next
		if node.isEnd {
			matched := string(runes[:i+1])
			rest := string(runes[i+1:])
			results = append(results, &MatchResult[T]{
				Matched: matched,
				Values:  node.values,
				Rest:    rest,
			})
		}
	}

	return results
}

// FuzzyMatchResult holds a fuzzy match result with score
type FuzzyMatchResult[T any] struct {
	MatchResult[T]
	ExtraChars []rune // chars skipped during fuzzy match
}

// FindWithFuzzy finds matches allowing one fuzzy character substitution
func (t *Trie[T]) FindWithFuzzy(input string, fuzzyChar rune) []*FuzzyMatchResult[T] {
	var results []*FuzzyMatchResult[T]

	runes := []rune(input)
	// Try exact matches first
	for _, m := range t.FindAll(input) {
		results = append(results, &FuzzyMatchResult[T]{MatchResult: *m})
	}

	// Try fuzzy: replace each character with fuzzyChar and try matching
	for i, ch := range runes {
		if ch == fuzzyChar {
			// Already a fuzzy char, try matching rest
			modified := append([]rune{}, runes[:i]...)
			modified = append(modified, runes[i+1:]...)
			for _, m := range t.FindAll(string(modified)) {
				results = append(results, &FuzzyMatchResult[T]{
					MatchResult: *m,
					ExtraChars:  []rune{ch},
				})
			}
		}
	}

	return results
}

// NormalizeKey normalizes an address key for trie lookup
// This handles common variations in Japanese addresses
func NormalizeKey(key string) string {
	// Replace various dash characters with standard dash
	key = strings.ReplaceAll(key, "－", "-")
	key = strings.ReplaceAll(key, "ー", "-")
	key = strings.ReplaceAll(key, "‐", "-")
	return key
}
