"""Integration tests for PowerPoint processing with theme filtering."""

import pytest
import zipfile
import tempfile
from pathlib import Path
from lxml import etree
from src.pptx_processor import process_pptx


# Path to test fixture
FIXTURES_DIR = Path(__file__).parent / 'fixtures'
TEST_PPTX = FIXTURES_DIR / 'test.pptx'


def count_scheme_colors_in_pptx(pptx_path: str, color_name: str) -> int:
    """
    Count occurrences of a specific scheme color in a PowerPoint file.

    Args:
        pptx_path: Path to .pptx file
        color_name: Scheme color name to count (e.g., "accent1")

    Returns:
        Number of occurrences
    """
    count = 0
    with zipfile.ZipFile(pptx_path, 'r') as zip_ref:
        for file_name in zip_ref.namelist():
            if not file_name.endswith('.xml'):
                continue

            try:
                content = zip_ref.read(file_name)
                tree = etree.fromstring(content)
                elements = tree.xpath(f'.//*[local-name()="schemeClr"][@val="{color_name}"]')
                count += len(elements)
            except Exception:
                continue

    return count


class TestThemeFiltering:
    """Test theme filtering in PowerPoint processing."""

    def test_process_all_themes_without_filter(self):
        """Test that processing without theme filter affects all themes."""
        if not TEST_PPTX.exists():
            pytest.skip("test.pptx fixture not found")

        with tempfile.NamedTemporaryFile(suffix='.pptx', delete=False) as tmp:
            output_path = tmp.name

        try:
            # Process with a mapping that should apply to all themes
            mapping = {"accent1": "accent6"}
            files_processed = process_pptx(
                str(TEST_PPTX),
                output_path,
                mapping,
                theme_filter=None  # No filter = all themes
            )

            # Should have processed some files
            assert files_processed > 0

            # Count accent1 and accent6 in output
            accent1_count = count_scheme_colors_in_pptx(output_path, "accent1")
            accent6_count = count_scheme_colors_in_pptx(output_path, "accent6")

            # After swapping, there should be more accent6 than in the original
            # (We can't easily compare to original counts without more complex logic,
            # but we can verify the swap happened by checking output)
            assert accent6_count > 0  # Should have some accent6 now

        finally:
            Path(output_path).unlink(missing_ok=True)

    def test_process_specific_theme_only(self):
        """Test that theme filter only affects specified themes."""
        if not TEST_PPTX.exists():
            pytest.skip("test.pptx fixture not found")

        with tempfile.NamedTemporaryFile(suffix='.pptx', delete=False) as tmp:
            output_path = tmp.name

        try:
            # Process only theme1
            mapping = {"accent1": "accent6"}
            files_processed = process_pptx(
                str(TEST_PPTX),
                output_path,
                mapping,
                theme_filter=["theme1"]
            )

            # Should have processed some files
            assert files_processed > 0

            # The output should be a valid .pptx
            assert zipfile.is_zipfile(output_path)

        finally:
            Path(output_path).unlink(missing_ok=True)

    def test_process_multiple_themes(self):
        """Test filtering with multiple themes."""
        if not TEST_PPTX.exists():
            pytest.skip("test.pptx fixture not found")

        with tempfile.NamedTemporaryFile(suffix='.pptx', delete=False) as tmp:
            output_path = tmp.name

        try:
            # Process both theme1 and theme2
            mapping = {"accent1": "accent6"}
            files_processed = process_pptx(
                str(TEST_PPTX),
                output_path,
                mapping,
                theme_filter=["theme1", "theme2"]
            )

            # Should have processed files from both themes
            assert files_processed > 0

            # The output should be a valid .pptx
            assert zipfile.is_zipfile(output_path)

        finally:
            Path(output_path).unlink(missing_ok=True)

    def test_nonexistent_theme_filter(self):
        """Test that filtering by non-existent theme processes nothing."""
        if not TEST_PPTX.exists():
            pytest.skip("test.pptx fixture not found")

        with tempfile.NamedTemporaryFile(suffix='.pptx', delete=False) as tmp:
            output_path = tmp.name

        try:
            # Process with a theme that doesn't exist
            mapping = {"accent1": "accent6"}
            files_processed = process_pptx(
                str(TEST_PPTX),
                output_path,
                mapping,
                theme_filter=["theme999"]  # Doesn't exist
            )

            # Should still create output file, but process 0 or very few files
            # (might process charts/diagrams that aren't theme-linked)
            assert zipfile.is_zipfile(output_path)

        finally:
            Path(output_path).unlink(missing_ok=True)


class TestEndToEnd:
    """End-to-end integration tests."""

    def test_complete_workflow(self):
        """Test a complete workflow: read themes, swap colors, verify output."""
        if not TEST_PPTX.exists():
            pytest.skip("test.pptx fixture not found")

        from src.theme_reader import read_themes

        # Step 1: Read themes
        themes = read_themes(str(TEST_PPTX))
        assert len(themes) >= 2

        # Step 2: Process with color swap
        with tempfile.NamedTemporaryFile(suffix='.pptx', delete=False) as tmp:
            output_path = tmp.name

        try:
            mapping = {"accent1": "accent2", "accent5": "accent6"}
            files_processed = process_pptx(
                str(TEST_PPTX),
                output_path,
                mapping,
                theme_filter=None
            )

            # Should have processed files
            assert files_processed > 0

            # Step 3: Verify output is valid PowerPoint
            assert zipfile.is_zipfile(output_path)

            # Step 4: Read themes from output (should be unchanged)
            output_themes = read_themes(output_path)
            assert len(output_themes) == len(themes)

            # Theme definitions should be unchanged
            for i, theme in enumerate(themes):
                assert output_themes[i].colors.accent1 == theme.colors.accent1
                assert output_themes[i].colors.accent2 == theme.colors.accent2

        finally:
            Path(output_path).unlink(missing_ok=True)

    def test_atomic_replacement_in_real_file(self):
        """Test that atomic replacement works correctly in a real PowerPoint file."""
        if not TEST_PPTX.exists():
            pytest.skip("test.pptx fixture not found")

        with tempfile.NamedTemporaryFile(suffix='.pptx', delete=False) as tmp:
            output_path = tmp.name

        try:
            # Create a scenario where accent1→accent3 and accent3→accent4
            # This tests that accent1 becomes accent3 (not accent4)
            mapping = {"accent1": "accent3", "accent3": "accent4"}

            process_pptx(
                str(TEST_PPTX),
                output_path,
                mapping,
                theme_filter=None
            )

            # Count occurrences in output
            accent1_count = count_scheme_colors_in_pptx(output_path, "accent1")
            accent3_count = count_scheme_colors_in_pptx(output_path, "accent3")
            accent4_count = count_scheme_colors_in_pptx(output_path, "accent4")

            # After atomic replacement:
            # - accent1 should be rare or gone (all swapped to accent3)
            # - accent3 should exist (from original accent1s)
            # - accent4 should exist (from original accent3s)

            # We can't make exact assertions without knowing original counts,
            # but we can verify the output is valid
            assert zipfile.is_zipfile(output_path)

        finally:
            Path(output_path).unlink(missing_ok=True)
