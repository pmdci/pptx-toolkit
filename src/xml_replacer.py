"""Replace scheme color references in PowerPoint XML."""

from lxml import etree


def replace_scheme_colors(xml_content: bytes, color_mapping: dict[str, str]) -> bytes:
    """
    Replace scheme color references in XML content.

    Args:
        xml_content: Raw XML bytes from a PowerPoint part
        color_mapping: Dictionary mapping source colors to target colors

    Returns:
        Modified XML bytes with color references replaced
    """
    if not color_mapping:
        return xml_content

    try:
        tree = etree.fromstring(xml_content)
    except etree.XMLSyntaxError:
        # Not valid XML or not an XML file, return unchanged
        return xml_content

    # PowerPoint uses the DrawingML namespace for scheme colors
    # <a:schemeClr val="accent1"/> where 'a' is typically the DrawingML namespace
    namespaces = tree.nsmap

    # Find the DrawingML namespace (usually mapped to 'a')
    drawingml_ns = None
    for prefix, uri in namespaces.items():
        if uri and 'drawingml' in uri:
            drawingml_ns = uri
            break

    if not drawingml_ns:
        # Try common DrawingML namespace URIs
        common_ns = [
            'http://schemas.openxmlformats.org/drawingml/2006/main',
            'http://purl.oclc.org/ooxml/drawingml/main'
        ]
        for ns in common_ns:
            test_elements = tree.xpath(f'.//*[local-name()="schemeClr"]')
            if test_elements:
                drawingml_ns = test_elements[0].nsmap.get(test_elements[0].prefix)
                break

    # Find all schemeClr elements regardless of namespace prefix
    scheme_color_elements = tree.xpath('.//*[local-name()="schemeClr"]')

    replacements_made = 0
    for element in scheme_color_elements:
        current_val = element.get('val')
        if current_val and current_val in color_mapping:
            element.set('val', color_mapping[current_val])
            replacements_made += 1

    return etree.tostring(tree, xml_declaration=True, encoding='UTF-8')
