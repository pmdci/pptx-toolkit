package main

import (
	"bytes"
	"regexp"
	"strings"
)

// ReplaceSchemeColors replaces scheme color references in PowerPoint XML content.
//
// It finds all <schemeClr val="accent1"/> elements (namespace-agnostic) and replaces
// the val attribute according to the color mapping. Replacement is atomic (no cascading).
//
// Returns the modified XML bytes, or the original if no replacements are needed.
func ReplaceSchemeColors(xmlContent []byte, colorMapping map[string]string) ([]byte, error) {
	if len(colorMapping) == 0 {
		return xmlContent, nil
	}

	// Use regex to find and replace schemeClr val attributes
	// Pattern matches: <prefix:schemeClr val="colorname" with any namespace prefix
	// This is namespace-agnostic and preserves XML structure
	pattern := regexp.MustCompile(`(<[^:>]*:?schemeClr[^>]*\sval=")([^"]+)(")`)

	// Atomic replacement: capture all matches first, then replace
	// This prevents cascading replacements
	matches := pattern.FindAllSubmatchIndex(xmlContent, -1)
	if len(matches) == 0 {
		return xmlContent, nil
	}

	// Build new content by copying unchanged parts and replacing matches
	var result bytes.Buffer
	lastEnd := 0

	for _, match := range matches {
		// match[0], match[1] = full match start, end
		// match[4], match[5] = color value start, end (capture group 2)

		// Write everything before this match
		result.Write(xmlContent[lastEnd:match[0]])

		// Get current color value
		currentColor := string(xmlContent[match[4]:match[5]])

		// Write opening (prefix + 'val="')
		result.Write(xmlContent[match[2]:match[3]])

		// Write replacement color or original
		if newColor, exists := colorMapping[currentColor]; exists {
			result.WriteString(newColor)
		} else {
			result.WriteString(currentColor)
		}

		// Write closing ('"')
		result.Write(xmlContent[match[6]:match[7]])

		lastEnd = match[1]
	}

	// Write remaining content
	result.Write(xmlContent[lastEnd:])

	return result.Bytes(), nil
}

// ReplaceSrgbColors replaces RGB color values in PowerPoint XML content.
//
// It finds all <srgbClr val="AABBCC"/> elements (namespace-agnostic) and either:
//   - Replaces the hex value with another hex value (HEX → HEX)
//   - Replaces the entire element with <schemeClr> (HEX → Scheme)
//
// Replacement is atomic (no cascading), matching the behavior of ReplaceSchemeColors.
//
// Returns the modified XML bytes, or the original if no replacements are needed.
func ReplaceSrgbColors(xmlContent []byte, colorMapping map[string]string) ([]byte, error) {
	if len(colorMapping) == 0 {
		return xmlContent, nil
	}

	// Build a case-insensitive mapping for hex values
	hexMapping := make(map[string]string)
	for source, target := range colorMapping {
		// Only include mappings where source is a hex value
		if isValidHexColor(source) {
			// Normalize to uppercase for consistent matching
			hexMapping[strings.ToUpper(source)] = target
		}
	}

	if len(hexMapping) == 0 {
		return xmlContent, nil
	}

	// Pattern matches: <prefix:srgbClr val="AABBCC" with any namespace prefix
	pattern := regexp.MustCompile(`(<[^:>]*:?srgbClr[^>]*\sval=")([0-9A-Fa-f]{6})(")`)

	// Atomic replacement: capture all matches first, then replace
	matches := pattern.FindAllSubmatchIndex(xmlContent, -1)
	if len(matches) == 0 {
		return xmlContent, nil
	}

	// Build new content by copying unchanged parts and replacing matches
	var result bytes.Buffer
	lastEnd := 0

	for _, match := range matches {
		// match[0], match[1] = full match start, end
		// match[4], match[5] = hex value start, end (capture group 2)

		// Write everything before this match
		result.Write(xmlContent[lastEnd:match[0]])

		// Get current hex value (normalize to uppercase)
		currentHex := strings.ToUpper(string(xmlContent[match[4]:match[5]]))

		// Check if we have a mapping for this hex value
		if newColor, exists := hexMapping[currentHex]; exists {
			// Determine if target is hex or scheme
			if isValidHexColor(newColor) {
				// HEX → HEX: just replace the value
				result.Write(xmlContent[match[2]:match[3]]) // opening (prefix + 'val="')
				result.WriteString(strings.ToUpper(newColor))
				result.Write(xmlContent[match[6]:match[7]]) // closing ('"')
			} else {
				// HEX → Scheme: replace entire element
				// Extract namespace prefix from opening tag
				opening := string(xmlContent[match[2]:match[3]])
				// opening looks like: <a:srgbClr val="
				// We need to extract the prefix (e.g., "a:")
				prefixEnd := strings.Index(opening, "srgbClr")
				prefix := ""
				if prefixEnd > 0 {
					prefix = opening[1:prefixEnd] // Extract prefix including ':'
				}

				// Write replacement as schemeClr
				result.WriteString("<")
				result.WriteString(prefix)
				result.WriteString("schemeClr val=\"")
				result.WriteString(newColor)
				result.WriteString("\"")
			}
		} else {
			// No mapping, write original
			result.Write(xmlContent[match[0]:match[1]])
		}

		lastEnd = match[1]
	}

	// Write remaining content
	result.Write(xmlContent[lastEnd:])

	return result.Bytes(), nil
}

