package main

import (
	"bytes"
	"testing"

	"github.com/antchfx/xmlquery"
)

const (
	presentationmlNS = "http://schemas.openxmlformats.org/presentationml/2006/main"
	drawingmlNS      = "http://schemas.openxmlformats.org/drawingml/2006/main"
)

// createSampleXML creates PowerPoint-style XML with scheme color references
func createSampleXML(schemeColors []string) []byte {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString(`<p:sld xmlns:p="` + presentationmlNS + `" xmlns:a="` + drawingmlNS + `">`)

	for _, color := range schemeColors {
		buf.WriteString(`<a:sp><a:schemeClr val="` + color + `"/></a:sp>`)
	}

	buf.WriteString(`</p:sld>`)
	return buf.Bytes()
}

// extractSchemeColors extracts all schemeClr val attributes from XML
func extractSchemeColors(xmlContent []byte) ([]string, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(xmlContent))
	if err != nil {
		return nil, err
	}

	nodes, err := xmlquery.QueryAll(doc, "//*[local-name()='schemeClr']")
	if err != nil {
		return nil, err
	}

	colors := make([]string, 0, len(nodes))
	for _, node := range nodes {
		for _, attr := range node.Attr {
			if attr.Name.Local == "val" {
				colors = append(colors, attr.Value)
				break
			}
		}
	}

	return colors, nil
}

