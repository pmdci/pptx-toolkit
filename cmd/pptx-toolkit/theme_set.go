package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	toolkitpptx "github.com/pmdci/pptx-toolkit/internal/pptx"
)

const docPropsVTypesNamespace = "http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes"
const extendedPropertiesNamespace = "http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"

// SetThemeName renames a single slide-master-bound theme and keeps app metadata in sync.
func SetThemeName(inputPath, outputPath, newName string, themeFilter []string) (int, error) {
	if err := ValidateName(newName); err != nil {
		return 0, err
	}
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("input file not found: %s", inputPath)
	}

	targetTheme, err := normalizeSingleThemeTarget(themeFilter)
	if err != nil {
		return 0, err
	}

	tempDir, err := os.MkdirTemp("", "pptx-toolkit-*")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := toolkitpptx.ExtractPPTX(inputPath, tempDir); err != nil {
		return 0, err
	}

	targetSummary, err := resolveThemeRenameTarget(tempDir, targetTheme)
	if err != nil {
		return 0, err
	}

	allThemes, err := readThemesFromDirStrict(tempDir)
	if err != nil {
		return 0, err
	}
	if conflictFile, conflictName, ok := findThemeNameConflict(allThemes, targetSummary.Theme.FileName, newName); ok {
		return 0, fmt.Errorf("theme name %q conflicts with existing theme %q (%q) under PowerPoint's first-20-characters uniqueness rule", newName, strings.TrimSuffix(conflictFile, ".xml"), conflictName)
	}

	themePath := filepath.Join(tempDir, "ppt", "theme", targetSummary.Theme.FileName)
	themeXML, err := os.ReadFile(themePath)
	if err != nil {
		return 0, err
	}

	currentName, err := readThemeRootName(themeXML)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", targetSummary.Theme.FileName, err)
	}
	if currentName == newName {
		if err := toolkitpptx.RepackPPTX(tempDir, outputPath); err != nil {
			return 0, err
		}
		return 0, nil
	}

	modifiedThemeXML, err := setThemeRootName(themeXML, newName)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", targetSummary.Theme.FileName, err)
	}
	if err := os.WriteFile(themePath, modifiedThemeXML, 0644); err != nil {
		return 0, err
	}

	appPath := filepath.Join(tempDir, "docProps", "app.xml")
	if _, err := os.Stat(appPath); err == nil {
		appXML, err := os.ReadFile(appPath)
		if err != nil {
			return 0, err
		}

		modifiedAppXML, err := replaceTitleOfPartsThemeName(appXML, currentName, newName)
		if err != nil {
			return 0, err
		}
		if err := os.WriteFile(appPath, modifiedAppXML, 0644); err != nil {
			return 0, err
		}
	} else if !os.IsNotExist(err) {
		return 0, err
	}

	if err := toolkitpptx.RepackPPTX(tempDir, outputPath); err != nil {
		return 0, err
	}
	return 1, nil
}

func normalizeSingleThemeTarget(themeFilter []string) (string, error) {
	normalized := normalizeThemeFileFilter(themeFilter)
	if len(normalized) == 0 {
		return "", fmt.Errorf("--theme is required when using theme set")
	}
	if len(normalized) != 1 {
		return "", fmt.Errorf("theme set requires exactly one target theme")
	}
	for theme := range normalized {
		return theme, nil
	}
	return "", fmt.Errorf("theme set requires exactly one target theme")
}

func resolveThemeRenameTarget(tempDir, themeFile string) (*ThemeSummary, error) {
	summaries, err := readThemeSummariesFromDir(tempDir, []string{themeFile})
	if err != nil {
		return nil, err
	}
	if len(summaries) != 1 {
		return nil, fmt.Errorf("theme set requires exactly one target theme")
	}

	for _, binding := range summaries[0].Bindings {
		if binding.MasterType == masterTypeSlide {
			return summaries[0], nil
		}
	}

	return nil, fmt.Errorf("%s is not a slide-master theme; only slide-master-bound themes can be renamed", strings.TrimSuffix(themeFile, ".xml"))
}

func findThemeNameConflict(themes []*Theme, targetFile, newName string) (string, string, bool) {
	targetPrefix := normalizeThemeNamePrefix(newName)
	for _, theme := range themes {
		if theme.FileName == targetFile {
			continue
		}
		if normalizeThemeNamePrefix(theme.ThemeName) == targetPrefix {
			return theme.FileName, theme.ThemeName, true
		}
	}
	return "", "", false
}

func firstNRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

func normalizeThemeNamePrefix(name string) string {
	return strings.ToLower(firstNRunes(name, 20))
}

