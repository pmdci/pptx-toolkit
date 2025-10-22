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
