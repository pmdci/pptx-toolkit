"""Tests for XML scheme color replacement."""

import pytest
from lxml import etree
from src.xml_replacer import replace_scheme_colors


# Sample PowerPoint DrawingML namespace
DRAWINGML_NS = "http://schemas.openxmlformats.org/drawingml/2006/main"


def create_sample_xml(scheme_colors: list[str]) -> bytes:
    """Create sample PowerPoint-style XML with scheme color references."""
    root = etree.Element(
        "{http://schemas.openxmlformats.org/presentationml/2006/main}sld",
        nsmap={
            'p': 'http://schemas.openxmlformats.org/presentationml/2006/main',
            'a': DRAWINGML_NS
        }
    )

    for i, color in enumerate(scheme_colors):
        shape = etree.SubElement(root, f"{{{DRAWINGML_NS}}}sp")
        scheme_clr = etree.SubElement(shape, f"{{{DRAWINGML_NS}}}schemeClr")
        scheme_clr.set('val', color)

    return etree.tostring(root, xml_declaration=True, encoding='UTF-8')


def extract_scheme_colors(xml_content: bytes) -> list[str]:
    """Extract all schemeClr val attributes from XML."""
    tree = etree.fromstring(xml_content)
    elements = tree.xpath('.//*[local-name()="schemeClr"]')
    return [elem.get('val') for elem in elements]


class TestBasicReplacement:
    """Test basic color replacement."""

    def test_single_replacement(self):
        xml = create_sample_xml(['accent1'])
        mapping = {'accent1': 'accent3'}

        result = replace_scheme_colors(xml, mapping)
        colors = extract_scheme_colors(result)

        assert colors == ['accent3']

    def test_multiple_replacements(self):
        xml = create_sample_xml(['accent1', 'accent5', 'dk1'])
        mapping = {'accent1': 'accent3', 'dk1': 'lt1'}

        result = replace_scheme_colors(xml, mapping)
        colors = extract_scheme_colors(result)

        assert colors == ['accent3', 'accent5', 'lt1']

    def test_unmapped_colors_unchanged(self):
        """Colors not in mapping should remain unchanged."""
        xml = create_sample_xml(['accent1', 'accent2', 'accent3'])
        mapping = {'accent1': 'accent6'}

        result = replace_scheme_colors(xml, mapping)
        colors = extract_scheme_colors(result)

        assert colors == ['accent6', 'accent2', 'accent3']


class TestAtomicReplacement:
    """Test that replacements are atomic, not cascading."""

    def test_no_cascading_replacement(self):
        """accent1→accent3 and accent3→accent4 should not cascade."""
        # Original: accent1, accent3
        # Expected: accent3, accent4
        # NOT: accent4, accent4 (which would happen with cascading)

        xml = create_sample_xml(['accent1', 'accent3'])
        mapping = {'accent1': 'accent3', 'accent3': 'accent4'}

        result = replace_scheme_colors(xml, mapping)
        colors = extract_scheme_colors(result)

        # First element was accent1, should become accent3 (not accent4)
        # Second element was accent3, should become accent4
        assert colors == ['accent3', 'accent4']

    def test_circular_mapping_safe(self):
        """Even circular mappings should work atomically."""
        xml = create_sample_xml(['accent1', 'accent2'])
        mapping = {'accent1': 'accent2', 'accent2': 'accent1'}

        result = replace_scheme_colors(xml, mapping)
        colors = extract_scheme_colors(result)

        # They should swap
        assert colors == ['accent2', 'accent1']


class TestManyToOne:
    """Test many-to-one mappings."""

    def test_multiple_sources_to_same_target(self):
        xml = create_sample_xml(['accent1', 'accent5', 'accent3'])
        mapping = {'accent1': 'accent3', 'accent5': 'accent3'}

        result = replace_scheme_colors(xml, mapping)
        colors = extract_scheme_colors(result)

        # Both accent1 and accent5 should become accent3
        # Original accent3 stays accent3 (no mapping for it)
        assert colors == ['accent3', 'accent3', 'accent3']


class TestEdgeCases:
    """Test edge cases and error handling."""

    def test_empty_mapping(self):
        """Empty mapping should leave XML unchanged."""
        xml = create_sample_xml(['accent1', 'accent2'])
        result = replace_scheme_colors(xml, {})
        colors = extract_scheme_colors(result)

        assert colors == ['accent1', 'accent2']

    def test_invalid_xml(self):
        """Non-XML content should be returned unchanged."""
        invalid_xml = b"This is not XML"
        result = replace_scheme_colors(invalid_xml, {'accent1': 'accent3'})

        assert result == invalid_xml

    def test_xml_without_scheme_colors(self):
        """XML without schemeClr elements should be unchanged."""
        xml = b'<?xml version="1.0" encoding="UTF-8"?><root><child>text</child></root>'
        result = replace_scheme_colors(xml, {'accent1': 'accent3'})

        # Should still be valid XML
        tree = etree.fromstring(result)
        assert tree.tag == 'root'


class TestComplexScenario:
    """Test realistic complex scenarios."""

    def test_realistic_slide_with_multiple_colors(self):
        """Test a scenario similar to the user's use case."""
        # Simulate a slide with various elements using different colors
        xml = create_sample_xml([
            'accent1',  # Title
            'accent1',  # Subtitle (same color as title)
            'accent5',  # Shape 1
            'accent3',  # Shape 2
            'accent4',  # Shape 3
            'dk1',      # Text
            'hlink'     # Hyperlink
        ])

        # User's mapping: accent1 and accent5 → accent3, accent3 → accent4
        mapping = {
            'accent1': 'accent3',
            'accent5': 'accent3',
            'accent3': 'accent4'
        }

        result = replace_scheme_colors(xml, mapping)
        colors = extract_scheme_colors(result)

        expected = [
            'accent3',  # Title (was accent1)
            'accent3',  # Subtitle (was accent1)
            'accent3',  # Shape 1 (was accent5)
            'accent4',  # Shape 2 (was accent3)
            'accent4',  # Shape 3 (unchanged)
            'dk1',      # Text (unchanged)
            'hlink'     # Hyperlink (unchanged)
        ]

        assert colors == expected
