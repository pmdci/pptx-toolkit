package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/antchfx/xmlquery"
)

// buildThemeRelationships builds a mapping of slide masters to their themes
func buildThemeRelationships(tempDir string) (map[string]string, error) {
	mapping := make(map[string]string)
	relsDir := filepath.Join(tempDir, "ppt", "slideMasters", "_rels")

	if _, err := os.Stat(relsDir); os.IsNotExist(err) {
		return mapping, nil
	}

	files, err := filepath.Glob(filepath.Join(relsDir, "slideMaster*.xml.rels"))
	if err != nil {
		return mapping, err
	}

	for _, relsFile := range files {
		masterName := strings.TrimSuffix(filepath.Base(relsFile), ".rels")

		file, err := os.Open(relsFile)
		if err != nil {
			continue
		}
		doc, err := xmlquery.Parse(file)
		file.Close()
		if err != nil {
			continue
		}

		// Find theme relationship
		xpath := "//ns:Relationship[@Type='http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme']"
		node := xmlquery.FindOne(doc, xpath)
		if node == nil {
			// Try without namespace prefix
			xpath = "//Relationship[@Type='http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme']"
			node = xmlquery.FindOne(doc, xpath)
		}

		if node != nil {
			themeTarget := node.SelectAttr("Target")
			// themeTarget is like "../theme/theme1.xml"
			themeName := filepath.Base(themeTarget)
			mapping[masterName] = themeName
		}
	}

	return mapping, nil
}

// buildLayoutToMasterMapping builds a mapping of slide layouts to their masters
func buildLayoutToMasterMapping(tempDir string) (map[string]string, error) {
	mapping := make(map[string]string)
	relsDir := filepath.Join(tempDir, "ppt", "slideLayouts", "_rels")

	if _, err := os.Stat(relsDir); os.IsNotExist(err) {
		return mapping, nil
	}

	files, err := filepath.Glob(filepath.Join(relsDir, "slideLayout*.xml.rels"))
	if err != nil {
		return mapping, err
	}

	for _, relsFile := range files {
		layoutName := strings.TrimSuffix(filepath.Base(relsFile), ".rels")

		file, err := os.Open(relsFile)
		if err != nil {
			continue
		}
		doc, err := xmlquery.Parse(file)
		file.Close()
		if err != nil {
			continue
		}

		// Find slideMaster relationship
		xpath := "//Relationship[@Type='http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster']"
		node := xmlquery.FindOne(doc, xpath)

		if node != nil {
			masterTarget := node.SelectAttr("Target")
			// masterTarget is like "../slideMasters/slideMaster1.xml"
			masterName := filepath.Base(masterTarget)
			mapping[layoutName] = masterName
		}
	}

	return mapping, nil
}

// getSlideTheme determines which theme a slide uses
func getSlideTheme(slidePath string, layoutToMaster, masterToTheme map[string]string) (string, error) {
	slideName := filepath.Base(slidePath)
	relsFile := filepath.Join(filepath.Dir(slidePath), "_rels", slideName+".rels")

	if _, err := os.Stat(relsFile); os.IsNotExist(err) {
		return "", nil
	}

	file, err := os.Open(relsFile)
	if err != nil {
		return "", nil
	}
	doc, err := xmlquery.Parse(file)
	file.Close()
	if err != nil {
		return "", nil
	}

	// Find slideLayout relationship
	xpath := "//Relationship[@Type='http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout']"
	node := xmlquery.FindOne(doc, xpath)

	if node == nil {
		return "", nil
	}

	layoutTarget := node.SelectAttr("Target")
	// layoutTarget is like "../slideLayouts/slideLayout1.xml"
	layoutName := filepath.Base(layoutTarget)

	// Find master for this layout
	masterName, exists := layoutToMaster[layoutName]
	if !exists {
		return "", nil
	}

	// Find theme for this master
	themeName, exists := masterToTheme[masterName]
	if !exists {
		return "", nil
	}

	return themeName, nil
}

