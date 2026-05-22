// layout_cmd.go contains Cobra command wiring for the layout domain.
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

var layoutRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove layout properties",
	Long:  "Remove properties from slide layouts in PowerPoint files.",
}

var layoutSetCmd = &cobra.Command{
	Use:   "set <source>:<target> <input.pptx> <output.pptx>",
	Short: "Set layout properties",
	Long: `Set slide layout properties in a PowerPoint file.

The mapping is directional:
  source:target

Source may be either:
  @name             Copy from p:cSld/@name
  @matching-name    Copy from p:sldLayout/@matchingName
  Literal string    Set a literal value

Target must be one of:
  name
  matching-name

Without filters, the assignment is applied to all layouts. Filters narrow which
layouts are affected.

Examples:
  # Copy name into matching-name for all layouts
  pptx-toolkit layout set @name:matching-name input.pptx output.pptx

  # Copy matching-name into name where present
  pptx-toolkit layout set @matching-name:name input.pptx output.pptx

  # Set a literal matching-name on one layout
  pptx-toolkit layout set "Layout with matchName property":matching-name input.pptx output.pptx --layout-id slideLayout12`,
	Args: cobra.ExactArgs(3),
	RunE: runLayoutSet,
}

var layoutRemoveMatchingNameCmd = &cobra.Command{
	Use:   "matching-name <input.pptx> <output.pptx>",
	Short: "Remove the matchingName attribute from slide layouts",
	Long: `Remove the p:sldLayout/@matchingName attribute from slide layouts.

Without filters, removes matchingName from all layouts that have it.
Filters narrow which layouts are affected.

Examples:
  # Remove from all layouts
  pptx-toolkit layout remove matching-name input.pptx output.pptx

  # Remove from a specific layout only
  pptx-toolkit layout remove matching-name input.pptx output.pptx --layout-id slideLayout12

  # Remove from layouts with a specific matching name
  pptx-toolkit layout remove matching-name input.pptx output.pptx --matching-name "Contacto + soher"`,
	Args: cobra.ExactArgs(2),
	RunE: runLayoutRemoveMatchingName,
}

func init() {
	layoutCmd.AddCommand(layoutListCmd)
	layoutCmd.AddCommand(layoutSetCmd)
	layoutCmd.AddCommand(layoutRemoveCmd)
	layoutRemoveCmd.AddCommand(layoutRemoveMatchingNameCmd)

	layoutListCmd.Flags().StringVar(&layoutIDFilter, "layout-id", "", "Filter by layout ID (e.g. slideLayout4)")
	layoutListCmd.Flags().StringVar(&layoutNameFilter, "name", "", "Filter by p:cSld/@name (exact, case-sensitive)")
	layoutListCmd.Flags().StringVar(&layoutMatchFilter, "matching-name", "", "Filter by p:sldLayout/@matchingName (exact, case-sensitive)")
	layoutListCmd.Flags().StringSliceVar(&layoutThemeFilter, "theme", nil, "Comma-separated list of themes to target (e.g. theme1,theme2)")

	layoutSetCmd.Flags().String("layout-id", "", "Filter by layout ID (e.g. slideLayout4)")
	layoutSetCmd.Flags().String("name", "", "Filter by p:cSld/@name (exact, case-sensitive)")
	layoutSetCmd.Flags().String("matching-name", "", "Filter by p:sldLayout/@matchingName (exact, case-sensitive)")
	layoutSetCmd.Flags().StringSlice("theme", nil, "Comma-separated list of themes to target (e.g. theme1,theme2)")

	layoutRemoveMatchingNameCmd.Flags().String("layout-id", "", "Filter by layout ID (e.g. slideLayout4)")
	layoutRemoveMatchingNameCmd.Flags().String("name", "", "Filter by p:cSld/@name (exact, case-sensitive)")
	layoutRemoveMatchingNameCmd.Flags().String("matching-name", "", "Filter by p:sldLayout/@matchingName (exact, case-sensitive)")
	layoutRemoveMatchingNameCmd.Flags().StringSlice("theme", nil, "Comma-separated list of themes to target (e.g. theme1,theme2)")
}

func runLayoutList(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	defer resetLayoutFlags(cmd)

	inputFile := args[0]

	if err := ValidateInputFile(inputFile); err != nil {
		cmd.PrintErrln("Error:", err)
		return fmt.Errorf("")
	}

	filters, err := layoutFiltersFromCommand(cmd)
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
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

func layoutFiltersFromCommand(cmd *cobra.Command) (LayoutFilters, error) {
	layoutID, err := cmd.Flags().GetString("layout-id")
	if err != nil {
		return LayoutFilters{}, err
	}

	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return LayoutFilters{}, err
	}

	matchingName, err := cmd.Flags().GetString("matching-name")
	if err != nil {
		return LayoutFilters{}, err
	}

	theme, err := cmd.Flags().GetStringSlice("theme")
	if err != nil {
		return LayoutFilters{}, err
	}

	return LayoutFilters{
		LayoutID:     layoutID,
		Name:         name,
		MatchingName: matchingName,
		Theme:        theme,
	}, nil
}

func runLayoutRemoveMatchingName(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	defer resetLayoutFlags(cmd)

	inputFile, outputFile := args[0], args[1]

	if err := PrepareMutation(cmd, inputFile, outputFile); err != nil {
		return err
	}

	filters, err := layoutFiltersFromCommand(cmd)
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}

	count, err := RemoveLayoutMatchingName(inputFile, outputFile, filters)
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}

	if count == 0 {
		cmd.Println("No layouts with matching-name found matching the specified filters.")
		return nil
	}

	cmd.Printf("Removed matching-name from %d layout(s) in %s\n", count, outputFile)
	return nil
}

func runLayoutSet(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	defer resetLayoutFlags(cmd)

	mappingStr, inputFile, outputFile := args[0], args[1], args[2]

	if err := PrepareMutation(cmd, inputFile, outputFile); err != nil {
		return err
	}

	mapping, err := ParseLayoutSetMapping(mappingStr)
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}

	filters, err := layoutFiltersFromCommand(cmd)
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}

	count, err := SetLayoutProperty(inputFile, outputFile, mapping, filters)
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}

	if count == 0 {
		cmd.Println("No layouts were updated matching the specified filters.")
		return nil
	}

	cmd.Printf("Updated %s on %d layout(s) in %s\n", mapping.TargetProperty, count, outputFile)
	return nil
}

func resetLayoutFlags(cmd *cobra.Command) {
	resets := map[string]string{
		"layout-id":     "",
		"name":          "",
		"matching-name": "",
		"theme":         "",
	}

	for name, value := range resets {
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			continue
		}
		_ = flag.Value.Set(value)
		flag.Changed = false
	}
}
