package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/antchfx/xmlquery"
	toolkitpptx "github.com/pmdci/pptx-toolkit/internal/pptx"
)

// FontScheme describes the font scheme embedded in a theme.
type FontScheme struct {
	FileName      string
	ThemeName     string
	SchemeName    string
	MajorTypeface string
	MinorTypeface string
}

// FontSchemeUpdate describes changes to apply to a theme font scheme.
type FontSchemeUpdate struct {
	Major      string
	Minor      string
	SchemeName string
}

func parseFontSchemeXML(xmlContent []byte, fileName string) (*FontScheme, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(xmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	root := xmlquery.FindOne(doc, "//*[local-name()='theme']")
	if root == nil {
		return nil, fmt.Errorf("no theme element found")
	}

	themeName := root.SelectAttr("name")
	if themeName == "" {
		themeName = fileName
	}

	fontScheme := xmlquery.FindOne(doc, "//*[local-name()='fontScheme']")
	if fontScheme == nil {
		return nil, fmt.Errorf("no fontScheme element found")
	}

	majorLatin := xmlquery.FindOne(doc, "//*[local-name()='fontScheme']/*[local-name()='majorFont']/*[local-name()='latin']")
	if majorLatin == nil {
		return nil, fmt.Errorf("no majorFont latin element found")
	}

	minorLatin := xmlquery.FindOne(doc, "//*[local-name()='fontScheme']/*[local-name()='minorFont']/*[local-name()='latin']")
	if minorLatin == nil {
		return nil, fmt.Errorf("no minorFont latin element found")
	}

	return &FontScheme{
		FileName:      fileName,
		ThemeName:     themeName,
		SchemeName:    fontScheme.SelectAttr("name"),
		MajorTypeface: majorLatin.SelectAttr("typeface"),
		MinorTypeface: minorLatin.SelectAttr("typeface"),
	}, nil
}

// ReadFontSchemes reads the font scheme from each matching theme in a PPTX file.
func ReadFontSchemes(pptxPath string, themeFilter []string) ([]*FontScheme, error) {
	zipReader, err := zip.OpenReader(pptxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PPTX file: %w", err)
	}
	defer zipReader.Close()

	normalizedFilter := normalizeThemeFileFilter(themeFilter)
	var themeFiles []string
	for _, file := range zipReader.File {
		if filepath.Dir(file.Name) != "ppt/theme" || filepath.Ext(file.Name) != ".xml" {
			continue
		}
		base := filepath.Base(file.Name)
		if len(normalizedFilter) > 0 && !normalizedFilter[base] {
			continue
		}
		themeFiles = append(themeFiles, file.Name)
	}

	sort.Strings(themeFiles)
	if len(themeFiles) == 0 {
		return nil, noThemesMatchedError(themeFilter)
	}

	schemes := make([]*FontScheme, 0, len(themeFiles))
	for _, themeFile := range themeFiles {
		file, err := zipReader.Open(themeFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", filepath.Base(themeFile), err)
		}

		var buf bytes.Buffer
		_, err = buf.ReadFrom(file)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filepath.Base(themeFile), err)
		}

		scheme, err := parseFontSchemeXML(buf.Bytes(), filepath.Base(themeFile))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(themeFile), err)
		}
		schemes = append(schemes, scheme)
	}

	return schemes, nil
}

// SetFontScheme updates only the latin typeface(s) and optional font scheme name.
func SetFontScheme(xmlContent []byte, update FontSchemeUpdate) ([]byte, error) {
	if update.SchemeName != "" {
		if err := ValidateName(update.SchemeName); err != nil {
			return nil, err
		}
	}

	scheme, err := parseFontSchemeXML(xmlContent, "")
	if err != nil {
		return nil, err
	}

	modified := append([]byte(nil), xmlContent...)

	if update.Major != "" {
		modified, err = replaceLatinTypeface(modified, "majorFont", scheme.MajorTypeface, update.Major)
		if err != nil {
			return nil, err
		}
	}

	if update.Minor != "" {
		modified, err = replaceLatinTypeface(modified, "minorFont", scheme.MinorTypeface, update.Minor)
		if err != nil {
			return nil, err
		}
	}

	if update.SchemeName != "" {
		modified, err = replaceFontSchemeName(modified, scheme.SchemeName, update.SchemeName)
		if err != nil {
			return nil, err
		}
	}

	return modified, nil
}

