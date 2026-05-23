# Spec: Theme Font Commands

## Feature summary

Add `theme font list` and `theme font set` subcommands to the pptx-toolkit CLI.
These commands expose the font scheme embedded in each PowerPoint theme — the two
typefaces (major/heading and minor/body) that drive text rendering across the file.

---

## OOXML ground truth

Each theme XML file (`ppt/theme/themeN.xml`) contains a `<a:fontScheme>` element
inside `<a:themeElements>`:

```xml
<a:fontScheme name="Office">
  <a:majorFont>
    <a:latin typeface="Aptos Display" panose="..."/>
    <a:ea typeface=""/>
    <a:cs typeface=""/>
    <a:font script="Jpan" typeface="游ゴシック Light"/>
    <!-- ... up to 47 per-script overrides ... -->
  </a:majorFont>
  <a:minorFont>
    <a:latin typeface="Aptos" panose="..."/>
    <a:ea typeface=""/>
    <a:cs typeface=""/>
    <a:font script="Jpan" typeface="游明朝"/>
    <!-- ... up to 47 per-script overrides ... -->
  </a:minorFont>
</a:fontScheme>
```

Key facts:
- `majorFont` drives headings, titles, and slide headers.
- `minorFont` drives body text, captions, and content placeholders.
- The latin typeface is the value of `<a:latin typeface="..."/>`.
- The `<a:fontScheme name="..."/>` attribute is a human-readable scheme name
  (e.g. `"Office"`, `"Custom Fonts"`) — it is decorative and not used for rendering.
- Per-script overrides (`<a:font script="..." typeface="..."/>`) exist independently
  of the latin typeface. Changing the latin font does not change them.
- Observed in test.pptx: stock Office themes carry 47 per-script overrides;
  custom themes may carry zero.

---

## Commands

### `pptx-toolkit theme font list <file.pptx>`

List the font scheme for every theme in the file.

**Arguments:**
- `<file.pptx>` — input file (read-only, no output path)

**Flags:**
- `--theme <name>` — filter to one or more themes by file basename (e.g. `theme1`, `theme2`); same semantics as `theme color list`

**Output (stdout):**

```
Found 2 theme(s) in presentation.pptx:

━━━ theme1.xml ━━━
Theme:        Office Theme Deck
Font Scheme:  Office

Fonts:
  major  (headings):  Aptos Display
  minor  (body):      Aptos

━━━ theme2.xml ━━━
Theme:        Blue II Deck
Font Scheme:  Office

Fonts:
  major  (headings):  Aptos Display
  minor  (body):      Aptos

```

- The output style mirrors `theme color list`: header line, then per-theme blocks with theme
  file, theme name, and scheme name at column 14. Values in the Fonts block are aligned at
  column 22 within the indented section.
- Font role codes are lowercase (`major`, `minor`) with a parenthetical description.
- Script-specific overrides are not shown; this is a deliberate scope decision
  (see Caveats section).

---

### `pptx-toolkit theme font set <input.pptx> <output.pptx>`

Set the latin typeface for major, minor, or both font roles in matching themes.

**Arguments:**
- `<input.pptx>` — source file
- `<output.pptx>` — destination file (same path as input triggers overwrite prompt)

**Flags:**
- `--major <typeface>` — set the heading font (`<a:majorFont><a:latin typeface="..."/>`)
- `--minor <typeface>` — set the body font (`<a:minorFont><a:latin typeface="..."/>`)
- `--scheme-name <name>` — rename the font scheme (`<a:fontScheme name="..."/>`)
- `--theme <name>` — filter to specific themes by file basename (e.g. `theme1`); same semantics as other theme commands

**Constraints:**
- At least one of `--major`, `--minor`, or `--scheme-name` must be provided; omitting all three is an error.
- Font names are treated as opaque strings — no validation against installed fonts.
  PowerPoint resolves unknown fonts at render time using its own fallback chain.
- The `panose` attribute on `<a:latin>` is preserved unchanged when only `typeface` is updated.
- Script-specific overrides (`<a:font script="..."/>`) are not modified.
- The `<a:fontScheme name="..."/>` scheme name is not modified.

**Output (stdout) — mirrors `theme color rename`:**

