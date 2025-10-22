package main

import (
	"strings"
	"testing"
)

func TestParseColorMapping_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:  "single mapping",
			input: "accent1:accent3",
			expected: map[string]string{
				"accent1": "accent3",
			},
		},
		{
			name:  "multiple mappings",
			input: "accent1:accent3,accent5:accent6",
			expected: map[string]string{
				"accent1": "accent3",
				"accent5": "accent6",
			},
		},
		{
			name:  "many-to-one mapping",
			input: "accent1:accent3,accent5:accent3",
			expected: map[string]string{
				"accent1": "accent3",
				"accent5": "accent3",
			},
		},
		{
			name:  "all color types",
			input: "dk1:dk2,lt1:lt2,accent1:accent2,hlink:folHlink",
			expected: map[string]string{
				"dk1":     "dk2",
				"lt1":     "lt2",
				"accent1": "accent2",
				"hlink":   "folHlink",
			},
		},
		{
			name:  "with whitespace",
			input: " accent1 : accent3 , accent5 : accent6 ",
			expected: map[string]string{
				"accent1": "accent3",
				"accent5": "accent6",
			},
		},
		{
			name:  "duplicate identical mapping",
			input: "accent1:accent3,accent1:accent3",
			expected: map[string]string{
				"accent1": "accent3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseColorMapping(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d mappings, got %d", len(tt.expected), len(result))
			}

			for source, expectedTarget := range tt.expected {
				if target, exists := result[source]; !exists {
					t.Errorf("missing mapping for %s", source)
				} else if target != expectedTarget {
					t.Errorf("expected %s:%s, got %s:%s", source, expectedTarget, source, target)
				}
			}
		})
	}
}

func TestParseColorMapping_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name:        "empty string",
			input:       "",
			errContains: "cannot be empty",
		},
		{
			name:        "whitespace only",
			input:       "   ",
			errContains: "cannot be empty",
		},
		{
			name:        "missing colon",
			input:       "accent1accent3",
			errContains: "invalid mapping format",
		},
		{
			name:        "multiple colons",
			input:       "accent1:accent3:accent5",
			errContains: "exactly one ':'",
		},
		{
			name:        "empty source",
			input:       ":accent3",
			errContains: "cannot be empty",
		},
		{
			name:        "empty target",
			input:       "accent1:",
			errContains: "cannot be empty",
		},
		{
			name:        "invalid source color",
			input:       "invalidcolor:accent3",
			errContains: "invalid source color",
		},
		{
			name:        "invalid target color",
			input:       "accent1:invalidcolor",
			errContains: "invalid target color",
		},
		{
			name:        "conflicting mappings",
			input:       "accent1:accent3,accent1:accent2",
			errContains: "conflicting mappings",
		},
		{
			name:        "only commas",
			input:       ",,,",
			errContains: "no valid mappings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseColorMapping(tt.input)
			if err == nil {
				t.Fatalf("expected error containing '%s', got nil", tt.errContains)
			}

			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
				t.Errorf("expected error containing '%s', got: %v", tt.errContains, err)
			}
		})
	}
}

func TestParseColorMapping_AtomicReplacement(t *testing.T) {
	// This mapping tests that accent1→accent3 and accent3→accent4
	// Both should exist independently (atomic replacement, no cascading)
	mapping, err := ParseColorMapping("accent1:accent3,accent3:accent4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mapping) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(mapping))
	}

	if mapping["accent1"] != "accent3" {
		t.Errorf("expected accent1→accent3, got accent1→%s", mapping["accent1"])
	}

	if mapping["accent3"] != "accent4" {
		t.Errorf("expected accent3→accent4, got accent3→%s", mapping["accent3"])
	}
}
