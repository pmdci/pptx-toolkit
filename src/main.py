#!/usr/bin/env python3
"""CLI tool for PowerPoint utilities."""

import sys
from pathlib import Path
from typing import Optional
import typer
from .cli_parser import parse_color_mapping, ColorMappingError
from .pptx_processor import process_pptx
from .theme_reader import read_themes


VERSION = "0.1"

VERSION_BANNER = f"""pptx-toolkit v{VERSION}

╭━━━┳━━━┳━━━━┳━╮╭━╮╭━━━━╮╱╱╱╱╭╮╭╮╱╱╭╮
┃╭━╮┃╭━╮┃╭╮╭╮┣╮╰╯╭╯┃╭╮╭╮┃╱╱╱╱┃┃┃┃╱╭╯╰╮ Copyright (C) 2025 Pedro Innecco
┃╰━╯┃╰━╯┣╯┃┃╰╯╰╮╭╯╱╰╯┃┃┣┻━┳━━┫┃┃┃╭╋╮╭╯ <https://pedroinnecco.com>
┃╭━━┫╭━━╯╱┃┃╱╱╭╯╰┳━━╮┃┃┃╭╮┃╭╮┃┃┃╰╯╋┫┃
┃┃╱╱┃┃╱╱╱╱┃┃╱╭╯╭╮╰┳━╯┃┃┃╰╯┃╰╯┃╰┫╭╮┫┃╰╮
╰╯╱╱╰╯╱╱╱╱╰╯╱╰━╯╰━╯╱╱╰╯╰━━┻━━┻━┻╯╰┻┻━╯

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program comes with ABSOLUTELY NO WARRANTY.
See <https://www.gnu.org/licenses/gpl-3.0.html> for details.

Source: https://github.com/pmdci/pptx-toolkit

Brought to you by the letter P.
"""


# Create the main app
app = typer.Typer(
    name="pptx-toolkit",
    help="PowerPoint toolkit for colors, themes, and other utilities.",
)

# Create colors sub-app
colors_app = typer.Typer(
    name="colors",
    help="Color-related operations",
    no_args_is_help=True,
)


def version_callback(value: bool):
    """Callback for --version flag."""
    if value:
        typer.echo(VERSION_BANNER)
        raise typer.Exit()


@app.callback()
def main_callback(
    version: Optional[bool] = typer.Option(
        None,
        "--version",
        callback=version_callback,
        is_eager=True,
        help="Show version information"
    )
):
    """
    pptx-toolkit: PowerPoint manipulation toolkit.

    Use "pptx-toolkit <group> <command> --help" for command-specific help.
    """
    pass


@colors_app.command("list")
def colors_list(
    input_file: str = typer.Argument(..., help="Input PowerPoint file (.pptx)")
):
    """List all color schemes in a PowerPoint file."""

    # Convert to Path object
    input_path = Path(input_file)

    # Validate file exists
    if not input_path.exists():
        typer.echo(f"Error: Input file not found: {input_file}", err=True)
        raise typer.Exit(1)

    # Validate file extension
    if input_path.suffix.lower() != '.pptx':
        typer.echo(f"Error: Input file must be a .pptx file: {input_file}", err=True)
        raise typer.Exit(1)

    # Read themes
    try:
        themes = read_themes(str(input_file))

        if not themes:
            typer.echo("No themes found in PowerPoint file.", err=True)
            raise typer.Exit(1)

        # Display themes
        typer.echo(f"\nFound {len(themes)} theme(s) in {input_file}:\n")

        for theme in themes:
            typer.echo(f"━━━ {theme.file_name} ━━━")
            typer.echo(f"Theme:        {theme.theme_name}")
            typer.echo(f"Color Scheme: {theme.color_scheme_name}")
            typer.echo()
            typer.echo("Colors:")
            typer.echo(f"  dk1      (Dark 1):              #{theme.colors.dk1}")
            typer.echo(f"  lt1      (Light 1):             #{theme.colors.lt1}")
            typer.echo(f"  dk2      (Dark 2):              #{theme.colors.dk2}")
            typer.echo(f"  lt2      (Light 2):             #{theme.colors.lt2}")
            typer.echo(f"  accent1  (Accent 1):            #{theme.colors.accent1}")
            typer.echo(f"  accent2  (Accent 2):            #{theme.colors.accent2}")
            typer.echo(f"  accent3  (Accent 3):            #{theme.colors.accent3}")
            typer.echo(f"  accent4  (Accent 4):            #{theme.colors.accent4}")
            typer.echo(f"  accent5  (Accent 5):            #{theme.colors.accent5}")
            typer.echo(f"  accent6  (Accent 6):            #{theme.colors.accent6}")
            typer.echo(f"  hlink    (Hyperlink):           #{theme.colors.hlink}")
            typer.echo(f"  folHlink (Followed Hyperlink):  #{theme.colors.fol_hlink}")
            typer.echo()

    except Exception as e:
        typer.echo(f"Error reading themes: {e}", err=True)
        raise typer.Exit(1)


@colors_app.command("swap")
def colors_swap(
    input_file: str = typer.Argument(..., help="Input PowerPoint file (.pptx)"),
    output_file: str = typer.Argument(..., help="Output PowerPoint file (.pptx)"),
    mapping: str = typer.Argument(..., help='Color mappings (e.g., "accent1:accent3,accent5:accent3")'),
    theme: Optional[str] = typer.Option(None, "--theme", help="Comma-separated list of themes to target (e.g., theme1,theme2)"),
):
    """
    Swap scheme color references in slides.

    Example: pptx-toolkit colors swap input.pptx output.pptx "accent1:accent3"
    """

    # Convert strings to Path objects
    input_path = Path(input_file)
    output_path = Path(output_file)

    # Validate input file exists
    if not input_path.exists():
        typer.echo(f"Error: Input file not found: {input_file}", err=True)
        raise typer.Exit(1)

    # Validate input file extension
    if input_path.suffix.lower() != '.pptx':
        typer.echo(f"Error: Input file must be a .pptx file: {input_file}", err=True)
        raise typer.Exit(1)

    # Validate output path
    if output_path.exists():
        overwrite = typer.confirm(f"Output file '{output_file}' already exists. Overwrite?")
        if not overwrite:
            typer.echo("Aborted.")
            raise typer.Exit(0)

    # Parse color mapping
    try:
        color_mapping = parse_color_mapping(mapping)
    except ColorMappingError as e:
        typer.echo(f"Error: {e}", err=True)
        raise typer.Exit(1)

    # Parse theme filter (if provided)
    theme_filter = None
    if theme:
        theme_filter = [t.strip() for t in theme.split(',')]

    # Process the file
    try:
        typer.echo(f"Processing {input_file}...")
        typer.echo(f"Mappings: {', '.join(f'{k}→{v}' for k, v in color_mapping.items())}")

        if theme_filter:
            typer.echo(f"Themes: {', '.join(theme_filter)}")
        else:
            typer.echo("Themes: all")

        files_processed = process_pptx(
            str(input_file),
            str(output_file),
            color_mapping,
            theme_filter=theme_filter
        )

        typer.echo(f"✓ Successfully processed {files_processed} files")
        typer.echo(f"✓ Output saved to {output_file}")

    except Exception as e:
        typer.echo(f"Error processing file: {e}", err=True)
        raise typer.Exit(1)


# Add colors sub-app to main app
app.add_typer(colors_app, name="colors")


def main():
    """Entry point for the CLI."""
    app()


if __name__ == '__main__':
    main()
