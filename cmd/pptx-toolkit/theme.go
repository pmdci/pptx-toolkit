package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/antchfx/xmlquery"
)

// ColorScheme represents a PowerPoint color scheme with all scheme colors
type ColorScheme struct {
	Dk1      string `json:"dk1"`
	Lt1      string `json:"lt1"`
	Dk2      string `json:"dk2"`
	Lt2      string `json:"lt2"`
	Accent1  string `json:"accent1"`
	Accent2  string `json:"accent2"`
	Accent3  string `json:"accent3"`
	Accent4  string `json:"accent4"`
	Accent5  string `json:"accent5"`
	Accent6  string `json:"accent6"`
	Hlink    string `json:"hlink"`
	FolHlink string `json:"folHlink"`
}

// Theme represents a PowerPoint theme
type Theme struct {
	FileName        string      `json:"fileName"`        // e.g., "theme1.xml"
	ThemeName       string      `json:"themeName"`       // e.g., "Office Theme Deck"
	ColorSchemeName string      `json:"colorSchemeName"` // e.g., "Office"
	FontSchemeName  string      `json:"fontSchemeName"`  // e.g., "Office"
	Colors          ColorScheme `json:"colors"`
}

// extractRGBColor extracts RGB color value from a color definition element
func extractRGBColor(colorElement *xmlquery.Node) string {
	if colorElement == nil {
		return "000000"
	}

	// Try <a:srgbClr val="156082"/>
	if srgbNode := colorElement.SelectElement("//*[local-name()='srgbClr']"); srgbNode != nil {
		if val := srgbNode.SelectAttr("val"); val != "" {
			return val
		}
	}

	// Try <a:sysClr val="windowText" lastClr="000000"/>
	if sysNode := colorElement.SelectElement("//*[local-name()='sysClr']"); sysNode != nil {
		if lastClr := sysNode.SelectAttr("lastClr"); lastClr != "" {
			return lastClr
		}
	}

	return "000000"
}

// parseThemeXML parses a theme XML file and extracts theme information
func parseThemeXML(xmlContent []byte, fileName string) (*Theme, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(xmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	// Extract theme name from root element
	root := xmlquery.FindOne(doc, "//*[local-name()='theme']")
	if root == nil {
		return nil, fmt.Errorf("no theme element found")
	}

	themeName := root.SelectAttr("name")
	if themeName == "" {
		themeName = fileName
	}

	// Find color scheme
	clrScheme := xmlquery.FindOne(doc, "//*[local-name()='clrScheme']")
	if clrScheme == nil {
		return nil, fmt.Errorf("no clrScheme element found")
	}

	colorSchemeName := clrScheme.SelectAttr("name")
	if colorSchemeName == "" {
		colorSchemeName = "Unknown"
	}

	fontSchemeName := ""
	if fontScheme := xmlquery.FindOne(doc, "//*[local-name()='fontScheme']"); fontScheme != nil {
		fontSchemeName = fontScheme.SelectAttr("name")
	}

	// Extract all scheme colors
	getColor := func(name string) string {
		xpath := fmt.Sprintf("//*[local-name()='clrScheme']/*[local-name()='%s']", name)
		elem := xmlquery.FindOne(doc, xpath)
		return extractRGBColor(elem)
	}

	colors := ColorScheme{
		Dk1:      getColor("dk1"),
		Lt1:      getColor("lt1"),
		Dk2:      getColor("dk2"),
		Lt2:      getColor("lt2"),
		Accent1:  getColor("accent1"),
		Accent2:  getColor("accent2"),
		Accent3:  getColor("accent3"),
		Accent4:  getColor("accent4"),
		Accent5:  getColor("accent5"),
		Accent6:  getColor("accent6"),
		Hlink:    getColor("hlink"),
		FolHlink: getColor("folHlink"),
	}

	return &Theme{
		FileName:        fileName,
		ThemeName:       themeName,
		ColorSchemeName: colorSchemeName,
		FontSchemeName:  fontSchemeName,
		Colors:          colors,
	}, nil
}

// ReadThemes reads all themes from a PowerPoint file
func ReadThemes(pptxPath string) ([]*Theme, error) {
	zipReader, err := zip.OpenReader(pptxPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PPTX file: %w", err)
	}
	defer zipReader.Close()

	var themes []*Theme
	var themeFiles []string

	// Collect theme files
	for _, file := range zipReader.File {
		if filepath.Dir(file.Name) == "ppt/theme" && filepath.Ext(file.Name) == ".xml" {
			themeFiles = append(themeFiles, file.Name)
		}
	}

	// Sort for consistent ordering (theme1, theme2, etc.)
	sort.Strings(themeFiles)

	// Parse each theme file
	for _, themeFile := range themeFiles {
		file, err := zipReader.Open(themeFile)
		if err != nil {
			continue
		}

		var buf bytes.Buffer
		_, err = buf.ReadFrom(file)
		file.Close()

		if err != nil {
			continue
		}

		fileName := filepath.Base(themeFile)
		theme, err := parseThemeXML(buf.Bytes(), fileName)
		if err == nil {
			themes = append(themes, theme)
		}
	}

	return themes, nil
}

func filterThemes(themes []*Theme, themeFilter []string) ([]*Theme, error) {
	if len(themeFilter) == 0 {
		return themes, nil
	}

	requested := make(map[string]bool, len(themeFilter))
	for _, theme := range themeFilter {
		if strings.HasSuffix(theme, ".xml") {
			requested[theme] = true
		} else {
			requested[theme] = true
			requested[theme+".xml"] = true
		}
	}

	var filtered []*Theme
	found := make(map[string]bool, len(requested))
	for _, theme := range themes {
		if requested[theme.FileName] {
			filtered = append(filtered, theme)
			found[theme.FileName] = true
			found[strings.TrimSuffix(theme.FileName, ".xml")] = true
		}
	}

	var missing []string
	for _, theme := range themeFilter {
		if !found[theme] && !found[strings.TrimSuffix(theme, ".xml")] {
			missing = append(missing, theme)
		}
	}

	if len(missing) > 0 {
		available := make([]string, 0, len(themes))
		for _, theme := range themes {
			available = append(available, strings.TrimSuffix(theme.FileName, ".xml"))
		}
		sort.Strings(available)
		return nil, fmt.Errorf("theme(s) not found: %s\nAvailable themes: %s", strings.Join(missing, ", "), strings.Join(available, ", "))
	}

	return filtered, nil
}
