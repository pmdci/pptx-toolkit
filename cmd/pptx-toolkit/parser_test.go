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

func TestIsValidHexColor(t *testing.T) {
	tests := []struct {
		name     string
		color    string
		expected bool
	}{
		{"valid uppercase", "AABBCC", true},
		{"valid lowercase", "aabbcc", true},
		{"valid mixed case", "AaBbCc", true},
		{"valid with numbers", "FF00AA", true},
		{"valid all numbers", "123456", true},
		{"invalid too short", "ABC", false},
		{"invalid too long", "AABBCCD", false},
		{"invalid characters", "GGHHII", false},
		{"invalid with hash", "#AABBCC", false},
		{"empty string", "", false},
		{"with spaces", "AA BB CC", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidHexColor(tt.color)
			if result != tt.expected {
				t.Errorf("isValidHexColor(%q) = %v, expected %v", tt.color, result, tt.expected)
			}
		})
	}
}

func TestIsValidColor(t *testing.T) {
	tests := []struct {
		name     string
		color    string
		expected bool
	}{
		// Scheme colors
		{"valid scheme accent1", "accent1", true},
		{"valid scheme dk1", "dk1", true},
		{"valid scheme folHlink", "folHlink", true},
		{"invalid scheme", "accent7", false},

		// Hex colors
		{"valid hex uppercase", "AABBCC", true},
		{"valid hex lowercase", "aabbcc", true},
		{"valid hex mixed", "AaBbCc", true},
		{"invalid hex", "GGHHII", false},

		// Edge cases
		{"empty", "", false},
		{"invalid mixed", "accent1X", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidColor(tt.color)
			if result != tt.expected {
				t.Errorf("isValidColor(%q) = %v, expected %v", tt.color, result, tt.expected)
			}
		})
	}
}

func TestParseColorMapping_HexColors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:  "hex to hex uppercase",
			input: "AABBCC:FF0000",
			expected: map[string]string{
				"AABBCC": "FF0000",
			},
		},
		{
			name:  "hex to hex lowercase",
			input: "aabbcc:ff0000",
			expected: map[string]string{
				"aabbcc": "ff0000",
			},
		},
		{
			name:  "hex to scheme",
			input: "AABBCC:accent1",
			expected: map[string]string{
				"AABBCC": "accent1",
			},
		},
		{
			name:  "scheme to hex",
			input: "accent1:BBFFCC",
			expected: map[string]string{
				"accent1": "BBFFCC",
			},
		},
		{
			name:  "mixed mappings",
			input: "accent1:BBFFCC,AABBCC:accent2,FF0000:00FF00",
			expected: map[string]string{
				"accent1": "BBFFCC",
				"AABBCC": "accent2",
				"FF0000": "00FF00",
			},
		},
		{
			name:  "case insensitive hex",
			input: "AaBbCc:fF0000",
			expected: map[string]string{
				"AaBbCc": "fF0000",
			},
		},
		{
			name:  "all black",
			input: "000000:FFFFFF",
			expected: map[string]string{
				"000000": "FFFFFF",
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

func TestParseColorMapping_InvalidHexColors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"hex too short source", "ABC:accent1"},
		{"hex too long source", "AABBCCD:accent1"},
		{"hex invalid chars source", "GGHHII:accent1"},
		{"hex with hash source", "#AABBCC:accent1"},
		{"hex too short target", "accent1:ABC"},
		{"hex too long target", "accent1:AABBCCD"},
		{"hex invalid chars target", "accent1:GGHHII"},
		{"hex with hash target", "accent1:#AABBCC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseColorMapping(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q but got none", tt.input)
			}
		})
	}
}

func TestParseLayoutSetMapping_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected LayoutSetMapping
	}{
		{
			name:  "property to property",
			input: "@name:matching-name",
			expected: LayoutSetMapping{
				SourceKind:     LayoutSetSourceProperty,
				SourceProperty: layoutPropertyName,
				TargetProperty: layoutPropertyMatchingName,
			},
		},
		{
			name:  "reverse property copy",
			input: "@matching-name:name",
			expected: LayoutSetMapping{
				SourceKind:     LayoutSetSourceProperty,
				SourceProperty: layoutPropertyMatchingName,
				TargetProperty: layoutPropertyName,
			},
		},
		{
			name:  "literal to property",
			input: "Layout with matchName property:matching-name",
			expected: LayoutSetMapping{
				SourceKind:     LayoutSetSourceLiteral,
				SourceLiteral:  "Layout with matchName property",
				TargetProperty: layoutPropertyMatchingName,
			},
		},
		{
			name:  "literal with colon uses last separator",
			input: "Layout: v2:name",
			expected: LayoutSetMapping{
				SourceKind:     LayoutSetSourceLiteral,
				SourceLiteral:  "Layout: v2",
				TargetProperty: layoutPropertyName,
			},
		},
		{
			name:  "whitespace trimmed around mapping parts",
			input: "  @name : matching-name  ",
			expected: LayoutSetMapping{
				SourceKind:     LayoutSetSourceProperty,
				SourceProperty: layoutPropertyName,
				TargetProperty: layoutPropertyMatchingName,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLayoutSetMapping(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.SourceKind != tt.expected.SourceKind ||
				got.SourceLiteral != tt.expected.SourceLiteral ||
				got.SourceProperty != tt.expected.SourceProperty ||
				got.TargetProperty != tt.expected.TargetProperty {
				t.Fatalf("unexpected mapping: got %+v want %+v", *got, tt.expected)
			}
		})
	}
}

func TestParseLayoutSetMapping_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{name: "empty", input: "", errContains: "cannot be empty"},
		{name: "missing colon", input: "@name", errContains: "invalid mapping format"},
		{name: "missing source", input: ":name", errContains: "cannot be empty"},
		{name: "missing target", input: "@name:", errContains: "cannot be empty"},
		{name: "invalid target sigil", input: "@name:@matching-name", errContains: "plain property name without '@'"},
		{name: "same name property", input: "@name:name", errContains: "source and target properties are the same"},
		{name: "same matching-name property", input: "@matching-name:matching-name", errContains: "source and target properties are the same"},
		{name: "invalid source property", input: "@unknown:name", errContains: "invalid source property"},
		{name: "invalid target property", input: "@name:unknown", errContains: "invalid target property"},
		{name: "reserved prefix with unknown property", input: "@literal:matching-name", errContains: "invalid source property"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseLayoutSetMapping(tt.input)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errContains)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
				t.Fatalf("expected error containing %q, got %v", tt.errContains, err)
			}
		})
	}
}