// shouldProcessFile determines if a file should be processed based on theme filter
func shouldProcessFile(filePath, tempDir string, themeFilter []string,
	layoutToMaster, masterToTheme map[string]string) bool {

	if len(themeFilter) == 0 {
		return true
	}

	// Normalize theme filter (ensure .xml extension)
	themeFiles := make([]string, len(themeFilter))
	for i, theme := range themeFilter {
		if strings.HasSuffix(theme, ".xml") {
			themeFiles[i] = theme
		} else {
			themeFiles[i] = theme + ".xml"
		}
	}

	relPath, err := filepath.Rel(tempDir, filePath)
	if err != nil {
		return true
	}

	relPath = filepath.ToSlash(relPath)

	// For slides, check which theme they use
	if strings.HasPrefix(relPath, "ppt/slides/slide") {
		theme, _ := getSlideTheme(filePath, layoutToMaster, masterToTheme)
		if theme != "" {
			for _, tf := range themeFiles {
				if theme == tf {
					return true
				}
			}
			return false
		}
	}

	// For slide layouts, check via master
	if strings.HasPrefix(relPath, "ppt/slideLayouts/slideLayout") {
		layoutName := filepath.Base(filePath)
		if masterName, exists := layoutToMaster[layoutName]; exists {
			if themeName, exists := masterToTheme[masterName]; exists {
				for _, tf := range themeFiles {
					if themeName == tf {
						return true
					}
				}
				return false
			}
		}
	}

	// For slide masters, check directly
	if strings.HasPrefix(relPath, "ppt/slideMasters/slideMaster") {
		masterName := filepath.Base(filePath)
		if themeName, exists := masterToTheme[masterName]; exists {
			for _, tf := range themeFiles {
				if themeName == tf {
					return true
				}
			}
			return false
		}
	}

	// For other files (charts, diagrams, etc.), process by default
	return true
}

// ProcessPPTX processes a PowerPoint file, replacing scheme color references
func ProcessPPTX(inputPath, outputPath string, colorMapping map[string]string, themeFilter []string) (int, error) {
	// Validate input
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("input file not found: %s", inputPath)
	}

	// XML file patterns to process
	xmlPatterns := []string{
		"ppt/slides/",
		"ppt/slideLayouts/",
		"ppt/slideMasters/",
		"ppt/charts/",
		"ppt/diagrams/",
		"ppt/notesMasters/",
		"ppt/notesSlides/",
		"ppt/handoutMasters/",
	}

	filesProcessed := 0

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "pptx-toolkit-*")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract PPTX
	zipReader, err := zip.OpenReader(inputPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open PPTX: %w", err)
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		filePath := filepath.Join(tempDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return 0, err
		}

		outFile, err := os.Create(filePath)
		if err != nil {
			return 0, err
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return 0, err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return 0, err
		}
	}

	// Build theme relationship mappings
	masterToTheme, _ := buildThemeRelationships(tempDir)
	layoutToMaster, _ := buildLayoutToMasterMapping(tempDir)

	// Process XML files
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".xml") {
			return nil
		}

		// Check if file is in target patterns
		relPath, _ := filepath.Rel(tempDir, path)
		relPath = filepath.ToSlash(relPath)

		shouldProcess := false
		for _, pattern := range xmlPatterns {
			if strings.HasPrefix(relPath, pattern) {
				shouldProcess = true
				break
			}
		}

		if !shouldProcess {
			return nil
		}

		// Check theme filter
		if !shouldProcessFile(path, tempDir, themeFilter, layoutToMaster, masterToTheme) {
			return nil
		}

		// Read, replace, write
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Apply scheme → scheme/hex replacements
		modified, err := ReplaceSchemeColorsWithSrgb(content, colorMapping)
		if err != nil {
			return nil
		}

		// Apply hex → scheme/hex replacements
		modified, err = ReplaceSrgbColors(modified, colorMapping)
		if err != nil {
			return nil
		}

		if err := os.WriteFile(path, modified, info.Mode()); err != nil {
			return nil
		}

		filesProcessed++
		return nil
	})

	if err != nil {
		return filesProcessed, err
	}

	// Create output ZIP
	outFile, err := os.Create(outputPath)
	if err != nil {
		return filesProcessed, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	// Add all files to ZIP
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}

		zipFile, err := zipWriter.Create(filepath.ToSlash(relPath))
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(zipFile, bytes.NewReader(content))
		return err
	})

	return filesProcessed, err
}
