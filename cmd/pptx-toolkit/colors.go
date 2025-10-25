package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var colorCmd = &cobra.Command{
	Use:     "color",
	Aliases: []string{"colour"},
	Short:   "Color-related operations",
	Long:    "Color-related operations for PowerPoint files.",
}

var colorListCmd = &cobra.Command{
	Use:   "list <input.pptx>",
	Short: "List all color schemes in a PowerPoint file",
	Args:  cobra.ExactArgs(1),
	RunE:  runColorList,
}

var colorSwapCmd = &cobra.Command{
	Use:   "swap <mapping> <input.pptx> <output.pptx>",
	Short: "Swap color references in slides",
	Long: `Swap color references in slides.

Supports swapping between scheme colors (e.g., accent1, dk1) and hex RGB values (e.g., AABBCC, FF0000).

Scope options:
  all      - Process all files (default)
  content  - Process user content only (slides, charts, diagrams, notes)
  master   - Process master infrastructure only (slideMasters, slideLayouts, notesMasters, handoutMasters)

Slide filtering:
  Use --slides to target specific slides. Automatically includes embedded content (charts, diagrams, notes).
  IMPORTANT: --slides can only be used with --scope content (explicit or implicit).
  If you don't specify --scope, it defaults to content when using --slides.

Examples:
  # Scheme to scheme
  pptx-toolkit color swap "accent1:accent3" input.pptx output.pptx

  # Scheme to hex
  pptx-toolkit color swap "accent1:BBFFCC" input.pptx output.pptx

  # Fix user overrides in content only
  pptx-toolkit color swap "AABBCC:accent2" input.pptx output.pptx --scope content

  # Update master template only
  pptx-toolkit color swap "accent1:accent5" input.pptx output.pptx --scope master

  # Process specific slides (auto-sets scope to content)
  pptx-toolkit color swap "accent1:accent3" input.pptx output.pptx --slides 1,3,5

  # Process slide range
  pptx-toolkit color swap "accent1:accent3" input.pptx output.pptx --slides 5-8

  # Combine slides with theme filtering
  pptx-toolkit color swap "accent1:accent3" input.pptx output.pptx --slides 1-5 --theme theme1

  # Multiple mappings
  pptx-toolkit color swap "accent1:BBFFCC,AABBCC:accent2,FF0000:00FF00" input.pptx output.pptx`,
	Args: cobra.ExactArgs(3),
	RunE: runColorSwap,
}

var colorRenameCmd = &cobra.Command{
	Use:   "rename <new-name> <input.pptx> <output.pptx>",
	Short: "Rename colour scheme(s)",
	Long: `Rename colour scheme(s) in themes.

By default, renames the colour scheme in all themes. Use --theme to target specific themes.

Examples:
  # Rename in all themes
  pptx-toolkit color rename "Azure Blue" input.pptx output.pptx

  # Rename in specific theme
  pptx-toolkit color rename "Corporate Brand" input.pptx output.pptx --theme theme1

  # Rename in multiple themes
  pptx-toolkit color rename "New Scheme" input.pptx output.pptx --theme theme1,theme2`,
	Args: cobra.ExactArgs(3),
	RunE: runColorRename,
}

var (
	themeFilter       []string
	renameThemeFilter []string
	scopeFilter       string
	slideFilter       string
)

func init() {
	colorCmd.AddCommand(colorListCmd)
	colorCmd.AddCommand(colorSwapCmd)
	colorCmd.AddCommand(colorRenameCmd)

	// Add --theme flag to swap command
	colorSwapCmd.Flags().StringSliceVar(&themeFilter, "theme", nil, "Comma-separated list of themes to target (e.g., theme1,theme2)")

	// Add --scope flag to swap command
	colorSwapCmd.Flags().StringVar(&scopeFilter, "scope", "all", "Processing scope (all, content, master)")

	// Add --slides flag to swap command
	colorSwapCmd.Flags().StringVar(&slideFilter, "slides", "", "Comma-separated slide numbers or ranges (e.g., 1,3,5-8)")

	// Add --theme flag to rename command
	colorRenameCmd.Flags().StringSliceVar(&renameThemeFilter, "theme", nil, "Comma-separated list of themes to target (e.g., theme1,theme2)")
}