// ReplaceSchemeColorsWithSrgb replaces scheme color references with RGB values.
//
// It finds all <schemeClr val="accent1"/> elements and replaces them with
// <srgbClr val="AABBCC"/> when the mapping specifies a hex target.
//
// Replacement is atomic (no cascading).
//
// Returns the modified XML bytes, or the original if no replacements are needed.
func ReplaceSchemeColorsWithSrgb(xmlContent []byte, colorMapping map[string]string) ([]byte, error) {
	if len(colorMapping) == 0 {
		return xmlContent, nil
	}

	// Build mapping for scheme → hex conversions only
	schemeToHexMapping := make(map[string]string)
	schemeToSchemeMapping := make(map[string]string)

	for source, target := range colorMapping {
		if ValidSchemeColors[source] {
			if isValidHexColor(target) {
				schemeToHexMapping[source] = strings.ToUpper(target)
			} else {
				schemeToSchemeMapping[source] = target
			}
		}
	}

	// Pattern matches: <prefix:schemeClr val="accent1" with any namespace prefix
	pattern := regexp.MustCompile(`(<[^:>]*:?schemeClr[^>]*\sval=")([^"]+)(")`)

	// Atomic replacement: capture all matches first, then replace
	matches := pattern.FindAllSubmatchIndex(xmlContent, -1)
	if len(matches) == 0 {
		return xmlContent, nil
	}

	// Build new content by copying unchanged parts and replacing matches
	var result bytes.Buffer
	lastEnd := 0

	for _, match := range matches {
		// match[0], match[1] = full match start, end
		// match[4], match[5] = color value start, end (capture group 2)

		// Write everything before this match
		result.Write(xmlContent[lastEnd:match[0]])

		// Get current scheme color
		currentColor := string(xmlContent[match[4]:match[5]])

		// Check for scheme → hex conversion
		if hexColor, exists := schemeToHexMapping[currentColor]; exists {
			// Scheme → HEX: replace entire element
			// Extract namespace prefix from opening tag
			opening := string(xmlContent[match[2]:match[3]])
			prefixEnd := strings.Index(opening, "schemeClr")
			prefix := ""
			if prefixEnd > 0 {
				prefix = opening[1:prefixEnd] // Extract prefix including ':'
			}

			// Write replacement as srgbClr
			result.WriteString("<")
			result.WriteString(prefix)
			result.WriteString("srgbClr val=\"")
			result.WriteString(hexColor)
			result.WriteString("\"")
		} else if newScheme, exists := schemeToSchemeMapping[currentColor]; exists {
			// Scheme → Scheme: just replace the value
			result.Write(xmlContent[match[2]:match[3]]) // opening
			result.WriteString(newScheme)
			result.Write(xmlContent[match[6]:match[7]]) // closing
		} else {
			// No mapping, write original
			result.Write(xmlContent[match[0]:match[1]])
		}

		lastEnd = match[1]
	}

	// Write remaining content
	result.Write(xmlContent[lastEnd:])

	return result.Bytes(), nil
}
