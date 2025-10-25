package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// ProcessingConfig holds configuration for processing operations
type ProcessingConfig struct {
	Mappings      []string // Color mappings (e.g., ["accent1→accent3"])
	NewName       string   // New name for rename operations
	Themes        []string // Theme filter or nil for all
	Slides        []int    // Slide filter or nil for all
	SlidesMatched *int     // Number of slides matched (nil if not applicable)
	Scope         string   // "all", "content", "master"
}

// ValidateInputFile checks if the input file exists
func ValidateInputFile(inputFile string) error {
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file not found: %s", inputFile)
	}
	return nil
}

// PromptOverwrite prompts the user if the output file already exists
// Returns true if user wants to overwrite, false if aborted
func PromptOverwrite(cmd *cobra.Command, outputFile string) (bool, error) {
	if _, err := os.Stat(outputFile); err == nil {
		// File exists, prompt for overwrite
		cmd.Printf("Output file '%s' already exists. Overwrite? (y/n): ", outputFile)
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			cmd.Println("Aborted.")
			return false, nil
		}
	}
	return true, nil
}

// PrintProcessingHeader prints a consistent header showing what will be processed
func PrintProcessingHeader(cmd *cobra.Command, inputFile string, config ProcessingConfig) {
	cmd.Printf("Processing %s...\n", inputFile)

	// Print mappings if present
	if len(config.Mappings) > 0 {
		cmd.Printf("Mappings: %s\n", strings.Join(config.Mappings, ", "))
	}

	// Print new name if present (for rename operations)
	if config.NewName != "" {
		cmd.Printf("New colour scheme name: %s\n", config.NewName)
	}

	// Print theme filter
	if len(config.Themes) > 0 {
		cmd.Printf("Themes: %s\n", strings.Join(config.Themes, ", "))
	} else {
		cmd.Println("Themes: all")
	}

	// Print slide filter
	if len(config.Slides) > 0 {
		cmd.Printf("Slides: %s\n", formatSlides(config.Slides))
	}

	// Print scope (only when not default "all")
	if config.Scope != "" && config.Scope != "all" {
		cmd.Printf("Scope: %s\n", config.Scope)
	}

	// Print matched slides feedback (only when both --slides and --theme are used)
	if config.SlidesMatched != nil {
		switch *config.SlidesMatched {
        case 0:
                cmd.Println("→ No slides matched the theme filter")
        case 1:
                cmd.Println("→ 1 slide matched")
        default:
                cmd.Printf("→ %d slides matched\n", *config.SlidesMatched)
        }
    }
}

// PrintSuccess prints a consistent success message
func PrintSuccess(cmd *cobra.Command, itemsProcessed int, itemType string, outputFile string) {
	cmd.Printf("✓ Successfully processed %d %s\n", itemsProcessed, itemType)
	cmd.Printf("✓ Output saved to %s\n", outputFile)
}

// formatSlides formats a slice of slide numbers for display
// Examples: [1,3,5,6,7,8] → "1, 3, 5-8"
func formatSlides(slides []int) string {
	if len(slides) == 0 {
		return "all"
	}

	// For simplicity, just join with commas for now
	// Could add range compression (1,2,3 → 1-3) as enhancement
	parts := make([]string, len(slides))
	for i, slide := range slides {
		parts[i] = fmt.Sprintf("%d", slide)
	}
	return strings.Join(parts, ", ")
}