```
Setting fonts in presentation.pptx → output.pptx

  Major (headings):  Calibri
  Minor (body):      Calibri Body
  Scheme name:       Corporate
  Theme filter:      theme1

Modified 1 theme(s).
Saved to output.pptx
```

Only lines for flags that were actually provided are shown.

---

## Actors and responsibilities

| Actor | Responsibility |
|---|---|
| `font.go` | Domain logic: `ReadFontSchemes`, `SetFontScheme`. Pure functions, no Cobra dependency. |
| `theme_font_cmd.go` | Cobra wiring: `fontCmd`, `fontListCmd`, `fontSetCmd`, flag registration, `RunE` handlers. |
| `theme_cmd.go` | Registers `fontCmd` as a subcommand of `themeCmd` (alongside `colorCmd`). |

---

## Behaviours

### List

1. Open and extract the PPTX.
2. For each theme file matching the `--theme` filter (or all, if no filter):
   a. Parse `<a:fontScheme>`, `<a:majorFont>/<a:latin>`, `<a:minorFont>/<a:latin>`.
   b. Collect theme file name, theme name, font scheme name, major typeface, minor typeface.
3. Print results in the format above.
4. If no themes match the filter, print an error and exit non-zero.

### Set

1. Validate: at least one of `--major`, `--minor`, `--scheme-name` must be non-empty.
2. Open and extract the PPTX to a temp directory.
3. For each theme file matching the `--theme` filter (or all, if no filter):
   a. Parse the theme XML.
   b. If `--major` is set: update `typeface` attribute on `<a:majorFont>/<a:latin>`.
   c. If `--minor` is set: update `typeface` attribute on `<a:minorFont>/<a:latin>`.
   d. If `--scheme-name` is set: update `name` attribute on `<a:fontScheme>`.
   e. Leave all other attributes and child elements (including script overrides) untouched.
   f. Write the modified XML back.
4. Repack the PPTX to the output path.
5. Print a summary of what was changed and how many themes were affected.
6. If no themes match the filter, print an error and exit non-zero.

---

## Error cases

| Condition | Behaviour |
|---|---|
| No `--major`, `--minor`, or `--scheme-name` provided | Error: "at least one of --major, --minor, or --scheme-name is required" |
| `--theme` filter matches no themes | Error: "no themes matched filter: <names>" |
| Input file does not exist | Error from `ValidateInputFile` |
| Input and output path are identical | Overwrite prompt (existing `PrepareMutation` behaviour) |
| Malformed theme XML | Propagate parse error with file name |

---

## Caveats and out-of-scope items

### Script-specific font overrides are not modified

Each theme can carry up to 47 per-script overrides (Japanese, Arabic, Hebrew, Thai,
etc.) in both `<a:majorFont>` and `<a:minorFont>`. This spec covers only the latin
typeface (`<a:latin typeface="..."/>`).

**Consequence:** if a file is used with non-latin content (CJK slides, Arabic text,
etc.), changing the latin font will not update the per-script fallbacks. Those slides
will continue rendering with the original script fonts. This is the correct behaviour
for the common case (latin-only presentations), but it is a gap for multilingual files.

**Future consideration:** a separate `--scripts` flag or a `theme font set-script`
command could expose per-script font assignment. Not in scope for this version.

### `panose` attribute is preserved

The `<a:latin>` element carries an optional `panose` attribute (a font classification
code). This spec preserves the existing `panose` value when updating the `typeface`.
A strictly correct implementation would clear or recompute `panose` for the new font,
but PowerPoint ignores it at render time; preserving it is safe and simpler.

Future: could clear `panose` when the typeface changes, to avoid misleading metadata.

### Font scheme name

The `<a:fontScheme name="..."/>` attribute is decorative (PowerPoint does not use it
for rendering). It can be updated via `--scheme-name` on `font set`. When omitted,
the existing name is preserved.

### No font name validation

Font names are passed through as opaque strings. There is no check against installed
fonts, no lookup against a known font list, and no fallback behaviour. This matches
the behaviour of `theme color rename`, which also accepts arbitrary strings.

---

## File naming (command wiring convention)

Following `docs/command-wiring-convention.md`:

