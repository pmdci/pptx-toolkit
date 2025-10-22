"""Tests for CLI parser and validation."""

import pytest
from src.cli_parser import parse_color_mapping, ColorMappingError


class TestBasicParsing:
    """Test basic mapping string parsing."""

    def test_simple_mapping(self):
        result = parse_color_mapping("accent1:accent3")
        assert result == {"accent1": "accent3"}

    def test_multiple_mappings(self):
        result = parse_color_mapping("accent1:accent3,accent5:accent6")
        assert result == {"accent1": "accent3", "accent5": "accent6"}

    def test_many_to_one_mapping(self):
        """Multiple sources can map to same target."""
        result = parse_color_mapping("accent1:accent3,accent5:accent3")
        assert result == {"accent1": "accent3", "accent5": "accent3"}

    def test_whitespace_handling(self):
        result = parse_color_mapping(" accent1 : accent3 , accent5 : accent6 ")
        assert result == {"accent1": "accent3", "accent5": "accent6"}


class TestConflictDetection:
    """Test that conflicting mappings are detected."""

    def test_conflicting_mappings_rejected(self):
        """Same source with different targets should fail."""
        with pytest.raises(ColorMappingError) as exc:
            parse_color_mapping("accent1:accent3,accent1:accent2")

        assert "Conflicting mappings for 'accent1'" in str(exc.value)
        assert "accent3" in str(exc.value)
        assert "accent2" in str(exc.value)

    def test_duplicate_identical_mapping_allowed(self):
        """Duplicate identical mappings should be ignored."""
        result = parse_color_mapping("accent1:accent3,accent1:accent3")
        assert result == {"accent1": "accent3"}


class TestColorValidation:
    """Test valid and invalid color names."""

    def test_all_valid_scheme_colors(self):
        """Test all valid scheme color types."""
        # Text/background colors
        result = parse_color_mapping("dk1:lt1,dk2:lt2")
        assert result == {"dk1": "lt1", "dk2": "lt2"}

        # Accent colors
        result = parse_color_mapping("accent1:accent2,accent3:accent4,accent5:accent6")
        assert len(result) == 3

        # Hyperlink colors
        result = parse_color_mapping("hlink:folHlink")
        assert result == {"hlink": "folHlink"}

    def test_invalid_source_color(self):
        with pytest.raises(ColorMappingError) as exc:
            parse_color_mapping("invalid:accent3")

        assert "Invalid source color: 'invalid'" in str(exc.value)

    def test_invalid_target_color(self):
        with pytest.raises(ColorMappingError) as exc:
            parse_color_mapping("accent1:invalid")

        assert "Invalid target color: 'invalid'" in str(exc.value)


class TestAtomicMapping:
    """Test that mappings represent atomic transformations."""

    def test_swap_scenario(self):
        """Test accent1→accent3 and accent3→accent4 are independent."""
        result = parse_color_mapping("accent1:accent3,accent3:accent4")

        # Both mappings should be present
        assert result["accent1"] == "accent3"
        assert result["accent3"] == "accent4"

        # This proves the mapping is a lookup table,
        # not a chain (accent1 won't become accent4)


class TestInvalidInput:
    """Test error handling for malformed input."""

    def test_empty_string(self):
        with pytest.raises(ColorMappingError) as exc:
            parse_color_mapping("")

        assert "cannot be empty" in str(exc.value)

    def test_whitespace_only(self):
        with pytest.raises(ColorMappingError) as exc:
            parse_color_mapping("   ")

        assert "cannot be empty" in str(exc.value)

    def test_missing_colon(self):
        with pytest.raises(ColorMappingError) as exc:
            parse_color_mapping("accent1")

        assert "Invalid mapping format" in str(exc.value)
        assert "Expected 'source:target'" in str(exc.value)

    def test_multiple_colons(self):
        with pytest.raises(ColorMappingError) as exc:
            parse_color_mapping("accent1:accent2:accent3")

        assert "Invalid mapping format" in str(exc.value)

    def test_empty_source(self):
        with pytest.raises(ColorMappingError) as exc:
            parse_color_mapping(":accent3")

        assert "cannot be empty" in str(exc.value)

    def test_empty_target(self):
        with pytest.raises(ColorMappingError) as exc:
            parse_color_mapping("accent1:")

        assert "cannot be empty" in str(exc.value)

    def test_no_valid_mappings(self):
        with pytest.raises(ColorMappingError) as exc:
            parse_color_mapping(",,,")

        assert "No valid mappings found" in str(exc.value)
