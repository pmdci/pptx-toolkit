// theme_cmd.go contains Cobra command wiring for the theme domain.
package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var themeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Theme-related operations",
	Long:  "Inspect and modify PowerPoint theme definitions.",
}

var themeListCmd = &cobra.Command{
	Use:   "list <input.pptx>",
	Short: "List themes in a PowerPoint file",
	Args:  cobra.ExactArgs(1),
	RunE:  runThemeList,
}

var themeSetCmd = &cobra.Command{
	Use:   "set <input.pptx> <output.pptx>",
	Short: "Set theme properties",
	Long: `Set theme properties in a PowerPoint file.

Sets the human-readable theme name on a single slide-master-bound theme.

Examples:
  # Rename one theme
  pptx-toolkit theme set input.pptx output.pptx --theme theme2 --name "Contoso Blue II Deck"

  # Theme filter also accepts the .xml suffix
  pptx-toolkit theme set input.pptx output.pptx --theme theme2.xml --name "AdventureWorks Deck"`,
	Args: cobra.ExactArgs(2),
	RunE: runThemeSet,
}

var themeColorCmd = &cobra.Command{
	Use:     "color",
	Aliases: []string{"colour"},
	Short:   "Theme color operations",
	Long:    "Inspect and modify theme-defined color schemes.",
}

var themeColorListCmd = &cobra.Command{
	Use:   "list <input.pptx>",
	Short: "List theme color schemes in a PowerPoint file",
	Args:  cobra.ExactArgs(1),
	RunE:  runThemeColorList,
}

var themeColorSetCmd = &cobra.Command{
	Use:   "set <input.pptx> <output.pptx>",
	Short: "Set theme colour scheme values",
	Long: `Set the colour scheme name in matching themes.

Examples:
  # Rename in all themes
  pptx-toolkit theme color set input.pptx output.pptx --scheme-name "Azure Blue"

  # Rename in specific theme
  pptx-toolkit theme color set input.pptx output.pptx --scheme-name "Corporate Brand" --theme theme1

  # Rename in multiple themes
  pptx-toolkit theme color set input.pptx output.pptx --scheme-name "New Scheme" --theme theme1,theme2`,
	Args: cobra.ExactArgs(2),
	RunE: runThemeColorSet,
}

var (
	themeListFilter      []string
	themeSetFilter       []string
	themeSetName         string
	themeColorListFilter []string
	themeColorSetFilter  []string
	themeColorSetName    string
)

func init() {
	themeCmd.AddCommand(themeListCmd)
	themeCmd.AddCommand(themeSetCmd)
	themeCmd.AddCommand(themeColorCmd)
	themeCmd.AddCommand(themeFontCmd)
	themeColorCmd.AddCommand(themeColorListCmd)
	themeColorCmd.AddCommand(themeColorSetCmd)

	themeListCmd.Flags().StringSliceVar(&themeListFilter, "theme", nil, "Comma-separated list of themes to target (e.g., theme1,theme2)")
	themeSetCmd.Flags().StringSliceVar(&themeSetFilter, "theme", nil, "Theme to target (required; accepts theme1 or theme1.xml)")
	themeSetCmd.Flags().StringVar(&themeSetName, "name", "", "Set the theme name")
	themeColorListCmd.Flags().StringSliceVar(&themeColorListFilter, "theme", nil, "Comma-separated list of themes to target (e.g., theme1,theme2)")
	themeColorSetCmd.Flags().StringVar(&themeColorSetName, "scheme-name", "", "Set the colour scheme name")
	themeColorSetCmd.Flags().StringSliceVar(&themeColorSetFilter, "theme", nil, "Comma-separated list of themes to target (e.g., theme1,theme2)")
}

func runThemeList(cmd *cobra.Command, args []string) error {
	defer resetThemeListFlags(cmd)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	if err := ValidateInputFile(args[0]); err != nil {
		cmd.PrintErrln("Error:", err)
		return fmt.Errorf("")
	}

	summaries, err := ReadThemeSummaries(args[0], themeListFilter)
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}

	cmd.Printf("\nFound %d theme(s) in %s:\n\n", len(summaries), args[0])
	for _, summary := range summaries {
		cmd.Printf("━━━ %s ━━━\n", summary.Theme.FileName)
		cmd.Printf("Theme:        %s\n", summary.Theme.ThemeName)
		cmd.Printf("Color Scheme: %s\n", summary.Theme.ColorSchemeName)
		cmd.Printf("Font Scheme:  %s\n", summary.Theme.FontSchemeName)
		cmd.Println()
		cmd.Println("Bindings:")
		printBindings(cmd, summary.Bindings)
		cmd.Println()
	}

	return nil
}

