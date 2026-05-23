# Command Wiring Convention

## Purpose

This note records the current command-wiring convention for `pptx-toolkit`.

It defines a clearer separation between:

- core domain logic
- Cobra CLI command wiring

## Current Situation

The following files already follow the convention:

- `layout.go` — core layout logic
- `layout_cmd.go` — Cobra command wiring for the layout domain
- `colors_cmd.go` — Cobra command wiring for the color domain
- `output_cmd.go` — command-facing output helpers; depends on Cobra types

`main.go` is the program entry point and is exempt from the `*_cmd.go` naming
requirement. It is unambiguously command-layer code and contains no domain logic.

The shared infrastructure files (`parser.go`, `processor.go`, `pptx.go`, `theme.go`,
`slides.go`, `rename.go`) do not import Cobra and are callable without a command
context, which is the intended state for non-command files.

## Shared Logic Clarification

The lower-level files used by the color commands should not be thought of as "color-owned" internals.

Files such as:

- `parser.go`
- `processor.go`
- `pptx.go`
- `theme.go`

are better understood as shared toolkit infrastructure.

That reuse is a good thing.
Multiple domains should be able to rely on those helpers where appropriate.

So the architectural concern here is **not** that shared logic lives outside a single command file.
That is normal and desirable.

The actual concern is narrower:

- command wiring is not named and organized consistently across domains

In other words:

- shared lower-level logic is good
- inconsistent command-layer organization is what is under review

## Convention

The project uses the following convention:

- singular domain files for core logic
  - examples: `layout.go`, `theme.go`
- `*_cmd.go` files for Cobra command wiring
  - examples: `layout_cmd.go`, future `theme_cmd.go`

This is intentionally simple:

- business logic and data structures live in domain files
- CLI definitions, flags, and `RunE` handlers live in command files

## Boundary Rule

The intended architectural rule is:

- non-command files should not depend on Cobra types or Cobra command lifecycle assumptions

As a simple first-order enforcement check:

- non-`*_cmd.go` files should not import `github.com/spf13/cobra`

That import rule is only a heuristic, not the full principle.
The actual goal is to keep domain and infrastructure code callable without constructing Cobra commands.

## Rationale

### Benefits

- makes file purpose obvious when navigating the codebase
- reduces mixing of CLI concerns with domain logic
- creates a repeatable pattern for future domains such as `theme`
- helps contributors understand where new code should go

### Costs

- some churn in filenames
- some refactoring effort if existing files mix orchestration and CLI concerns
- temporary inconsistency while the migration is incomplete

## Scope

This convention is about file organization, not package boundaries.

Everything would still remain in the same Go package unless there is a separate decision to break code into subpackages.

The goal here is modest:

- improve clarity
- reduce architectural drift
- make the command surface easier to extend

## Migration Strategy

The remaining migration approach is gradual, not a large one-shot refactor.

What still matters going forward:

1. Apply the convention to new command domains.
2. Opportunistically rename any remaining command files when touching those areas anyway.
3. Avoid behavior changes during file-organization-only refactors.

## Isolation Rule

Structural refactors and feature work should be kept separate.

In practice:

- file-organization refactors should be in their own commits or PRs
- feature additions should not be bundled into the same change as architectural cleanup

This makes regressions easier to attribute and makes it safe to revert a structural decision without also reverting product work.

## Decision Log

The following points are considered settled for now:

1. `*_cmd.go` is the preferred naming pattern for Cobra wiring files.
2. The migration should be gradual rather than a one-shot reorganization.
3. `layout.go` for logic and `layout_cmd.go` for CLI wiring is the model to follow.
4. `colors.go` was command-layer code and has been renamed to `colors_cmd.go`; this was a rename, not a domain split — the domain logic was already in `processor.go`, `parser.go`, and related files.
5. The isolation rule should be formalized in `AGENTS.md` and followed in practice.