func setThemeRootName(content []byte, newName string) ([]byte, error) {
	start, end, err := findStartElementRange(content, func(se xml.StartElement, depth int) bool {
		return depth == 0 && se.Name.Local == "theme" && se.Name.Space == drawingMLNamespace
	}, "theme XML")
	if err != nil {
		return nil, err
	}
	if start == -1 {
		return nil, fmt.Errorf("no theme element found")
	}

	startTag := content[start:end]
	if !nameAttrRe.Match(startTag) {
		return nil, fmt.Errorf("theme element has no name attribute")
	}
	startTag = setAttrOnStartTag(startTag, "name", newName, nameAttrRe)
	return replaceByteRange(content, start, end, startTag), nil
}

func readThemeRootName(content []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			return "", fmt.Errorf("no theme element found")
		}
		if err != nil {
			return "", fmt.Errorf("parsing theme XML: %w", err)
		}

		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "theme" || start.Name.Space != drawingMLNamespace {
			continue
		}

		for _, attr := range start.Attr {
			if attr.Name.Local == "name" {
				if attr.Value == "" {
					return "", fmt.Errorf("theme element has empty name attribute")
				}
				return attr.Value, nil
			}
		}
		return "", fmt.Errorf("theme element has no name attribute")
	}
}

func replaceTitleOfPartsThemeName(content []byte, currentName, newName string) ([]byte, error) {
	matches, err := findLPStrMatches(content, currentName)
	if err != nil {
		return nil, fmt.Errorf("app.xml: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("app.xml: no TitlesOfParts entry matched theme name %q", currentName)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("app.xml: multiple TitlesOfParts entries matched theme name %q", currentName)
	}

	replacement := []byte(escapeXMLTextContent(newName))
	match := matches[0]
	return replaceByteRange(content, match.Start, match.End, replacement), nil
}

type textMatch struct {
	Start int
	End   int
}

func findLPStrMatches(content []byte, want string) ([]textMatch, error) {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	var matches []textMatch
	inTitlesOfParts := false
	titlesDepth := 0

	for {
		preOffset := int(decoder.InputOffset())
		tok, err := decoder.Token()
		if err == io.EOF {
			return matches, nil
		}
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "TitlesOfParts" && t.Name.Space == extendedPropertiesNamespace {
				inTitlesOfParts = true
				titlesDepth = 1
				continue
			}
			if !inTitlesOfParts {
				continue
			}
			if t.Name.Local != "lpstr" || t.Name.Space != docPropsVTypesNamespace {
				titlesDepth++
				continue
			}

			textStart := int(decoder.InputOffset())
			textEnd := textStart
			var b strings.Builder
			depth := 1

			for depth > 0 {
				preInnerOffset := int(decoder.InputOffset())
				innerTok, err := decoder.Token()
				if err != nil {
					if err == io.EOF {
						return nil, fmt.Errorf("unexpected EOF while reading lpstr starting at byte %d", preOffset)
					}
					return nil, err
				}

				switch inner := innerTok.(type) {
				case xml.CharData:
					if depth == 1 {
						b.Write([]byte(inner))
						textEnd = int(decoder.InputOffset())
					}
				case xml.StartElement:
					depth++
					titlesDepth++
				case xml.EndElement:
					depth--
					if depth == 0 {
						textEnd = preInnerOffset
					} else {
						titlesDepth--
					}
				}
			}
			if b.String() == want {
				matches = append(matches, textMatch{Start: textStart, End: textEnd})
			}
		case xml.EndElement:
			if inTitlesOfParts {
				titlesDepth--
				if titlesDepth == 0 {
					inTitlesOfParts = false
				}
			}
		}
	}
}

func titlesOfParts(content []byte) ([]string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	var values []string
	inTitlesOfParts := false
	titlesDepth := 0

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			return values, nil
		}
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "TitlesOfParts" && t.Name.Space == extendedPropertiesNamespace {
				inTitlesOfParts = true
				titlesDepth = 1
				continue
			}
			if !inTitlesOfParts {
				continue
			}
			if t.Name.Local != "lpstr" || t.Name.Space != docPropsVTypesNamespace {
				titlesDepth++
				continue
			}

			var b strings.Builder
			depth := 1
			for depth > 0 {
				innerTok, err := decoder.Token()
				if err != nil {
					if err == io.EOF {
						return nil, fmt.Errorf("unexpected EOF while reading lpstr")
					}
					return nil, err
				}
				switch inner := innerTok.(type) {
				case xml.CharData:
					if depth == 1 {
						b.Write([]byte(inner))
					}
				case xml.StartElement:
					depth++
					titlesDepth++
				case xml.EndElement:
					depth--
					if depth > 0 {
						titlesDepth--
					}
				}
			}
			values = append(values, b.String())
		case xml.EndElement:
			if inTitlesOfParts {
				titlesDepth--
				if titlesDepth == 0 {
					inTitlesOfParts = false
				}
			}
		}
	}
}
