package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadThemes(t *testing.T) {
	// Path to test fixture
	testPPTX := filepath.Join("testdata", "test.pptx")

	// Check if fixture exists
	if _, err := os.Stat(testPPTX); os.IsNotExist(err) {
		t.Skip("test.pptx fixture not found")
	}

	themes, err := ReadThemes(testPPTX)
	if err != nil {
		t.Fatalf("failed to read themes: %v", err)
	}

	if len(themes) == 0 {
		t.Fatal("expected at least one theme, got none")
	}

	// Verify each theme has required fields
	for i, theme := range themes {
		if theme.FileName == "" {
			t.Errorf("theme %d: file name is empty", i)
		}

		if theme.ThemeName == "" {
			t.Errorf("theme %d: theme name is empty", i)
		}

		if theme.ColorSchemeName == "" {
			t.Errorf("theme %d: color scheme name is empty", i)
		}

		// Verify colors are extracted (at least some should not be default "000000")
		colors := []string{
			theme.Colors.Dk1,
			theme.Colors.Lt1,
			theme.Colors.Dk2,
			theme.Colors.Lt2,
			theme.Colors.Accent1,
			theme.Colors.Accent2,
			theme.Colors.Accent3,
			theme.Colors.Accent4,
			theme.Colors.Accent5,
			theme.Colors.Accent6,
			theme.Colors.Hlink,
			theme.Colors.FolHlink,
		}

		nonDefaultCount := 0
		for _, color := range colors {
			if len(color) == 6 { // Valid hex color
				nonDefaultCount++
			}
		}

		if nonDefaultCount == 0 {
			t.Errorf("theme %d: no valid colors extracted", i)
		}
	}

	t.Logf("Successfully read %d theme(s):", len(themes))
	for i, theme := range themes {
		t.Logf("  Theme %d: %s (%s) - %s", i+1, theme.ThemeName, theme.ColorSchemeName, theme.FileName)
	}
}

func TestParseThemeXML(t *testing.T) {
	// Create minimal valid theme XML
	xmlContent := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="Test Theme">
	<a:themeElements>
		<a:clrScheme name="Test Colors">
			<a:dk1><a:srgbClr val="000000"/></a:dk1>
			<a:lt1><a:srgbClr val="FFFFFF"/></a:lt1>
			<a:dk2><a:srgbClr val="1F497D"/></a:dk2>
			<a:lt2><a:srgbClr val="EEECE1"/></a:lt2>
			<a:accent1><a:srgbClr val="4F81BD"/></a:accent1>
			<a:accent2><a:srgbClr val="C0504D"/></a:accent2>
			<a:accent3><a:srgbClr val="9BBB59"/></a:accent3>
			<a:accent4><a:srgbClr val="8064A2"/></a:accent4>
			<a:accent5><a:srgbClr val="4BACC6"/></a:accent5>
			<a:accent6><a:srgbClr val="F79646"/></a:accent6>
			<a:hlink><a:srgbClr val="0000FF"/></a:hlink>
			<a:folHlink><a:srgbClr val="800080"/></a:folHlink>
		</a:clrScheme>
	</a:themeElements>
</a:theme>`)

	theme, err := parseThemeXML(xmlContent, "theme1.xml")
	if err != nil {
		t.Fatalf("failed to parse theme XML: %v", err)
	}

	if theme.FileName != "theme1.xml" {
		t.Errorf("expected fileName 'theme1.xml', got '%s'", theme.FileName)
	}

	if theme.ThemeName != "Test Theme" {
		t.Errorf("expected themeName 'Test Theme', got '%s'", theme.ThemeName)
	}

	if theme.ColorSchemeName != "Test Colors" {
		t.Errorf("expected colorSchemeName 'Test Colors', got '%s'", theme.ColorSchemeName)
	}

	// Verify specific colors
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"Dk1", theme.Colors.Dk1, "000000"},
		{"Lt1", theme.Colors.Lt1, "FFFFFF"},
		{"Accent1", theme.Colors.Accent1, "4F81BD"},
		{"Accent2", theme.Colors.Accent2, "C0504D"},
		{"Hlink", theme.Colors.Hlink, "0000FF"},
		{"FolHlink", theme.Colors.FolHlink, "800080"},
	}

	for _, tt := range tests {
		if tt.got != tt.expected {
			t.Errorf("color %s: expected %s, got %s", tt.name, tt.expected, tt.got)
		}
	}
}

func TestParseThemeXML_SystemColors(t *testing.T) {
	// Test with system colors (sysClr instead of srgbClr)
	xmlContent := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="System Theme">
	<a:themeElements>
		<a:clrScheme name="System Colors">
			<a:dk1><a:sysClr val="windowText" lastClr="000000"/></a:dk1>
			<a:lt1><a:sysClr val="window" lastClr="FFFFFF"/></a:lt1>
			<a:dk2><a:srgbClr val="1F497D"/></a:dk2>
			<a:lt2><a:srgbClr val="EEECE1"/></a:lt2>
			<a:accent1><a:srgbClr val="156082"/></a:accent1>
			<a:accent2><a:srgbClr val="C0504D"/></a:accent2>
			<a:accent3><a:srgbClr val="9BBB59"/></a:accent3>
			<a:accent4><a:srgbClr val="8064A2"/></a:accent4>
			<a:accent5><a:srgbClr val="4BACC6"/></a:accent5>
			<a:accent6><a:srgbClr val="F79646"/></a:accent6>
			<a:hlink><a:srgbClr val="0000FF"/></a:hlink>
			<a:folHlink><a:srgbClr val="800080"/></a:folHlink>
		</a:clrScheme>
	</a:themeElements>
</a:theme>`)

	theme, err := parseThemeXML(xmlContent, "theme2.xml")
	if err != nil {
		t.Fatalf("failed to parse theme XML: %v", err)
	}

	// System colors should be extracted from lastClr
	if theme.Colors.Dk1 != "000000" {
		t.Errorf("expected dk1 '000000' from sysClr, got '%s'", theme.Colors.Dk1)
	}

	if theme.Colors.Lt1 != "FFFFFF" {
		t.Errorf("expected lt1 'FFFFFF' from sysClr, got '%s'", theme.Colors.Lt1)
	}

	// Regular srgbClr should still work
	if theme.Colors.Accent1 != "156082" {
		t.Errorf("expected accent1 '156082', got '%s'", theme.Colors.Accent1)
	}
}
