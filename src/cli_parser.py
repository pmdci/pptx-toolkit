"""Parse and validate CLI color mapping arguments."""

VALID_SCHEME_COLORS = {
    "dk1", "lt1", "dk2", "lt2",
    "accent1", "accent2", "accent3", "accent4", "accent5", "accent6",
    "hlink", "folHlink"
}


class ColorMappingError(Exception):
    """Raised when color mapping validation fails."""
    pass


def parse_color_mapping(mapping_string: str) -> dict[str, str]:
    """
    Parse a color mapping string into a validated dictionary.

    Args:
        mapping_string: Comma-separated mappings, e.g., "accent1:accent3,accent5:accent3"

    Returns:
        Dictionary mapping source colors to target colors

    Raises:
        ColorMappingError: If mapping is invalid, has conflicts, or uses invalid color names
    """
    if not mapping_string or not mapping_string.strip():
        raise ColorMappingError("Mapping string cannot be empty")

    mappings: dict[str, str] = {}
    pairs = mapping_string.split(",")

    for pair in pairs:
        pair = pair.strip()
        if not pair:
            continue

        if ":" not in pair:
            raise ColorMappingError(f"Invalid mapping format: '{pair}'. Expected 'source:target'")

        parts = pair.split(":")
        if len(parts) != 2:
            raise ColorMappingError(f"Invalid mapping format: '{pair}'. Expected exactly one ':'")

        source = parts[0].strip()
        target = parts[1].strip()

        if not source or not target:
            raise ColorMappingError(f"Invalid mapping: '{pair}'. Source and target cannot be empty")

        # Validate color names
        if source not in VALID_SCHEME_COLORS:
            raise ColorMappingError(
                f"Invalid source color: '{source}'. "
                f"Valid colors are: {', '.join(sorted(VALID_SCHEME_COLORS))}"
            )

        if target not in VALID_SCHEME_COLORS:
            raise ColorMappingError(
                f"Invalid target color: '{target}'. "
                f"Valid colors are: {', '.join(sorted(VALID_SCHEME_COLORS))}"
            )

        # Check for conflicts
        if source in mappings:
            if mappings[source] != target:
                raise ColorMappingError(
                    f"Conflicting mappings for '{source}':\n"
                    f"  - {source} → {mappings[source]}\n"
                    f"  - {source} → {target}"
                )
            # Duplicate identical mapping, skip
            continue

        mappings[source] = target

    if not mappings:
        raise ColorMappingError("No valid mappings found")

    return mappings
