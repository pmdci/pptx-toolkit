"""Tests for theme reading functionality."""

import pytest
from pathlib import Path
from src.theme_reader import read_themes, parse_theme_xml, extract_rgb_color
from lxml import etree


# Path to test fixture
FIXTURES_DIR = Path(__file__).parent / 'fixtures'
TEST_PPTX = FIXTURES_DIR / 'test.pptx'


class TestReadThemes:
    """Test reading themes from PowerPoint files."""

    def test_read_themes_from_test_file(self):
        """Test reading themes from the test.pptx fixture."""
        if not TEST_PPTX.exists():
            pytest.skip("test.pptx fixture not found")

        themes = read_themes(str(TEST_PPTX))

        # Should have 2 themes
        assert len(themes) == 2

        # Check theme1
        theme1 = themes[0]
        assert theme1.file_name == "theme1.xml"
        assert "Office" in theme1.theme_name or "Office" in theme1.color_scheme_name

        # Check theme2
        theme2 = themes[1]
        assert theme2.file_name == "theme2.xml"
        assert "Blue" in theme2.theme_name or "Blue" in theme2.color_scheme_name

    def test_theme_has_all_colors(self):
        """Test that themes have all required colors."""
        if not TEST_PPTX.exists():
            pytest.skip("test.pptx fixture not found")

        themes = read_themes(str(TEST_PPTX))
        assert len(themes) > 0

        theme = themes[0]
        colors = theme.colors

        # All colors should be present (6-character hex strings)
        assert len(colors.dk1) == 6
        assert len(colors.lt1) == 6
        assert len(colors.dk2) == 6
        assert len(colors.lt2) == 6
        assert len(colors.accent1) == 6
        assert len(colors.accent2) == 6
        assert len(colors.accent3) == 6
        assert len(colors.accent4) == 6
        assert len(colors.accent5) == 6
        assert len(colors.accent6) == 6
        assert len(colors.hlink) == 6
        assert len(colors.fol_hlink) == 6

        # All should be valid hex
        all_colors = [
            colors.dk1, colors.lt1, colors.dk2, colors.lt2,
            colors.accent1, colors.accent2, colors.accent3,
            colors.accent4, colors.accent5, colors.accent6,
            colors.hlink, colors.fol_hlink
        ]

        for color in all_colors:
            # Should be valid hex (no exception)
            int(color, 16)

    def test_themes_have_different_colors(self):
        """Test that different themes have different color schemes."""
        if not TEST_PPTX.exists():
            pytest.skip("test.pptx fixture not found")

        themes = read_themes(str(TEST_PPTX))
        assert len(themes) >= 2

        theme1_accent1 = themes[0].colors.accent1
        theme2_accent1 = themes[1].colors.accent1

        # The two themes should have different accent1 colors
        assert theme1_accent1 != theme2_accent1


class TestExtractRGBColor:
    """Test RGB color extraction from XML elements."""

    def test_extract_srgb_color(self):
        """Test extracting sRGB color value."""
        xml = '<a:accent1 xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:srgbClr val="156082"/></a:accent1>'
        element = etree.fromstring(xml.encode())
        color = extract_rgb_color(element)
        assert color == "156082"

    def test_extract_sys_color(self):
        """Test extracting system color value."""
        xml = '<a:dk1 xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:sysClr val="windowText" lastClr="000000"/></a:dk1>'
        element = etree.fromstring(xml.encode())
        color = extract_rgb_color(element)
        assert color == "000000"

    def test_extract_color_returns_none_for_invalid(self):
        """Test that invalid color elements return None."""
        xml = '<a:accent1 xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"></a:accent1>'
        element = etree.fromstring(xml.encode())
        color = extract_rgb_color(element)
        assert color is None


class TestParseThemeXML:
    """Test theme XML parsing."""

    def test_parse_minimal_theme(self):
        """Test parsing a minimal theme XML."""
        xml = b'''<?xml version="1.0" encoding="UTF-8"?>
<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="Test Theme">
    <a:themeElements>
        <a:clrScheme name="Test Colors">
            <a:dk1><a:srgbClr val="000000"/></a:dk1>
            <a:lt1><a:srgbClr val="FFFFFF"/></a:lt1>
            <a:dk2><a:srgbClr val="111111"/></a:dk2>
            <a:lt2><a:srgbClr val="EEEEEE"/></a:lt2>
            <a:accent1><a:srgbClr val="FF0000"/></a:accent1>
            <a:accent2><a:srgbClr val="00FF00"/></a:accent2>
            <a:accent3><a:srgbClr val="0000FF"/></a:accent3>
            <a:accent4><a:srgbClr val="FFFF00"/></a:accent4>
            <a:accent5><a:srgbClr val="FF00FF"/></a:accent5>
            <a:accent6><a:srgbClr val="00FFFF"/></a:accent6>
            <a:hlink><a:srgbClr val="0000EE"/></a:hlink>
            <a:folHlink><a:srgbClr val="800080"/></a:folHlink>
        </a:clrScheme>
    </a:themeElements>
</a:theme>'''

        theme = parse_theme_xml(xml, "test.xml")

        assert theme is not None
        assert theme.file_name == "test.xml"
        assert theme.theme_name == "Test Theme"
        assert theme.color_scheme_name == "Test Colors"
        assert theme.colors.accent1 == "FF0000"
        assert theme.colors.accent2 == "00FF00"
        assert theme.colors.hlink == "0000EE"

    def test_parse_invalid_xml_returns_none(self):
        """Test that invalid XML returns None."""
        xml = b"This is not XML"
        theme = parse_theme_xml(xml, "test.xml")
        assert theme is None

    def test_parse_xml_without_color_scheme_returns_none(self):
        """Test that XML without color scheme returns None."""
        xml = b'''<?xml version="1.0" encoding="UTF-8"?>
<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="Test">
    <a:themeElements>
    </a:themeElements>
</a:theme>'''

        theme = parse_theme_xml(xml, "test.xml")
        assert theme is None
