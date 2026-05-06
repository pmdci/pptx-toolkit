package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var layoutCmd = &cobra.Command{
	Use:   "layout",
	Short: "Layout-related operations",
	Long:  "Slide layout operations for PowerPoint files.",
}

var layoutListCmd = &cobra.Command{
	Use:   "list <input.pptx>",
	Short: "List all slide layouts in a PowerPoint file",
	Long: `List all slide layouts in a PowerPoint file.

Each layout shows its Layout ID, Name (p:cSld/@name), Matching Name (p:sldLayout/@matchingName),
the slide master and theme it belongs to, and which slides use it.

Definitions:
  name          The layout name stored in p:cSld/@name, typically shown in Slide Master view
  matching-name The optional layout matching name stored in p:sldLayout/@matchingName,
                often shown in the New Slide / layout picker UI

Examples:
  # List all layouts
  pptx-toolkit layout list input.pptx

  # Filter by layout file
  pptx-toolkit layout list input.pptx --layout-id slideLayout4

  # Filter by name (exact, case-sensitive)
  pptx-toolkit layout list input.pptx --name "CONTACT CARD"

  # Filter by matching name
  pptx-toolkit layout list input.pptx --matching-name "Contacto + soher"

  # Filter by theme
  pptx-toolkit layout list input.pptx --theme theme1`,
	Args: cobra.ExactArgs(1),
	RunE: runLayoutList,
}

var (
	layoutIDFilter    string
	layoutNameFilter  string
	layoutMatchFilter string
	layoutThemeFilter []string
)

func init() {
	layoutCmd.AddCommand(layoutListCmd)

	layoutListCmd.Flags().StringVar(&layoutIDFilter, "layout-id", "", "Filter by layout ID (e.g. slideLayout4)")
	layoutListCmd.Flags().StringVar(&layoutNameFilter, "name", "", "Filter by p:cSld/@name (exact, case-sensitive)")
	layoutListCmd.Flags().StringVar(&layoutMatchFilter, "matching-name", "", "Filter by p:sldLayout/@matchingName (exact, case-sensitive)")
	layoutListCmd.Flags().StringSliceVar(&layoutThemeFilter, "theme", nil, "Comma-separated list of themes to target (e.g. theme1,theme2)")
}

func runLayoutList(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	inputFile := args[0]

	if err := ValidateInputFile(inputFile); err != nil {
		cmd.PrintErrln("Error:", err)
		return fmt.Errorf("")
	}

	filters := LayoutFilters{
		LayoutID:     layoutIDFilter,
		Name:         layoutNameFilter,
		MatchingName: layoutMatchFilter,
		Theme:        layoutThemeFilter,
	}

	layouts, err := ReadLayouts(inputFile, filters)
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}

	if len(layouts) == 0 {
		cmd.Println("No layouts found matching the specified filters.")
		return nil
	}

	cmd.Printf("\nFound %d layout(s) in %s:\n\n", len(layouts), inputFile)

	for _, l := range layouts {
		cmd.Printf("━━━ %s ━━━\n", l.FileName)
		cmd.Printf("%-15s %s\n", "Layout ID:", l.LayoutID)
		cmd.Printf("%-15s %s\n", "Name:", l.Name)

		matchingName := l.MatchingName
		if matchingName == "" {
			matchingName = "<none>"
		}
		cmd.Printf("%-15s %s\n", "Matching Name:", matchingName)

		master := l.Master
		if master == "" {
			master = "<none>"
		}
		cmd.Printf("%-15s %s\n", "Master:", master)

		theme := l.Theme
		if theme == "" {
			theme = "<none>"
		}
		cmd.Printf("%-15s %s\n", "Theme:", theme)

		if len(l.UsedBySlides) > 0 {
			slideStrs := make([]string, len(l.UsedBySlides))
			for i, n := range l.UsedBySlides {
				slideStrs[i] = fmt.Sprintf("%d", n)
			}
			cmd.Printf("%-15s %s\n", "Used By Slides:", strings.Join(slideStrs, ", "))
		} else {
			cmd.Printf("%-15s %s\n", "Used By Slides:", "none")
		}

		cmd.Println()
	}

	return nil
}
