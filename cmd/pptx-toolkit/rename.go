package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/antchfx/xmlquery"
)

// invalidNameChars contains characters that are not allowed in PowerPoint element names
// (colour schemes, font schemes, etc.). Based on empirical testing with PowerPoint.
var invalidNameChars = []rune{'.', '/', '\\', '?', ':', '*'}

// ValidateName checks if a name is valid for PowerPoint elements (colour schemes, font schemes, etc.).
// Returns an error if the name contains forbidden characters.
//
// PowerPoint accepts most characters including emoji, quotes, brackets, etc., but rejects:
// . / \ ? : * & ^ # @ !
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	// Check for invalid characters
	for _, char := range name {
		for _, invalid := range invalidNameChars {
			if char == invalid {
				// Build forbidden chars string from array
				var forbiddenChars []string
				for _, r := range invalidNameChars {
					forbiddenChars = append(forbiddenChars, string(r))
				}
				return fmt.Errorf("name contains invalid character '%c'. The following characters are not allowed: %s",
					char, strings.Join(forbiddenChars, " "))
			}
		}
	}

	return nil
}

// RenameColorScheme renames colour scheme(s) in a PowerPoint file
func RenameColorScheme(inputPath, outputPath, newName string, themeFilter []string) (int, error) {
	// Validate input
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("input file not found: %s", inputPath)
	}

	themesRenamed := 0

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "pptx-toolkit-*")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := extractPPTX(inputPath, tempDir); err != nil {
		return 0, err
	}

	// Build theme relationship mappings for validation
	masterToTheme, _ := buildThemeRelationships(tempDir)

	// Validate theme filter
	if err := validateThemeFilter(themeFilter, masterToTheme); err != nil {
		return 0, err
	}

	// Process theme files
	themesDir := filepath.Join(tempDir, "ppt", "theme")
	if _, err := os.Stat(themesDir); os.IsNotExist(err) {
		return 0, fmt.Errorf("no themes directory found")
	}

	themeFiles, err := filepath.Glob(filepath.Join(themesDir, "theme*.xml"))
	if err != nil {
		return 0, err
	}

	// Normalize theme filter (ensure .xml extension)
	normalizedFilter := make(map[string]bool)
	if len(themeFilter) > 0 {
		for _, theme := range themeFilter {
			if strings.HasSuffix(theme, ".xml") {
				normalizedFilter[theme] = true
			} else {
				normalizedFilter[theme+".xml"] = true
			}
		}
	}

	for _, themeFile := range themeFiles {
		themeName := filepath.Base(themeFile)

		// Check theme filter
		if len(normalizedFilter) > 0 {
			if !normalizedFilter[themeName] {
				continue
			}
		}

		// Read theme XML
		content, err := os.ReadFile(themeFile)
		if err != nil {
			return themesRenamed, err
		}

		// Parse to verify structure and find clrScheme
		doc, err := xmlquery.Parse(bytes.NewReader(content))
		if err != nil {
			return themesRenamed, err
		}

		// Find the clrScheme element - try with namespace first
		node := xmlquery.FindOne(doc, "//a:clrScheme")
		if node == nil {
			// Try without namespace
			node = xmlquery.FindOne(doc, "//clrScheme")
		}

		if node == nil {
			continue
		}

		// Get the current name
		var currentName string
		for _, attr := range node.Attr {
			if attr.Name.Local == "name" {
				currentName = attr.Value
				break
			}
		}

		if currentName == "" {
			continue
		}

		// Use string replacement to update the name attribute
		// Match: <...clrScheme name="currentName"...>
		// Replace with: <...clrScheme name="newName"...>
		oldAttr := fmt.Sprintf(`name="%s"`, currentName)
		newAttr := fmt.Sprintf(`name="%s"`, newName)
		modified := bytes.Replace(content, []byte(oldAttr), []byte(newAttr), 1)

		// Write back to file
		if err := os.WriteFile(themeFile, modified, 0644); err != nil {
			return themesRenamed, err
		}

		themesRenamed++
	}

	if themesRenamed == 0 {
		return 0, fmt.Errorf("no themes were renamed (this might indicate an issue with the theme filter)")
	}

	return themesRenamed, repackPPTX(tempDir, outputPath)
}