func runColorList(cmd *cobra.Command, args []string) error {
	inputFile := args[0]

	// Read themes
	themes, err := ReadThemes(inputFile)
	if err != nil {
		return fmt.Errorf("error reading themes: %w", err)
	}

	if len(themes) == 0 {
		cmd.PrintErrln("No themes found in PowerPoint file.")
		return fmt.Errorf("no themes found")
	}

	// Display themes
	cmd.Printf("\nFound %d theme(s) in %s:\n\n", len(themes), inputFile)

	for _, theme := range themes {
		cmd.Printf("━━━ %s ━━━\n", theme.FileName)
		cmd.Printf("Theme:        %s\n", theme.ThemeName)
		cmd.Printf("Color Scheme: %s\n", theme.ColorSchemeName)
		cmd.Println()
		cmd.Println("Colors:")
		cmd.Printf("  dk1      (Dark 1):              #%s\n", theme.Colors.Dk1)
		cmd.Printf("  lt1      (Light 1):             #%s\n", theme.Colors.Lt1)
		cmd.Printf("  dk2      (Dark 2):              #%s\n", theme.Colors.Dk2)
		cmd.Printf("  lt2      (Light 2):             #%s\n", theme.Colors.Lt2)
		cmd.Printf("  accent1  (Accent 1):            #%s\n", theme.Colors.Accent1)
		cmd.Printf("  accent2  (Accent 2):            #%s\n", theme.Colors.Accent2)
		cmd.Printf("  accent3  (Accent 3):            #%s\n", theme.Colors.Accent3)
		cmd.Printf("  accent4  (Accent 4):            #%s\n", theme.Colors.Accent4)
		cmd.Printf("  accent5  (Accent 5):            #%s\n", theme.Colors.Accent5)
		cmd.Printf("  accent6  (Accent 6):            #%s\n", theme.Colors.Accent6)
		cmd.Printf("  hlink    (Hyperlink):           #%s\n", theme.Colors.Hlink)
		cmd.Printf("  folHlink (Followed Hyperlink):  #%s\n", theme.Colors.FolHlink)
		cmd.Println()
	}

	return nil
}

func runColorSwap(cmd *cobra.Command, args []string) error {
	// Suppress usage and errors for validation errors - syntax errors are
	// already handled by Cobra's Args validator. We'll print errors ourselves.
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	mappingStr := args[0]
	inputFile := args[1]
	outputFile := args[2]

	// Validate input file
	if err := ValidateInputFile(inputFile); err != nil {
		cmd.PrintErrln("Error:", err)
		return fmt.Errorf("") // Return empty error to set exit code
	}

	// Prompt for overwrite if needed
	if shouldContinue, err := PromptOverwrite(cmd, outputFile); err != nil || !shouldContinue {
		return err
	}

	// Parse color mapping
	colorMapping, err := ParseColorMapping(mappingStr)
	if err != nil {
		cmd.PrintErrln("Error:", err)
		return fmt.Errorf("") // Return empty error to set exit code
	}

	// Parse slide filter if provided
	var slides []int
	if slideFilter != "" {
		slides, err = ParseSlideRange(slideFilter)
		if err != nil {
			cmd.PrintErrln("Error:", err)
			return fmt.Errorf("") // Return empty error to set exit code
		}
	}

	// Validate scope compatibility with slides
	scopeSource := "default"
	if len(slides) > 0 {
		// --slides can only be used with --scope content
		if scopeFilter != "all" && scopeFilter != "content" {
			cmd.PrintErrln("Error: --slides can only be used with --scope content")
			return fmt.Errorf("") // Return empty error to set exit code
		}

		// Auto-set scope to content if not explicitly set
		if scopeFilter == "all" {
			scopeFilter = "content"
			scopeSource = "auto"
		} else {
			scopeSource = "explicit"
		}
	} else if scopeFilter != "all" {
		scopeSource = "explicit"
	}

	// Format mappings for display
	var mappingStrs []string
	for source, target := range colorMapping {
		mappingStrs = append(mappingStrs, fmt.Sprintf("%s→%s", source, target))
	}

	// Print processing header
	config := ProcessingConfig{
		Mappings:    mappingStrs,
		Themes:      themeFilter,
		Slides:      slides,
		Scope:       scopeFilter,
		ScopeSource: scopeSource,
	}
	PrintProcessingHeader(cmd, inputFile, config)

	filesProcessed, err := ProcessPPTX(inputFile, outputFile, colorMapping, themeFilter, scopeFilter, slides)
	if err != nil {
		cmd.PrintErrf("\nError: %v\n", err)
		return fmt.Errorf("") // Return empty error to set exit code
	}

	PrintSuccess(cmd, filesProcessed, "files", outputFile)

	return nil
}

func runColorRename(cmd *cobra.Command, args []string) error {
	// Suppress usage and errors for validation errors - syntax errors are
	// already handled by Cobra's Args validator. We'll print errors ourselves.
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	newName := args[0]
	inputFile := args[1]
	outputFile := args[2]

	// Validate name
	if err := ValidateName(newName); err != nil {
		cmd.PrintErrln("Error:", err)
		return fmt.Errorf("") // Return empty error to set exit code
	}

	// Validate input file
	if err := ValidateInputFile(inputFile); err != nil {
		cmd.PrintErrln("Error:", err)
		return fmt.Errorf("") // Return empty error to set exit code
	}

	// Prompt for overwrite if needed
	if shouldContinue, err := PromptOverwrite(cmd, outputFile); err != nil || !shouldContinue {
		return err
	}

	// Print processing header
	config := ProcessingConfig{
		NewName: newName,
		Themes:  renameThemeFilter,
	}
	PrintProcessingHeader(cmd, inputFile, config)

	themesRenamed, err := RenameColorScheme(inputFile, outputFile, newName, renameThemeFilter)
	if err != nil {
		cmd.PrintErrf("\nError: %v\n", err)
		return fmt.Errorf("") // Return empty error to set exit code
	}

	PrintSuccess(cmd, themesRenamed, "theme(s)", outputFile)

	return nil
}