- `cmd/pptx-toolkit/font.go` — domain logic (`ReadFontSchemes`, `SetFontScheme`, types)
- `cmd/pptx-toolkit/theme_font_cmd.go` — Cobra wiring only; no domain logic
- `theme_font_cmd.go` must not import any package other than `cobra`, the domain
  file's types, and output helpers

---

## Test plan

### Font fixtures

Tests use **Arial** (`--major`) and **Times New Roman** (`--minor`) as font values.
Both are bundled at the OS level on Windows and macOS and are available in any CI
environment without Office installed. They are visually distinct from the default
test fixture fonts (`Aptos Display` / `Aptos`), so before/after assertions are
unambiguous.

Font names are opaque strings in this implementation (no system lookup), so any
string would exercise the code path equally — the real fonts are chosen for
readability when inspecting fixtures or diffs.

### Unit tests (`font_test.go`)

| Test | What it checks |
|---|---|
| `TestReadFontSchemes` | Parses `testdata/test.pptx`; asserts known major/minor typefaces and scheme name for each theme. |
| `TestSetFontScheme_MajorOnly` | Applies `--major "Arial"` to a theme XML fragment; asserts `majorFont/latin@typeface` changed, `minorFont/latin@typeface` unchanged, `panose` unchanged, script overrides unchanged. |
| `TestSetFontScheme_MinorOnly` | Applies `--minor "Times New Roman"`; asserts `minorFont` changed, `majorFont` unchanged. |
| `TestSetFontScheme_Both` | Applies both `--major "Arial"` and `--minor "Times New Roman"`; asserts both updated. |
| `TestSetFontScheme_SchemeName` | Applies `--scheme-name "Corporate"`; asserts `fontScheme@name` changed, typefaces unchanged. |
| `TestSetFontScheme_Preserves47ScriptOverrides` | Verifies that a theme XML with 47 `<a:font script="..."/>` entries has all 47 intact after a set operation. |
| `TestSetFontScheme_PreservesPanose` | Verifies the `panose` attribute value on `<a:latin>` is identical before and after a typeface change. |

### Integration tests (`theme_font_cmd_test.go`)

Table-driven; each case runs the real CLI against `testdata/test.pptx` and inspects
the output file with `ReadFontSchemes`.

| Case | Flags | Assertion |
|---|---|---|
| Major only | `--major "Arial"` | All themes: major = `Arial`; minor unchanged. |
| Minor only | `--minor "Times New Roman"` | All themes: minor = `Times New Roman`; major unchanged. |
| Both | `--major "Arial" --minor "Times New Roman"` | All themes: both updated. |
| Scheme name only | `--scheme-name "Corporate"` | All themes: scheme name = `Corporate`; typefaces unchanged. |
| Theme filter | `--major "Arial" --theme theme1` | Only `theme1.xml` major changed; other themes unchanged. |
| No flags | _(none)_ | Exit non-zero; stderr contains "at least one of". |
| Unknown theme filter | `--theme "NoSuchTheme"` | Exit non-zero; stderr contains "no themes matched". |
| Round-trip | list → set → list | Second list output shows changed values. |

---

## Acceptance criteria

- [ ] `theme font list testdata/test.pptx` prints major and minor fonts for all 5 themes
- [ ] `theme font list testdata/test.pptx --theme theme1` prints only theme1
- [ ] `theme font set in.pptx out.pptx --major "Arial"` updates only `majorFont`; `minorFont` unchanged
- [ ] `theme font set in.pptx out.pptx --minor "Times New Roman"` updates only `minorFont`
- [ ] `theme font set in.pptx out.pptx --major "Arial" --minor "Times New Roman"` updates both
- [ ] `theme font set in.pptx out.pptx --scheme-name "Corporate"` renames scheme, leaves typefaces unchanged
- [ ] `theme font set in.pptx out.pptx` (no flags) exits non-zero with clear error
- [ ] `theme font set` with `--theme` only modifies matching themes; others are unchanged
- [ ] `theme font list` then `theme font set` then `theme font list` shows the change
- [ ] Script overrides are untouched after a `set` operation
- [ ] `panose` attribute is untouched after a `set` operation
- [ ] All existing tests continue to pass
