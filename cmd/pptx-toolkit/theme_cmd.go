// theme_cmd.go contains Cobra command wiring for the theme domain.
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var themeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Theme-related operations",
	Long:  "Inspect and modify PowerPoint theme definitions.",
}

var themeColorCmd = &cobra.Command{
	Use:   "color",
	Short: "Theme color operations",
	Long:  "Inspect and modify theme-defined color schemes.",
}

var themeColorListCmd = &cobra.Command{
	Use:   "list <input.pptx>",
	Short: "List theme color schemes in a PowerPoint file",
	Args:  cobra.ExactArgs(1),
	RunE:  runThemeColorList,
}

var themeColorRenameCmd = &cobra.Command{
	Use:   "rename <new-name> <input.pptx> <output.pptx>",
	Short: "Rename colour scheme(s)",
	Long: `Rename colour scheme(s) in themes.

By default, renames the colour scheme in all themes. Use --theme to target specific themes.

Examples:
  # Rename in all themes
  pptx-toolkit theme color rename "Azure Blue" input.pptx output.pptx

  # Rename in specific theme
  pptx-toolkit theme color rename "Corporate Brand" input.pptx output.pptx --theme theme1

  # Rename in multiple themes
  pptx-toolkit theme color rename "New Scheme" input.pptx output.pptx --theme theme1,theme2`,
	Args: cobra.ExactArgs(3),
	RunE: runThemeColorRename,
}

var (
	themeColorListFilter   []string
	themeColorRenameFilter []string
)

func init() {
	themeCmd.AddCommand(themeColorCmd)
	themeCmd.AddCommand(themeFontCmd)
	themeColorCmd.AddCommand(themeColorListCmd)
	themeColorCmd.AddCommand(themeColorRenameCmd)

	themeColorListCmd.Flags().StringSliceVar(&themeColorListFilter, "theme", nil, "Comma-separated list of themes to target (e.g., theme1,theme2)")
	themeColorRenameCmd.Flags().StringSliceVar(&themeColorRenameFilter, "theme", nil, "Comma-separated list of themes to target (e.g., theme1,theme2)")
}

func runThemeColorList(cmd *cobra.Command, args []string) error {
	return printThemeColorList(cmd, args[0], themeColorListFilter)
}

func runThemeColorRename(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	newName := args[0]
	inputFile := args[1]
	outputFile := args[2]

	if err := ValidateName(newName); err != nil {
		cmd.PrintErrln("Error:", err)
		return fmt.Errorf("")
	}

	if err := PrepareMutation(cmd, inputFile, outputFile); err != nil {
		return err
	}

	config := ProcessingConfig{
		NewName: newName,
		Themes:  themeColorRenameFilter,
	}
	PrintProcessingHeader(cmd, inputFile, config)

	themesRenamed, err := RenameColorScheme(inputFile, outputFile, newName, themeColorRenameFilter)
	if err != nil {
		cmd.PrintErrf("\nError: %v\n", err)
		return fmt.Errorf("")
	}

	PrintSuccess(cmd, themesRenamed, "theme(s)", outputFile)

	return nil
}
