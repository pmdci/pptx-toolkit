package main

import (
	"fmt"
	"os"
	"strings"

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

Examples:
  # Scheme to scheme
  pptx-toolkit color swap "accent1:accent3" input.pptx output.pptx

  # Scheme to hex
  pptx-toolkit color swap "accent1:BBFFCC" input.pptx output.pptx

  # Hex to scheme
  pptx-toolkit color swap "AABBCC:accent2" input.pptx output.pptx

  # Hex to hex
  pptx-toolkit color swap "FF0000:00FF00" input.pptx output.pptx

  # Multiple mappings
  pptx-toolkit color swap "accent1:BBFFCC,AABBCC:accent2,FF0000:00FF00" input.pptx output.pptx

  # Filter by theme
  pptx-toolkit color swap "accent1:BBFFCC" input.pptx output.pptx --theme theme1`,
	Args: cobra.ExactArgs(3),
	RunE: runColorSwap,
}

var (
	themeFilter []string
)

func init() {
	colorCmd.AddCommand(colorListCmd)
	colorCmd.AddCommand(colorSwapCmd)

	// Add --theme flag to swap command
	colorSwapCmd.Flags().StringSliceVar(&themeFilter, "theme", nil, "Comma-separated list of themes to target (e.g., theme1,theme2)")
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
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		cmd.PrintErrln("Error: input file not found:", inputFile)
		return fmt.Errorf("") // Return empty error to set exit code
	}

	// Validate output file
	if _, err := os.Stat(outputFile); err == nil {
		// File exists, prompt for overwrite
		cmd.Printf("Output file '%s' already exists. Overwrite? (y/n): ", outputFile)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			cmd.Println("Aborted.")
			return nil
		}
	}

	// Parse color mapping
	colorMapping, err := ParseColorMapping(mappingStr)
	if err != nil {
		cmd.PrintErrln("Error:", err)
		return fmt.Errorf("") // Return empty error to set exit code
	}

	// Process the file
	cmd.Printf("Processing %s...\n", inputFile)

	// Format mappings for display
	var mappingStrs []string
	for source, target := range colorMapping {
		mappingStrs = append(mappingStrs, fmt.Sprintf("%s→%s", source, target))
	}
	cmd.Printf("Mappings: %s\n", strings.Join(mappingStrs, ", "))

	if len(themeFilter) > 0 {
		cmd.Printf("Themes: %s\n", strings.Join(themeFilter, ", "))
	} else {
		cmd.Println("Themes: all")
	}

	filesProcessed, err := ProcessPPTX(inputFile, outputFile, colorMapping, themeFilter)
	if err != nil {
		cmd.PrintErrf("\nError: %v\n", err)
		return fmt.Errorf("") // Return empty error to set exit code
	}

	cmd.Printf("✓ Successfully processed %d files\n", filesProcessed)
	cmd.Printf("✓ Output saved to %s\n", outputFile)

	return nil
}
