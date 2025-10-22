package main

import (
	"bytes"
	"regexp"
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