// ApplyFontScheme updates matching themes in a PPTX and writes the result to outputPath.
func ApplyFontScheme(inputPath, outputPath string, update FontSchemeUpdate, themeFilter []string) (int, error) {
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("input file not found: %s", inputPath)
	}

	// Validate the filter against the zip before extracting to avoid wasted I/O.
	normalizedFilter := normalizeThemeFileFilter(themeFilter)
	if len(normalizedFilter) > 0 {
		if err := validateFilterInZip(inputPath, normalizedFilter); err != nil {
			return 0, err
		}
	}

	tempDir, err := os.MkdirTemp("", "pptx-toolkit-*")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := toolkitpptx.ExtractPPTX(inputPath, tempDir); err != nil {
		return 0, err
	}

	themesDir := filepath.Join(tempDir, "ppt", "theme")
	themeFiles, err := filepath.Glob(filepath.Join(themesDir, "theme*.xml"))
	if err != nil {
		return 0, err
	}

	sort.Strings(themeFiles)

	modifiedCount := 0
	for _, themeFile := range themeFiles {
		base := filepath.Base(themeFile)
		if len(normalizedFilter) > 0 && !normalizedFilter[base] {
			continue
		}

		content, err := os.ReadFile(themeFile)
		if err != nil {
			return modifiedCount, err
		}

		modified, err := SetFontScheme(content, update)
		if err != nil {
			return modifiedCount, fmt.Errorf("%s: %w", base, err)
		}

		if err := os.WriteFile(themeFile, modified, 0644); err != nil {
			return modifiedCount, err
		}

		modifiedCount++
	}

	if modifiedCount == 0 {
		return 0, noThemesMatchedError(themeFilter)
	}

	return modifiedCount, toolkitpptx.RepackPPTX(tempDir, outputPath)
}

func normalizeThemeFileFilter(themeFilter []string) map[string]bool {
	normalized := make(map[string]bool)
	for _, theme := range themeFilter {
		trimmed := strings.TrimSpace(theme)
		if trimmed == "" {
			continue
		}
		if !strings.HasSuffix(trimmed, ".xml") {
			trimmed += ".xml"
		}
		normalized[trimmed] = true
	}
	return normalized
}

func validateFilterInZip(pptxPath string, normalizedFilter map[string]bool) error {
	zr, err := zip.OpenReader(pptxPath)
	if err != nil {
		return fmt.Errorf("failed to open PPTX file: %w", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		if filepath.Dir(f.Name) == "ppt/theme" && normalizedFilter[filepath.Base(f.Name)] {
			return nil
		}
	}
	names := make([]string, 0, len(normalizedFilter))
	for name := range normalizedFilter {
		names = append(names, strings.TrimSuffix(name, ".xml"))
	}
	sort.Strings(names)
	return fmt.Errorf("no themes matched filter: %s", strings.Join(names, ", "))
}

func noThemesMatchedError(themeFilter []string) error {
	if len(themeFilter) == 0 {
		return fmt.Errorf("no themes found")
	}
	return fmt.Errorf("no themes matched filter: %s", strings.Join(themeFilter, ", "))
}

func replaceLatinTypeface(content []byte, role, oldTypeface, newTypeface string) ([]byte, error) {
	pattern := fmt.Sprintf(`(?s)(<[^>]*%s[^>]*>.*?<[^>]*latin[^>]*\btypeface=")%s(")`,
		regexp.QuoteMeta(role), regexp.QuoteMeta(oldTypeface))
	re := regexp.MustCompile(pattern)
	idx := re.FindSubmatchIndex(content)
	if idx == nil {
		return nil, fmt.Errorf("could not locate %s latin typeface attribute", role)
	}
	return spliceSubmatch(content, idx, newTypeface), nil
}

// TODO: Distinguish empty name="" from a missing name attribute if we ever need to
// support malformed or non-PowerPoint-authored fontScheme elements.
func replaceFontSchemeName(content []byte, oldName, newName string) ([]byte, error) {
	re := regexp.MustCompile(`(<[^>]*fontScheme[^>]*\bname=")` + regexp.QuoteMeta(escapeXMLAttributeValue(oldName)) + `(")`)
	idx := re.FindSubmatchIndex(content)
	if idx == nil {
		return nil, fmt.Errorf("could not locate fontScheme name attribute")
	}
	return spliceSubmatch(content, idx, escapeXMLAttributeValue(newName)), nil
}

// spliceSubmatch replaces the content between groups 1 and 2 of a two-group
// match with literal (non-regex) text, leaving all other bytes untouched.
func spliceSubmatch(content []byte, idx []int, literal string) []byte {
	var out []byte
	out = append(out, content[:idx[0]]...)
	out = append(out, content[idx[2]:idx[3]]...)
	out = append(out, []byte(literal)...)
	out = append(out, content[idx[4]:idx[5]]...)
	out = append(out, content[idx[1]:]...)
	return out
}