func resetThemeListFlags(_ *cobra.Command) {
	themeListFilter = nil
}

func resetThemeSetFlags(cmd *cobra.Command) {
	themeSetFilter = nil
	themeSetName = ""

	if flag := cmd.Flags().Lookup("name"); flag != nil {
		_ = flag.Value.Set("")
		flag.Changed = false
	}
	if flag := cmd.Flags().Lookup("theme"); flag != nil {
		flag.Changed = false
	}
}

func runThemeColorList(cmd *cobra.Command, args []string) error {
	return printThemeColorList(cmd, args[0], themeColorListFilter)
}

func runThemeSet(cmd *cobra.Command, args []string) error {
	defer resetThemeSetFlags(cmd)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	inputFile := args[0]
	outputFile := args[1]

	if themeSetName == "" {
		cmd.PrintErrln("Error: --name is required")
		return fmt.Errorf("")
	}
	targetTheme, err := normalizeSingleThemeTarget(themeSetFilter)
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}
	err = PrepareMutation(cmd, inputFile, outputFile)
	aborted, err := ignoreMutationAborted(err)
	if aborted {
		return nil
	}
	if err != nil {
		return err
	}

	cmd.Printf("Setting theme properties in %s → %s\n\n", inputFile, outputFile)
	cmd.Printf("  Theme target:  %s\n", strings.TrimSuffix(targetTheme, ".xml"))
	cmd.Printf("  Theme name:    %s\n", themeSetName)

	count, err := SetThemeName(inputFile, outputFile, themeSetName, themeSetFilter)
	if err != nil {
		cmd.PrintErrf("\nError: %v\n", err)
		return fmt.Errorf("")
	}

	cmd.Printf("\nModified %d theme(s).\n", count)
	cmd.Printf("Saved to %s\n", outputFile)
	return nil
}

func runThemeColorSet(cmd *cobra.Command, args []string) error {
	defer resetThemeColorSetFlags(cmd)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	inputFile := args[0]
	outputFile := args[1]
	newName := themeColorSetName

	if newName == "" {
		cmd.PrintErrln("Error: --scheme-name is required")
		return fmt.Errorf("")
	}

	err := PrepareMutation(cmd, inputFile, outputFile)
	aborted, err := ignoreMutationAborted(err)
	if aborted {
		return nil
	}
	if err != nil {
		return err
	}

	config := ProcessingConfig{
		NewName: newName,
		Themes:  themeColorSetFilter,
	}
	PrintProcessingHeader(cmd, inputFile, config)

	themesRenamed, err := SetColorSchemeName(inputFile, outputFile, newName, themeColorSetFilter)
	if err != nil {
		cmd.PrintErrf("\nError: %v\n", err)
		return fmt.Errorf("")
	}

	PrintSuccess(cmd, themesRenamed, "theme(s)", outputFile)

	return nil
}

func printBindings(cmd *cobra.Command, bindings []MasterBinding) {
	if len(bindings) == 0 {
		cmd.Println("  Unbound")
		return
	}

	slides, notes, handouts, unknown := groupedBindings(bindings)
	if len(slides) > 0 {
		cmd.Printf("  %-15s %s\n", bindingLabel(masterTypeSlide, len(slides)), strings.Join(slides, ", "))
	}
	if len(notes) > 0 {
		cmd.Printf("  %-15s %s\n", bindingLabel(masterTypeNotes, len(notes)), strings.Join(notes, ", "))
	}
	if len(handouts) > 0 {
		cmd.Printf("  %-15s %s\n", bindingLabel(masterTypeHandout, len(handouts)), strings.Join(handouts, ", "))
	}
	if len(unknown) > 0 {
		cmd.Printf("  %-15s %s\n", bindingLabel("", len(unknown)), strings.Join(unknown, ", "))
	}
}

func resetThemeColorSetFlags(cmd *cobra.Command) {
	themeColorSetFilter = nil
	themeColorSetName = ""

	if flag := cmd.Flags().Lookup("scheme-name"); flag != nil {
		_ = flag.Value.Set("")
		flag.Changed = false
	}
	if flag := cmd.Flags().Lookup("theme"); flag != nil {
		flag.Changed = false
	}
}

func bindingLabel(masterType string, count int) string {
	switch masterType {
	case masterTypeSlide:
		if count == 1 {
			return "Slide master:"
		}
		return "Slide masters:"
	case masterTypeNotes:
		if count == 1 {
			return "Notes master:"
		}
		return "Notes masters:"
	case masterTypeHandout:
		if count == 1 {
			return "Handout master:"
		}
		return "Handout masters:"
	default:
		if count == 1 {
			return "Binding:"
		}
		return "Bindings:"
	}
}
