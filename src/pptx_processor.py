"""Process PowerPoint files to replace scheme colors."""

import os
import zipfile
import tempfile
from pathlib import Path
from lxml import etree
from .xml_replacer import replace_scheme_colors


RELS_NS = "http://schemas.openxmlformats.org/package/2006/relationships"


def build_theme_relationships(extract_path: Path) -> dict[str, str]:
    """
    Build a mapping of slide masters to their themes.

    Args:
        extract_path: Path to extracted .pptx contents

    Returns:
        Dict mapping slideMaster file names to theme file names
        e.g., {"slideMaster1.xml": "theme1.xml", "slideMaster2.xml": "theme2.xml"}
    """
    mapping = {}
    slide_masters_dir = extract_path / 'ppt' / 'slideMasters' / '_rels'

    if not slide_masters_dir.exists():
        return mapping

    for rels_file in slide_masters_dir.glob('slideMaster*.xml.rels'):
        master_name = rels_file.stem.replace('.xml', '')  # e.g., "slideMaster1"

        try:
            tree = etree.parse(str(rels_file))
            # Find the theme relationship
            theme_rels = tree.xpath(
                f'//r:Relationship[@Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme"]',
                namespaces={'r': RELS_NS}
            )

            if theme_rels:
                theme_target = theme_rels[0].get('Target')
                # theme_target is like "../theme/theme1.xml"
                theme_name = theme_target.split('/')[-1]  # Extract "theme1.xml"
                mapping[f"{master_name}.xml"] = theme_name

        except Exception:
            continue

    return mapping


def build_layout_to_master_mapping(extract_path: Path) -> dict[str, str]:
    """
    Build a mapping of slide layouts to their slide masters.

    Args:
        extract_path: Path to extracted .pptx contents

    Returns:
        Dict mapping slideLayout file names to slideMaster file names
        e.g., {"slideLayout1.xml": "slideMaster1.xml"}
    """
    mapping = {}
    layouts_dir = extract_path / 'ppt' / 'slideLayouts' / '_rels'

    if not layouts_dir.exists():
        return mapping

    for rels_file in layouts_dir.glob('slideLayout*.xml.rels'):
        layout_name = rels_file.stem.replace('.xml', '')  # e.g., "slideLayout1"

        try:
            tree = etree.parse(str(rels_file))
            # Find the slideMaster relationship
            master_rels = tree.xpath(
                f'//r:Relationship[@Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster"]',
                namespaces={'r': RELS_NS}
            )

            if master_rels:
                master_target = master_rels[0].get('Target')
                # master_target is like "../slideMasters/slideMaster1.xml"
                master_name = master_target.split('/')[-1]  # Extract "slideMaster1.xml"
                mapping[f"{layout_name}.xml"] = master_name

        except Exception:
            continue

    return mapping


def get_slide_theme(slide_path: Path, extract_path: Path, layout_to_master: dict, master_to_theme: dict) -> str | None:
    """
    Determine which theme a slide uses.

    Args:
        slide_path: Path to the slide XML file
        extract_path: Path to extracted .pptx contents
        layout_to_master: Mapping of layouts to masters
        master_to_theme: Mapping of masters to themes

    Returns:
        Theme file name (e.g., "theme1.xml") or None if not found
    """
    slide_name = slide_path.name
    rels_file = slide_path.parent / '_rels' / f'{slide_name}.rels'

    if not rels_file.exists():
        return None

    try:
        tree = etree.parse(str(rels_file))
        # Find the slideLayout relationship
        layout_rels = tree.xpath(
            f'//r:Relationship[@Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout"]',
            namespaces={'r': RELS_NS}
        )

        if not layout_rels:
            return None

        layout_target = layout_rels[0].get('Target')
        # layout_target is like "../slideLayouts/slideLayout1.xml"
        layout_name = layout_target.split('/')[-1]  # Extract "slideLayout1.xml"

        # Find the master for this layout
        master_name = layout_to_master.get(layout_name)
        if not master_name:
            return None

        # Find the theme for this master
        theme_name = master_to_theme.get(master_name)
        return theme_name

    except Exception:
        return None


