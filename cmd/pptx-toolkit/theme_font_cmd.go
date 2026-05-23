package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var themeFontCmd = &cobra.Command{
	Use:   "font",
	Short: "Theme font operations",
	Long:  "Inspect and modify theme-defined font schemes.",
}

var themeFontListCmd = &cobra.Command{
	Use:   "list <input.pptx>",
	Short: "List theme font schemes in a PowerPoint file",
	Args:  cobra.ExactArgs(1),
	RunE:  runThemeFontList,
}

var themeFontSetCmd = &cobra.Command{
	Use:   "set <input.pptx> <output.pptx>",
	Short: "Set theme font scheme values",
	Args:  cobra.ExactArgs(2),
	RunE:  runThemeFontSet,
}

var (
	themeFontListFilter []string
	themeFontSetFilter  []string
	themeFontSetMajor   string
	themeFontSetMinor   string
	themeFontSetName    string
)

func init() {
	themeFontCmd.AddCommand(themeFontListCmd)
	themeFontCmd.AddCommand(themeFontSetCmd)

	themeFontListCmd.Flags().StringSliceVar(&themeFontListFilter, "theme", nil, "Comma-separated list of themes to target (e.g., theme1,theme2)")
	themeFontSetCmd.Flags().StringVar(&themeFontSetMajor, "major", "", "Set the major (headings) latin typeface")
	themeFontSetCmd.Flags().StringVar(&themeFontSetMinor, "minor", "", "Set the minor (body) latin typeface")
	themeFontSetCmd.Flags().StringVar(&themeFontSetName, "scheme-name", "", "Set the font scheme name")
	themeFontSetCmd.Flags().StringSliceVar(&themeFontSetFilter, "theme", nil, "Comma-separated list of themes to target (e.g., theme1,theme2)")
}

func runThemeFontList(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	defer resetThemeFontFlags(cmd)

	themeFilter, err := cmd.Flags().GetStringSlice("theme")
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}

	if err := ValidateInputFile(args[0]); err != nil {
		cmd.PrintErrln("Error:", err)
		return fmt.Errorf("")
	}

	schemes, err := ReadFontSchemes(args[0], themeFilter)
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}

	cmd.Printf("\nFound %d theme(s) in %s:\n\n", len(schemes), args[0])
	for _, scheme := range schemes {
		cmd.Printf("━━━ %s ━━━\n", scheme.FileName)
		cmd.Printf("Theme:        %s\n", scheme.ThemeName)
		cmd.Printf("Font Scheme:  %s\n", scheme.SchemeName)
		cmd.Println()
		cmd.Println("Fonts:")
		cmd.Printf("  major  (headings):  %s\n", scheme.MajorTypeface)
		cmd.Printf("  minor  (body):      %s\n", scheme.MinorTypeface)
		cmd.Println()
	}

	return nil
}

func runThemeFontSet(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	defer resetThemeFontFlags(cmd)

	inputFile := args[0]
	outputFile := args[1]

	themeFilter, err := cmd.Flags().GetStringSlice("theme")
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}
	major, err := cmd.Flags().GetString("major")
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}
	minor, err := cmd.Flags().GetString("minor")
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}
	schemeName, err := cmd.Flags().GetString("scheme-name")
	if err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return fmt.Errorf("")
	}

	if major == "" && minor == "" && schemeName == "" {
		cmd.PrintErrln("Error: at least one of --major, --minor, or --scheme-name is required")
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

	cmd.Printf("Setting fonts in %s → %s\n\n", inputFile, outputFile)
	if major != "" {
		cmd.Printf("  Major (headings):  %s\n", major)
	}
	if minor != "" {
		cmd.Printf("  Minor (body):      %s\n", minor)
	}
	if schemeName != "" {
		cmd.Printf("  Scheme name:       %s\n", schemeName)
	}
	if len(themeFilter) > 0 {
		cmd.Printf("  Theme filter:      %s\n", themeFilter[0])
		for _, theme := range themeFilter[1:] {
			cmd.Printf("                     %s\n", theme)
		}
	}

	count, err := ApplyFontScheme(inputFile, outputFile, FontSchemeUpdate{
		Major:      major,
		Minor:      minor,
		SchemeName: schemeName,
	}, themeFilter)
	if err != nil {
		cmd.PrintErrf("\nError: %v\n", err)
		return fmt.Errorf("")
	}

	cmd.Printf("\nModified %d theme(s).\n", count)
	cmd.Printf("Saved to %s\n", outputFile)
	return nil
}

func resetThemeFontFlags(cmd *cobra.Command) {
	themeFontListFilter = nil
	themeFontSetFilter = nil
	themeFontSetMajor = ""
	themeFontSetMinor = ""
	themeFontSetName = ""

	// StringSlice flags (--theme) are reset via the nil assignments above.
	// pflag's StringSlice.Set() appends rather than replaces, so calling
	// Set("") would leave the var as [""] rather than nil.
	resets := map[string]string{
		"major":       "",
		"minor":       "",
		"scheme-name": "",
	}

	for name, value := range resets {
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			continue
		}
		_ = flag.Value.Set(value)
		flag.Changed = false
	}

	if f := cmd.Flags().Lookup("theme"); f != nil {
		f.Changed = false
	}
}
