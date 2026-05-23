# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

pptx-toolkit is a Go CLI tool for manipulating PowerPoint (.pptx) files. It swaps scheme color references (e.g., `accent1`, `accent5`) throughout presentations without modifying the actual theme definitions.

**Core feature:** Atomic many-to-one color mapping. Example: `accent1→accent3, accent5→accent3` swaps both colors to `accent3` in one pass, with no cascading (if `accent3→accent4` is also defined, the original `accent1` stays as `accent3`, not `accent4`).

## Architecture

The codebase follows standard Go CLI patterns:

```
cmd/pptx-toolkit/
├── main.go           # Cobra CLI entry point and root command
├── colors_cmd.go     # Color command wiring (*_cmd.go convention)
├── layout_cmd.go     # Layout command wiring (*_cmd.go convention)
├── output_cmd.go     # Command-layer output helpers (*_cmd.go convention)
├── layout.go         # Layout domain logic and types
├── parser.go         # Color mapping and layout set parsing
├── processor.go      # XML scheme color replacement (regex-based)
├── pptx.go           # Theme filtering and document processing orchestration
├── rename.go         # Color scheme rename logic
├── slides.go         # Slide range parsing and mapping
├── theme.go          # Theme extraction (reads color schemes from themes)
└── *_test.go         # Unit tests alongside implementation

internal/
└── pptx/
    ├── pptx.go       # PPTX archive extraction and repacking helpers
    └── *_test.go     # Base-layer tests for archive I/O
```

**Key design decisions:**

1. **Direct XML manipulation** (regex + xmlquery) instead of full Office document libraries. PowerPoint files are ZIP archives containing XML. We find `<a:schemeClr val="accent1"/>` and replace the `val` attribute atomically.

2. **Theme filtering**: PowerPoint files can have multiple themes. We trace relationships: slide → layout → master → theme, then filter which files to process based on `--theme` flag.

3. **Atomic replacement**: Build a map of all replacements first, then apply in one pass to prevent cascading.

## Development Commands

```bash
# Build and test
make build              # Optimized binary → bin/pptx-toolkit
make test               # Run all tests
make clean              # Remove build artifacts

# Run specific tests
go test ./cmd/pptx-toolkit -v -run TestName

# Development iteration
make dev ARGS="theme color list testdata/test.pptx"

# Release builds
make build-release      # With UPX compression (if available)
make cross-compile      # All 6 platforms (macOS/Linux/Windows × ARM64/AMD64)
```

**Version is injected via git tags:** The Makefile uses `git describe --tags` and injects it via ldflags: `-X main.Version=$(VERSION)`. Run `./bin/pptx-toolkit -v` to verify.

## CLI Structure

Uses Cobra with nested commands:

- `pptx-toolkit theme color list <file.pptx>` - List all themes and color schemes
- `pptx-toolkit theme color rename <new-name> <in.pptx> <out.pptx> [--theme theme1]` - Rename theme colour scheme names
- `pptx-toolkit theme font list <file.pptx>` - List major/minor typefaces for each theme
- `pptx-toolkit theme font set <in.pptx> <out.pptx> --major "Arial" --minor "Times New Roman"` - Set theme fonts
- `pptx-toolkit color swap <in.pptx> <out.pptx> "accent1:accent3" --theme theme1` - Swap colors

**UK English alias:** `colour` works as an alias for `color` (both are valid).

## Testing

Test files live alongside implementation (`*_test.go`). Test fixtures are in `cmd/pptx-toolkit/testdata/test.pptx` (Go convention for test data).

Integration tests use the real `test.pptx` fixture with 5 themes (theme1–theme5).

When extracting pptx files to analyse contents, extract them in `cmd/pptx-toolkit/testdata/extracted/`.

An additional directory named `tests/` contain other tests files users for manual testing, which are ignored by git.

## Release Process

1. Commit changes
2. Tag: `git tag -a v0.x.y -m "Release v0.x.y"`
3. Push: `git push origin main && git push origin v0.x.y`
4. GitHub Actions (`.github/workflows/release.yml`) triggers GoReleaser
5. Binaries are built, compressed with UPX (where supported), and attached to GitHub release

**UPX compression caveats:**

- macOS: Skipped (Apple code signing issues)
- Windows ARM64: Skipped (UPX doesn't support PE ARM64 yet)
- Linux + Windows AMD64: Compressed (~57% size reduction)

## Code Conventions

- **Go naming:** US spelling in code (`color`, `ParseColorMapping`, etc.) - matches Go stdlib conventions
- **CLI/docs:** UK spelling accepted (`colour` alias, British English in user-facing text)
- **Error handling:** Return errors, don't panic
- **Tests:** Table-driven tests preferred

### Command Wiring Convention

Keep core domain logic in singular domain files (e.g. `layout.go`, `theme.go`). Put Cobra command wiring — command definitions, flag registration, `RunE` handlers — in `*_cmd.go` files (e.g. `layout_cmd.go`, `colors_cmd.go`).

**Boundary rule:** Non-command files must not depend on Cobra types or Cobra command lifecycle assumptions. As a first-order enforcement check: non-`*_cmd.go` files must not import `github.com/spf13/cobra`. The import check is a heuristic; the deeper intent is that domain and infrastructure code must remain callable without constructing a Cobra command.

**Isolation rule:** Structural refactors (file renames, reorganisation) must be in their own commits, separate from feature additions. This makes regressions attributable and keeps structural decisions safely revertable.