def should_process_file(file_path: Path, extract_path: Path, theme_filter: list[str] | None,
                        layout_to_master: dict, master_to_theme: dict) -> bool:
    """
    Determine if a file should be processed based on theme filter.

    Args:
        file_path: Path to the file to check
        extract_path: Path to extracted .pptx contents
        theme_filter: List of theme names to include (e.g., ["theme1.xml"]), or None for all
        layout_to_master: Mapping of layouts to masters
        master_to_theme: Mapping of masters to themes

    Returns:
        True if file should be processed
    """
    if theme_filter is None:
        return True

    relative_path = file_path.relative_to(extract_path)
    path_str = str(relative_path)

    # For slides, check which theme they use
    if path_str.startswith('ppt/slides/slide'):
        theme = get_slide_theme(file_path, extract_path, layout_to_master, master_to_theme)
        if theme:
            # Convert theme_filter entries to file names if needed
            theme_files = [f if f.endswith('.xml') else f'{f}.xml' for f in theme_filter]
            return theme in theme_files

    # For slide layouts, check via master
    if path_str.startswith('ppt/slideLayouts/slideLayout'):
        layout_name = file_path.name
        master_name = layout_to_master.get(layout_name)
        if master_name:
            theme_name = master_to_theme.get(master_name)
            if theme_name:
                theme_files = [f if f.endswith('.xml') else f'{f}.xml' for f in theme_filter]
                return theme_name in theme_files

    # For slide masters, check directly
    if path_str.startswith('ppt/slideMasters/slideMaster'):
        master_name = file_path.name
        theme_name = master_to_theme.get(master_name)
        if theme_name:
            theme_files = [f if f.endswith('.xml') else f'{f}.xml' for f in theme_filter]
            return theme_name in theme_files

    # For other files (charts, diagrams, etc.), process if no theme filter
    # or default to processing them (they might be embedded in filtered slides)
    return True


def process_pptx(input_path: str, output_path: str, color_mapping: dict[str, str],
                 theme_filter: list[str] | None = None) -> int:
    """
    Process a PowerPoint file, replacing scheme color references.

    Args:
        input_path: Path to the input .pptx file
        output_path: Path to the output .pptx file
        color_mapping: Dictionary mapping source colors to target colors
        theme_filter: Optional list of theme names to filter (e.g., ["theme1", "theme2"])
                     If None, all themes are processed.

    Returns:
        Number of files processed

    Raises:
        FileNotFoundError: If input file doesn't exist
        ValueError: If input is not a valid .pptx file
    """
    if not os.path.exists(input_path):
        raise FileNotFoundError(f"Input file not found: {input_path}")

    if not zipfile.is_zipfile(input_path):
        raise ValueError(f"Input file is not a valid PowerPoint file: {input_path}")

    # XML files within the .pptx that may contain scheme color references
    xml_patterns = [
        'ppt/slides/',           # Slide content
        'ppt/slideLayouts/',     # Slide layouts
        'ppt/slideMasters/',     # Slide masters
        'ppt/charts/',           # Embedded charts
        'ppt/diagrams/',         # Diagrams
        'ppt/notesMasters/',     # Notes masters
        'ppt/notesSlides/',      # Notes slides
        'ppt/handoutMasters/',   # Handout masters
    ]

    files_processed = 0

    # Create a temporary directory to work in
    with tempfile.TemporaryDirectory() as temp_dir:
        temp_extract = Path(temp_dir) / 'extracted'
        temp_extract.mkdir()

        # Extract the entire .pptx
        with zipfile.ZipFile(input_path, 'r') as zip_ref:
            zip_ref.extractall(temp_extract)

        # Build theme relationship mappings
        master_to_theme = build_theme_relationships(temp_extract)
        layout_to_master = build_layout_to_master_mapping(temp_extract)

        # Process XML files
        for root, dirs, files in os.walk(temp_extract):
            for file in files:
                file_path = Path(root) / file

                # Only process XML files
                if not file.endswith('.xml'):
                    continue

                # Check if this file is in one of our target paths
                relative_path = file_path.relative_to(temp_extract)
                should_process = any(
                    str(relative_path).startswith(pattern.replace('/', os.sep))
                    for pattern in xml_patterns
                )

                if not should_process:
                    continue

                # Check theme filter
                if not should_process_file(file_path, temp_extract, theme_filter,
                                          layout_to_master, master_to_theme):
                    continue

                # Read, replace, and write back
                with open(file_path, 'rb') as f:
                    original_content = f.read()

                modified_content = replace_scheme_colors(original_content, color_mapping)

                with open(file_path, 'wb') as f:
                    f.write(modified_content)

                files_processed += 1

        # Rezip everything into the output file
        with zipfile.ZipFile(output_path, 'w', zipfile.ZIP_DEFLATED) as zip_out:
            for root, dirs, files in os.walk(temp_extract):
                for file in files:
                    file_path = Path(root) / file
                    arcname = file_path.relative_to(temp_extract)
                    zip_out.write(file_path, arcname)

    return files_processed
