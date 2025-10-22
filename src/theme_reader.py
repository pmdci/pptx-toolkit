"""Extract and read theme information from PowerPoint files."""

import zipfile
from dataclasses import dataclass
from lxml import etree
from typing import Optional


@dataclass
class ColorScheme:
    """Represents a color scheme with all scheme colors."""
    dk1: str
    lt1: str
    dk2: str
    lt2: str
    accent1: str
    accent2: str
    accent3: str
    accent4: str
    accent5: str
    accent6: str
    hlink: str
    fol_hlink: str


@dataclass
class Theme:
    """Represents a PowerPoint theme."""
    file_name: str  # e.g., "theme1.xml"
    theme_name: str  # e.g., "Office Theme Deck"
    color_scheme_name: str  # e.g., "Office"
    colors: ColorScheme


DRAWINGML_NS = "http://schemas.openxmlformats.org/drawingml/2006/main"


def extract_rgb_color(color_element: etree._Element) -> Optional[str]:
    """
    Extract RGB color value from a color definition element.

    Args:
        color_element: The element containing color definition (e.g., <a:accent1>)

    Returns:
        Hex color string (e.g., "156082") or None if not found
    """
    # Try <a:srgbClr val="156082"/>
    srgb = color_element.find(f"{{{DRAWINGML_NS}}}srgbClr")
    if srgb is not None:
        return srgb.get('val')

    # Try <a:sysClr val="windowText" lastClr="000000"/>
    sys_clr = color_element.find(f"{{{DRAWINGML_NS}}}sysClr")
    if sys_clr is not None:
        last_clr = sys_clr.get('lastClr')
        if last_clr:
            return last_clr

    return None


def parse_theme_xml(xml_content: bytes, file_name: str) -> Optional[Theme]:
    """
    Parse a theme XML file and extract theme information.

    Args:
        xml_content: Raw XML bytes from a theme file
        file_name: Name of the theme file (e.g., "theme1.xml")

    Returns:
        Theme object or None if parsing fails
    """
    try:
        tree = etree.fromstring(xml_content)
    except etree.XMLSyntaxError:
        return None

    # Extract theme name
    theme_name = tree.get('name', file_name)

    # Find color scheme
    clr_scheme = tree.find(f".//{{{DRAWINGML_NS}}}clrScheme")
    if clr_scheme is None:
        return None

    color_scheme_name = clr_scheme.get('name', 'Unknown')

    # Extract all scheme colors
    def get_color(name: str) -> str:
        elem = clr_scheme.find(f"{{{DRAWINGML_NS}}}{name}")
        if elem is not None:
            color = extract_rgb_color(elem)
            if color:
                return color
        return "000000"  # Default to black if not found

    colors = ColorScheme(
        dk1=get_color('dk1'),
        lt1=get_color('lt1'),
        dk2=get_color('dk2'),
        lt2=get_color('lt2'),
        accent1=get_color('accent1'),
        accent2=get_color('accent2'),
        accent3=get_color('accent3'),
        accent4=get_color('accent4'),
        accent5=get_color('accent5'),
        accent6=get_color('accent6'),
        hlink=get_color('hlink'),
        fol_hlink=get_color('folHlink')
    )

    return Theme(
        file_name=file_name,
        theme_name=theme_name,
        color_scheme_name=color_scheme_name,
        colors=colors
    )


def read_themes(pptx_path: str) -> list[Theme]:
    """
    Read all themes from a PowerPoint file.

    Args:
        pptx_path: Path to the .pptx file

    Returns:
        List of Theme objects
    """
    themes = []

    with zipfile.ZipFile(pptx_path, 'r') as zip_ref:
        # Get list of theme files
        theme_files = [
            name for name in zip_ref.namelist()
            if name.startswith('ppt/theme/theme') and name.endswith('.xml')
        ]

        # Sort to ensure consistent ordering (theme1, theme2, etc.)
        theme_files.sort()

        for theme_file in theme_files:
            xml_content = zip_ref.read(theme_file)
            file_name = theme_file.split('/')[-1]  # Extract just "theme1.xml"

            theme = parse_theme_xml(xml_content, file_name)
            if theme:
                themes.append(theme)

    return themes