func TestReplaceSchemeColors_BasicReplacement(t *testing.T) {
	t.Run("single replacement", func(t *testing.T) {
		xml := createSampleXML([]string{"accent1"})
		mapping := map[string]string{"accent1": "accent3"}

		result, err := ReplaceSchemeColors(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		colors, err := extractSchemeColors(result)
		if err != nil {
			t.Fatalf("failed to extract colors: %v", err)
		}

		expected := []string{"accent3"}
		if len(colors) != len(expected) || colors[0] != expected[0] {
			t.Errorf("expected %v, got %v", expected, colors)
		}
	})

	t.Run("multiple replacements", func(t *testing.T) {
		xml := createSampleXML([]string{"accent1", "accent5", "dk1"})
		mapping := map[string]string{"accent1": "accent3", "dk1": "lt1"}

		result, err := ReplaceSchemeColors(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		colors, err := extractSchemeColors(result)
		if err != nil {
			t.Fatalf("failed to extract colors: %v", err)
		}

		expected := []string{"accent3", "accent5", "lt1"}
		if len(colors) != len(expected) {
			t.Fatalf("expected %d colors, got %d", len(expected), len(colors))
		}
		for i, exp := range expected {
			if colors[i] != exp {
				t.Errorf("color %d: expected %s, got %s", i, exp, colors[i])
			}
		}
	})

	t.Run("unmapped colors unchanged", func(t *testing.T) {
		xml := createSampleXML([]string{"accent1", "accent2", "accent3"})
		mapping := map[string]string{"accent1": "accent6"}

		result, err := ReplaceSchemeColors(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		colors, err := extractSchemeColors(result)
		if err != nil {
			t.Fatalf("failed to extract colors: %v", err)
		}

		expected := []string{"accent6", "accent2", "accent3"}
		if len(colors) != len(expected) {
			t.Fatalf("expected %d colors, got %d", len(expected), len(colors))
		}
		for i, exp := range expected {
			if colors[i] != exp {
				t.Errorf("color %d: expected %s, got %s", i, exp, colors[i])
			}
		}
	})
}

func TestReplaceSchemeColors_AtomicReplacement(t *testing.T) {
	t.Run("no cascading replacement", func(t *testing.T) {
		// accent1→accent3 and accent3→accent4 should NOT cascade
		// Original: [accent1, accent3]
		// Expected: [accent3, accent4] (NOT [accent4, accent4])
		xml := createSampleXML([]string{"accent1", "accent3"})
		mapping := map[string]string{"accent1": "accent3", "accent3": "accent4"}

		result, err := ReplaceSchemeColors(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		colors, err := extractSchemeColors(result)
		if err != nil {
			t.Fatalf("failed to extract colors: %v", err)
		}

		expected := []string{"accent3", "accent4"}
		if len(colors) != len(expected) {
			t.Fatalf("expected %d colors, got %d", len(expected), len(colors))
		}
		for i, exp := range expected {
			if colors[i] != exp {
				t.Errorf("color %d: expected %s, got %s", i, exp, colors[i])
			}
		}
	})

	t.Run("circular mapping safe", func(t *testing.T) {
		// Even circular mappings should work atomically (they swap)
		xml := createSampleXML([]string{"accent1", "accent2"})
		mapping := map[string]string{"accent1": "accent2", "accent2": "accent1"}

		result, err := ReplaceSchemeColors(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		colors, err := extractSchemeColors(result)
		if err != nil {
			t.Fatalf("failed to extract colors: %v", err)
		}

		expected := []string{"accent2", "accent1"}
		if len(colors) != len(expected) {
			t.Fatalf("expected %d colors, got %d", len(expected), len(colors))
		}
		for i, exp := range expected {
			if colors[i] != exp {
				t.Errorf("color %d: expected %s, got %s", i, exp, colors[i])
			}
		}
	})
}

func TestReplaceSchemeColors_ManyToOne(t *testing.T) {
	t.Run("multiple sources to same target", func(t *testing.T) {
		xml := createSampleXML([]string{"accent1", "accent5", "accent3"})
		mapping := map[string]string{"accent1": "accent3", "accent5": "accent3"}

		result, err := ReplaceSchemeColors(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		colors, err := extractSchemeColors(result)
		if err != nil {
			t.Fatalf("failed to extract colors: %v", err)
		}

		// Both accent1 and accent5 become accent3
		// Original accent3 stays accent3 (no mapping)
		expected := []string{"accent3", "accent3", "accent3"}
		if len(colors) != len(expected) {
			t.Fatalf("expected %d colors, got %d", len(expected), len(colors))
		}
		for i, exp := range expected {
			if colors[i] != exp {
				t.Errorf("color %d: expected %s, got %s", i, exp, colors[i])
			}
		}
	})
}

func TestReplaceSchemeColors_EdgeCases(t *testing.T) {
	t.Run("empty mapping", func(t *testing.T) {
		xml := createSampleXML([]string{"accent1", "accent2"})
		result, err := ReplaceSchemeColors(xml, map[string]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		colors, err := extractSchemeColors(result)
		if err != nil {
			t.Fatalf("failed to extract colors: %v", err)
		}

		expected := []string{"accent1", "accent2"}
		if len(colors) != len(expected) {
			t.Fatalf("expected %d colors, got %d", len(expected), len(colors))
		}
		for i, exp := range expected {
			if colors[i] != exp {
				t.Errorf("color %d: expected %s, got %s", i, exp, colors[i])
			}
		}
	})

	t.Run("invalid xml", func(t *testing.T) {
		invalid := []byte("This is not XML")
		result, err := ReplaceSchemeColors(invalid, map[string]string{"accent1": "accent3"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !bytes.Equal(result, invalid) {
			t.Error("invalid XML should be returned unchanged")
		}
	})

	t.Run("xml without scheme colors", func(t *testing.T) {
		xml := []byte(`<?xml version="1.0" encoding="UTF-8"?><root><child>text</child></root>`)
		result, err := ReplaceSchemeColors(xml, map[string]string{"accent1": "accent3"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should still be valid XML
		doc, err := xmlquery.Parse(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("result should be valid XML: %v", err)
		}

		if doc.SelectElement("root") == nil {
			t.Error("root element should exist")
		}
	})
}

func TestReplaceSchemeColors_ComplexScenario(t *testing.T) {
	t.Run("realistic slide with multiple colors", func(t *testing.T) {
		// Simulate a slide with various elements
		xml := createSampleXML([]string{
			"accent1", // Title
			"accent1", // Subtitle (same as title)
			"accent5", // Shape 1
			"accent3", // Shape 2
			"accent4", // Shape 3
			"dk1",     // Text
			"hlink",   // Hyperlink
		})

		// User's mapping: accent1 and accent5 → accent3, accent3 → accent4
		mapping := map[string]string{
			"accent1": "accent3",
			"accent5": "accent3",
			"accent3": "accent4",
		}

		result, err := ReplaceSchemeColors(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		colors, err := extractSchemeColors(result)
		if err != nil {
			t.Fatalf("failed to extract colors: %v", err)
		}

		expected := []string{
			"accent3", // Title (was accent1)
			"accent3", // Subtitle (was accent1)
			"accent3", // Shape 1 (was accent5)
			"accent4", // Shape 2 (was accent3)
			"accent4", // Shape 3 (unchanged)
			"dk1",     // Text (unchanged)
			"hlink",   // Hyperlink (unchanged)
		}

		if len(colors) != len(expected) {
			t.Fatalf("expected %d colors, got %d", len(expected), len(colors))
		}
		for i, exp := range expected {
			if colors[i] != exp {
				t.Errorf("color %d: expected %s, got %s", i, exp, colors[i])
			}
		}
	})
}

// createSampleXMLWithRgb creates PowerPoint-style XML with RGB color references
func createSampleXMLWithRgb(rgbColors []string) []byte {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString(`<p:sld xmlns:p="` + presentationmlNS + `" xmlns:a="` + drawingmlNS + `">`)

	for _, color := range rgbColors {
		buf.WriteString(`<a:sp><a:solidFill><a:srgbClr val="` + color + `"/></a:solidFill></a:sp>`)
	}

	buf.WriteString(`</p:sld>`)
	return buf.Bytes()
}

// extractSrgbColors extracts all srgbClr val attributes from XML
func extractSrgbColors(xmlContent []byte) ([]string, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(xmlContent))
	if err != nil {
		return nil, err
	}

	nodes, err := xmlquery.QueryAll(doc, "//*[local-name()='srgbClr']")
	if err != nil {
		return nil, err
	}

	colors := make([]string, 0, len(nodes))
	for _, node := range nodes {
		for _, attr := range node.Attr {
			if attr.Name.Local == "val" {
				colors = append(colors, attr.Value)
				break
			}
		}
	}

	return colors, nil
}

func TestReplaceSrgbColors_HexToHex(t *testing.T) {
	t.Run("single replacement", func(t *testing.T) {
		xml := createSampleXMLWithRgb([]string{"AABBCC"})
		mapping := map[string]string{"AABBCC": "FF0000"}

		result, err := ReplaceSrgbColors(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		colors, err := extractSrgbColors(result)
		if err != nil {
			t.Fatalf("failed to extract colors: %v", err)
		}

		expected := []string{"FF0000"}
		if len(colors) != len(expected) || colors[0] != expected[0] {
			t.Errorf("expected %v, got %v", expected, colors)
		}
	})

	t.Run("multiple replacements", func(t *testing.T) {
		xml := createSampleXMLWithRgb([]string{"AABBCC", "FF0000", "00FF00"})
		mapping := map[string]string{
			"AABBCC": "111111",
			"FF0000": "222222",
		}

		result, err := ReplaceSrgbColors(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		colors, err := extractSrgbColors(result)
		if err != nil {
			t.Fatalf("failed to extract colors: %v", err)
		}

		expected := []string{"111111", "222222", "00FF00"}
		if len(colors) != len(expected) {
			t.Fatalf("expected %d colors, got %d", len(expected), len(colors))
		}
		for i, exp := range expected {
			if colors[i] != exp {
				t.Errorf("color %d: expected %s, got %s", i, exp, colors[i])
			}
		}
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		xml := createSampleXMLWithRgb([]string{"aabbcc", "AABBCC", "AaBbCc"})
		mapping := map[string]string{"AABBCC": "FF0000"}

		result, err := ReplaceSrgbColors(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		colors, err := extractSrgbColors(result)
		if err != nil {
			t.Fatalf("failed to extract colors: %v", err)
		}

		// All variants should be replaced and normalized to uppercase
		expected := []string{"FF0000", "FF0000", "FF0000"}
		if len(colors) != len(expected) {
			t.Fatalf("expected %d colors, got %d", len(expected), len(colors))
		}
		for i, exp := range expected {
			if colors[i] != exp {
				t.Errorf("color %d: expected %s, got %s", i, exp, colors[i])
			}
		}
	})
}

func TestReplaceSrgbColors_HexToScheme(t *testing.T) {
	t.Run("single hex to scheme", func(t *testing.T) {
		xml := createSampleXMLWithRgb([]string{"AABBCC"})
		mapping := map[string]string{"AABBCC": "accent1"}

		result, err := ReplaceSrgbColors(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// After hex→scheme conversion, srgbClr should be replaced with schemeClr
		colors, err := extractSchemeColors(result)
		if err != nil {
			t.Fatalf("failed to extract scheme colors: %v", err)
		}

		expected := []string{"accent1"}
		if len(colors) != len(expected) || colors[0] != expected[0] {
			t.Errorf("expected %v, got %v", expected, colors)
		}

		// Should no longer have srgbClr elements
		rgbColors, _ := extractSrgbColors(result)
		if len(rgbColors) != 0 {
			t.Errorf("expected no srgbClr elements, but found %d", len(rgbColors))
		}
	})
}

func TestReplaceSchemeColorsWithSrgb_SchemeToHex(t *testing.T) {
	t.Run("single scheme to hex", func(t *testing.T) {
		xml := createSampleXML([]string{"accent1"})
		mapping := map[string]string{"accent1": "BBFFCC"}

		result, err := ReplaceSchemeColorsWithSrgb(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// After scheme→hex conversion, schemeClr should be replaced with srgbClr
		colors, err := extractSrgbColors(result)
		if err != nil {
			t.Fatalf("failed to extract srgb colors: %v", err)
		}

		expected := []string{"BBFFCC"}
		if len(colors) != len(expected) || colors[0] != expected[0] {
			t.Errorf("expected %v, got %v", expected, colors)
		}

		// Should no longer have schemeClr elements for this color
		schemeColors, _ := extractSchemeColors(result)
		if len(schemeColors) != 0 {
			t.Errorf("expected no schemeClr elements, but found %d", len(schemeColors))
		}
	})

	t.Run("multiple scheme to hex", func(t *testing.T) {
		xml := createSampleXML([]string{"accent1", "accent2", "accent3"})
		mapping := map[string]string{
			"accent1": "BBFFCC",
			"accent3": "FF0000",
		}

		result, err := ReplaceSchemeColorsWithSrgb(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// accent1 and accent3 should become srgbClr
		rgbColors, err := extractSrgbColors(result)
		if err != nil {
			t.Fatalf("failed to extract srgb colors: %v", err)
		}

		expectedRgb := []string{"BBFFCC", "FF0000"}
		if len(rgbColors) != len(expectedRgb) {
			t.Fatalf("expected %d rgb colors, got %d", len(expectedRgb), len(rgbColors))
		}

		// accent2 should remain as schemeClr
		schemeColors, _ := extractSchemeColors(result)
		if len(schemeColors) != 1 || schemeColors[0] != "accent2" {
			t.Errorf("expected [accent2] schemeClr, got %v", schemeColors)
		}
	})
}

func TestReplaceSrgbColors_AtomicReplacement(t *testing.T) {
	t.Run("no cascading replacement", func(t *testing.T) {
		xml := createSampleXMLWithRgb([]string{"AABBCC", "FF0000"})
		mapping := map[string]string{
			"AABBCC": "FF0000",
			"FF0000": "00FF00",
		}

		result, err := ReplaceSrgbColors(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		colors, err := extractSrgbColors(result)
		if err != nil {
			t.Fatalf("failed to extract colors: %v", err)
		}

		// AABBCC→FF0000, FF0000→00FF00 (NOT AABBCC→00FF00)
		expected := []string{"FF0000", "00FF00"}
		if len(colors) != len(expected) {
			t.Fatalf("expected %d colors, got %d", len(expected), len(colors))
		}
		for i, exp := range expected {
			if colors[i] != exp {
				t.Errorf("color %d: expected %s, got %s", i, exp, colors[i])
			}
		}
	})
}

func TestReplaceSrgbColors_EmptyMapping(t *testing.T) {
	xml := createSampleXMLWithRgb([]string{"AABBCC", "FF0000"})
	result, err := ReplaceSrgbColors(xml, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(result, xml) {
		t.Error("empty mapping should return unchanged XML")
	}
}

func TestReplaceSrgbColors_NoMatches(t *testing.T) {
	xml := createSampleXMLWithRgb([]string{"AABBCC", "FF0000"})
	mapping := map[string]string{"123456": "FEDCBA"}

	result, err := ReplaceSrgbColors(xml, mapping)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	colors, err := extractSrgbColors(result)
	if err != nil {
		t.Fatalf("failed to extract colors: %v", err)
	}

	expected := []string{"AABBCC", "FF0000"}
	if len(colors) != len(expected) {
		t.Fatalf("expected %d colors, got %d", len(expected), len(colors))
	}
	for i, exp := range expected {
		if colors[i] != exp {
			t.Errorf("color %d: expected %s, got %s", i, exp, colors[i])
		}
	}
}

func TestReplaceSchemeColorsWithSrgb_WithTintModifiers(t *testing.T) {
	t.Run("scheme to hex with tint modifiers - strips children", func(t *testing.T) {
		// Create XML with tint/shade modifiers (container elements with children)
		xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>` +
			`<p:sld xmlns:p="` + presentationmlNS + `" xmlns:a="` + drawingmlNS + `">` +
			`<a:solidFill>` +
			`<a:schemeClr val="accent1">` +
			`<a:lumMod val="20000"/>` +
			`<a:lumOff val="80000"/>` +
			`</a:schemeClr>` +
			`</a:solidFill>` +
			`</p:sld>`)

		mapping := map[string]string{"accent1": "FF00FF"}

		result, err := ReplaceSchemeColorsWithSrgb(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify conversion to srgbClr
		rgbColors, err := extractSrgbColors(result)
		if err != nil {
			t.Fatalf("failed to extract srgb colors: %v", err)
		}

		if len(rgbColors) != 1 || rgbColors[0] != "FF00FF" {
			t.Errorf("expected [FF00FF], got %v", rgbColors)
		}

		// Verify no schemeClr elements remain for accent1
		schemeColors, _ := extractSchemeColors(result)
		if len(schemeColors) != 0 {
			t.Errorf("expected no schemeClr elements, but found %d: %v", len(schemeColors), schemeColors)
		}

		// Verify XML structure is valid (no mismatched tags)
		doc, err := xmlquery.Parse(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("result should be valid XML: %v", err)
		}

		// Verify the element is self-closing (no children)
		srgbNode := xmlquery.FindOne(doc, "//*[local-name()='srgbClr']")
		if srgbNode == nil {
			t.Fatal("srgbClr element not found")
		}

		// Check that srgbClr has no children (modifiers should be stripped)
		if srgbNode.FirstChild != nil {
			t.Errorf("srgbClr should have no children, but has: %v", srgbNode.FirstChild)
		}
	})

	t.Run("scheme to hex with multiple tint variants", func(t *testing.T) {
		// Simulate multiple tint variants of the same color (like PowerPoint's color picker)
		xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>` +
			`<p:sld xmlns:p="` + presentationmlNS + `" xmlns:a="` + drawingmlNS + `">` +
			`<a:sp><a:solidFill><a:schemeClr val="accent1"/></a:solidFill></a:sp>` + // BASE
			`<a:sp><a:solidFill><a:schemeClr val="accent1"><a:lumMod val="20000"/><a:lumOff val="80000"/></a:schemeClr></a:solidFill></a:sp>` + // L80
			`<a:sp><a:solidFill><a:schemeClr val="accent1"><a:lumMod val="40000"/><a:lumOff val="60000"/></a:schemeClr></a:solidFill></a:sp>` + // L60
			`<a:sp><a:solidFill><a:schemeClr val="accent1"><a:lumMod val="75000"/></a:schemeClr></a:solidFill></a:sp>` + // D25
			`</p:sld>`)

		mapping := map[string]string{"accent1": "FF00FF"}

		result, err := ReplaceSchemeColorsWithSrgb(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// All 4 variants should become srgbClr with FF00FF (tints stripped)
		rgbColors, err := extractSrgbColors(result)
		if err != nil {
			t.Fatalf("failed to extract srgb colors: %v", err)
		}

		expected := []string{"FF00FF", "FF00FF", "FF00FF", "FF00FF"}
		if len(rgbColors) != len(expected) {
			t.Fatalf("expected %d rgb colors, got %d", len(expected), len(rgbColors))
		}
		for i, exp := range expected {
			if rgbColors[i] != exp {
				t.Errorf("color %d: expected %s, got %s", i, exp, rgbColors[i])
			}
		}

		// Verify XML is valid
		doc, err := xmlquery.Parse(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("result should be valid XML: %v", err)
		}

		// Verify all srgbClr elements have no children
		srgbNodes, _ := xmlquery.QueryAll(doc, "//*[local-name()='srgbClr']")
		for i, node := range srgbNodes {
			if node.FirstChild != nil {
				t.Errorf("srgbClr element %d should have no children", i)
			}
		}
	})

	t.Run("scheme to hex preserves self-closing tags", func(t *testing.T) {
		// Self-closing tags (no tint modifiers) should still work
		xml := createSampleXML([]string{"accent1", "accent2"})
		mapping := map[string]string{"accent1": "BBFFCC"}

		result, err := ReplaceSchemeColorsWithSrgb(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// accent1 should become srgbClr
		rgbColors, err := extractSrgbColors(result)
		if err != nil {
			t.Fatalf("failed to extract srgb colors: %v", err)
		}

		if len(rgbColors) != 1 || rgbColors[0] != "BBFFCC" {
			t.Errorf("expected [BBFFCC], got %v", rgbColors)
		}

		// accent2 should remain schemeClr
		schemeColors, _ := extractSchemeColors(result)
		if len(schemeColors) != 1 || schemeColors[0] != "accent2" {
			t.Errorf("expected [accent2], got %v", schemeColors)
		}
	})

	t.Run("scheme to scheme preserves tint modifiers", func(t *testing.T) {
		// When converting scheme→scheme, tint modifiers should be preserved
		xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>` +
			`<p:sld xmlns:p="` + presentationmlNS + `" xmlns:a="` + drawingmlNS + `">` +
			`<a:solidFill>` +
			`<a:schemeClr val="accent1">` +
			`<a:lumMod val="75000"/>` +
			`</a:schemeClr>` +
			`</a:solidFill>` +
			`</p:sld>`)

		mapping := map[string]string{"accent1": "accent3"}

		result, err := ReplaceSchemeColorsWithSrgb(xml, mapping)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should still be schemeClr (not srgbClr)
		schemeColors, err := extractSchemeColors(result)
		if err != nil {
			t.Fatalf("failed to extract scheme colors: %v", err)
		}

		if len(schemeColors) != 1 || schemeColors[0] != "accent3" {
			t.Errorf("expected [accent3], got %v", schemeColors)
		}

		// Verify lumMod modifier is preserved
		if !bytes.Contains(result, []byte("lumMod")) {
			t.Error("expected lumMod modifier to be preserved for scheme→scheme conversion")
		}

		// Verify XML structure is valid
		doc, err := xmlquery.Parse(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("result should be valid XML: %v", err)
		}

		// Verify the schemeClr element has children (modifiers preserved)
		schemeNode := xmlquery.FindOne(doc, "//*[local-name()='schemeClr']")
		if schemeNode == nil {
			t.Fatal("schemeClr element not found")
		}

		lumModNode := xmlquery.FindOne(schemeNode, "//*[local-name()='lumMod']")
		if lumModNode == nil {
			t.Error("lumMod child element should be preserved for scheme→scheme conversion")
		}
	})
}
