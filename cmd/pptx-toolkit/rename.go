package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toolkitpptx "github.com/pmdci/pptx-toolkit/internal/pptx"
)

const drawingMLNamespace = "http://schemas.openxmlformats.org/drawingml/2006/main"

// ValidateName checks if a name is valid for PowerPoint elements (colour schemes, font schemes, etc.).
// Returns an error if the name violates a general invariant shared by name-setting operations.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	return nil
}

// SetColorSchemeName sets the colour scheme name in matching theme definitions.
func SetColorSchemeName(inputPath, outputPath, newName string, themeFilter []string) (int, error) {
	if err := ValidateName(newName); err != nil {
		return 0, err
	}

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

	if err := toolkitpptx.ExtractPPTX(inputPath, tempDir); err != nil {
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

		start, end, err := findClrSchemeStartTagRange(content)
		if err != nil {
			return themesRenamed, fmt.Errorf("%s: %w", themeName, err)
		}
		if start == -1 {
			continue
		}

		startTag := content[start:end]
		if !nameAttrRe.Match(startTag) {
			continue
		}

		startTag = setAttrOnStartTag(startTag, "name", newName, nameAttrRe)
		modified := replaceByteRange(content, start, end, startTag)

		// Write back to file
		if bytes.Equal(modified, content) {
			continue
		}
		if err := os.WriteFile(themeFile, modified, 0644); err != nil {
			return themesRenamed, err
		}

		themesRenamed++
	}

	if themesRenamed == 0 {
		return 0, fmt.Errorf("no themes were renamed (this might indicate an issue with the theme filter)")
	}

	return themesRenamed, toolkitpptx.RepackPPTX(tempDir, outputPath)
}

func findClrSchemeStartTagRange(content []byte) (int, int, error) {
	return findStartElementRange(content, func(se xml.StartElement, _ int) bool {
		return se.Name.Local == "clrScheme" && se.Name.Space == drawingMLNamespace
	}, "theme XML")
}
