package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	layoutPropertyName         = "name"
	layoutPropertyMatchingName = "matching-name"
)

type LayoutSetSourceKind string

const (
	LayoutSetSourceLiteral  LayoutSetSourceKind = "literal"
	LayoutSetSourceProperty LayoutSetSourceKind = "property"
)

type LayoutSetMapping struct {
	SourceKind     LayoutSetSourceKind
	SourceLiteral  string
	SourceProperty string
	TargetProperty string
}

// ValidSchemeColors defines the set of valid PowerPoint scheme colors
var ValidSchemeColors = map[string]bool{
	"dk1":      true,
	"lt1":      true,
	"dk2":      true,
	"lt2":      true,
	"accent1":  true,
	"accent2":  true,
	"accent3":  true,
	"accent4":  true,
	"accent5":  true,
	"accent6":  true,
	"hlink":    true,
	"folHlink": true,
}

// hexColorPattern matches 6-character hex color codes (case-insensitive)
var hexColorPattern = regexp.MustCompile(`^[0-9A-Fa-f]{6}$`)

// isValidHexColor checks if a string is a valid 6-character hex color
func isValidHexColor(color string) bool {
	return hexColorPattern.MatchString(color)
}

// isValidColor checks if a color is either a valid scheme color or hex color
func isValidColor(color string) bool {
	return ValidSchemeColors[color] || isValidHexColor(color)
}

// ParseColorMapping parses a color mapping string into a validated map.
//
// Supports both scheme colors (e.g., accent1, dk1) and hex colors (e.g., AABBCC, FF0000).
//
// Examples:
//   - "accent1:accent3,accent5:accent3" -> scheme to scheme
//   - "accent1:BBFFCC" -> scheme to hex
//   - "AABBCC:accent2" -> hex to scheme
//   - "FF0000:00FF00" -> hex to hex
//
// Returns an error if:
// - Mapping is empty
// - Format is invalid
// - Color values are invalid (not a scheme color or valid 6-digit hex)
// - Conflicting mappings exist (e.g., accent1:accent3,accent1:accent2)
func ParseColorMapping(mappingStr string) (map[string]string, error) {
	mappingStr = strings.TrimSpace(mappingStr)
	if mappingStr == "" {
		return nil, fmt.Errorf("mapping string cannot be empty")
	}

	mappings := make(map[string]string)
	pairs := strings.Split(mappingStr, ",")

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		if !strings.Contains(pair, ":") {
			return nil, fmt.Errorf("invalid mapping format: '%s'. Expected 'source:target'", pair)
		}

		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid mapping format: '%s'. Expected exactly one ':'", pair)
		}

		source := strings.TrimSpace(parts[0])
		target := strings.TrimSpace(parts[1])

		if source == "" || target == "" {
			return nil, fmt.Errorf("invalid mapping: '%s'. Source and target cannot be empty", pair)
		}

		// Validate colors (scheme names or hex values)
		if !isValidColor(source) {
			if isValidHexColor(source) {
				// Already valid hex, shouldn't reach here
				return nil, fmt.Errorf("internal error validating source color: '%s'", source)
			}
			return nil, fmt.Errorf("invalid source color: '%s'. Must be a valid scheme color (%s) or 6-digit hex color (e.g., AABBCC)",
				source, getValidColorsString())
		}

		if !isValidColor(target) {
			if isValidHexColor(target) {
				// Already valid hex, shouldn't reach here
				return nil, fmt.Errorf("internal error validating target color: '%s'", target)
			}
			return nil, fmt.Errorf("invalid target color: '%s'. Must be a valid scheme color (%s) or 6-digit hex color (e.g., AABBCC)",
				target, getValidColorsString())
		}

		// Check for conflicts
		if existingTarget, exists := mappings[source]; exists {
			if existingTarget != target {
				return nil, fmt.Errorf("conflicting mappings for '%s':\n  - %s → %s\n  - %s → %s",
					source, source, existingTarget, source, target)
			}
			// Duplicate identical mapping, skip
			continue
		}

		mappings[source] = target
	}

	if len(mappings) == 0 {
		return nil, fmt.Errorf("no valid mappings found")
	}

	return mappings, nil
}

// getValidColorsString returns a sorted, comma-separated string of valid color names
func getValidColorsString() string {
	colors := make([]string, 0, len(ValidSchemeColors))
	for color := range ValidSchemeColors {
		colors = append(colors, color)
	}
	sort.Strings(colors)
	return strings.Join(colors, ", ")
}

func isValidLayoutProperty(property string) bool {
	return property == layoutPropertyName || property == layoutPropertyMatchingName
}

// ParseLayoutSetMapping parses a layout set mapping expression.
//
// The syntax is:
//   - "@name:matching-name" for property-to-property copy
//   - "Literal value:matching-name" for literal-to-property assignment
//
// The left side may be either a literal string or a property reference marked
// with '@'. The right side must always be a supported property name without a sigil.
func ParseLayoutSetMapping(mappingStr string) (*LayoutSetMapping, error) {
	mappingStr = strings.TrimSpace(mappingStr)
	if mappingStr == "" {
		return nil, fmt.Errorf("mapping string cannot be empty")
	}

	sep := strings.LastIndex(mappingStr, ":")
	if sep == -1 {
		return nil, fmt.Errorf("invalid mapping format: %q. Expected 'source:target'", mappingStr)
	}

	source := strings.TrimSpace(mappingStr[:sep])
	target := strings.TrimSpace(mappingStr[sep+1:])
	if source == "" || target == "" {
		return nil, fmt.Errorf("invalid mapping: %q. Source and target cannot be empty", mappingStr)
	}

	if strings.HasPrefix(target, "@") {
		return nil, fmt.Errorf("invalid target property: %q. Target must be a plain property name without '@'", target)
	}
	if !isValidLayoutProperty(target) {
		return nil, fmt.Errorf("invalid target property: %q. Must be one of: %s, %s", target, layoutPropertyName, layoutPropertyMatchingName)
	}

	mapping := &LayoutSetMapping{TargetProperty: target}
	if strings.HasPrefix(source, "@") {
		property := strings.TrimSpace(strings.TrimPrefix(source, "@"))
		if property == "" {
			return nil, fmt.Errorf("invalid source property: %q", source)
		}
		if !isValidLayoutProperty(property) {
			return nil, fmt.Errorf("invalid source property: %q. Must be one of: %s, %s", property, layoutPropertyName, layoutPropertyMatchingName)
		}
		if property == target {
			return nil, fmt.Errorf("source and target properties are the same; exiting")
		}
		mapping.SourceKind = LayoutSetSourceProperty
		mapping.SourceProperty = property
		return mapping, nil
	}

	mapping.SourceKind = LayoutSetSourceLiteral
	mapping.SourceLiteral = source
	return mapping, nil
}
