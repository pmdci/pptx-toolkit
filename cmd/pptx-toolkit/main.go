package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set via ldflags during build
var Version = "dev"

const versionBanner = `pptx-toolkit %s

╭━━━┳━━━┳━━━━┳━╮╭━╮╭━━━━╮╱╱╱╱╭╮╭╮╱╱╭╮
┃╭━╮┃╭━╮┃╭╮╭╮┣╮╰╯╭╯┃╭╮╭╮┃╱╱╱╱┃┃┃┃╱╭╯╰╮ Copyright (C) 2025 Pedro Innecco
┃╰━╯┃╰━╯┣╯┃┃╰╯╰╮╭╯╱╰╯┃┃┣┻━┳━━┫┃┃┃╭╋╮╭╯ <https://pedroinnecco.com>
┃╭━━┫╭━━╯╱┃┃╱╱╭╯╰┳━━╮┃┃┃╭╮┃╭╮┃┃┃╰╯╋┫┃
┃┃╱╱┃┃╱╱╱╱┃┃╱╭╯╭╮╰┳━╯┃┃┃╰╯┃╰╯┃╰┫╭╮┫┃╰╮
╰╯╱╱╰╯╱╱╱╱╰╯╱╰━╯╰━╯╱╱╰╯╰━━┻━━┻━┻╯╰┻┻━╯

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program comes with ABSOLUTELY NO WARRANTY.
See <https://www.gnu.org/licenses/gpl-3.0.html> for details.

Source: https://github.com/pmdci/pptx-toolkit

Brought to you by the letter P.`

var rootCmd = &cobra.Command{
	Use:   "pptx-toolkit",
	Short: "Microsoft® PowerPoint toolkit for colors, themes, and other utilities",
	Long:  "Microsoft® PowerPoint manipulation toolkit.\n\nUse \"pptx-toolkit <group> <command> --help\" for command-specific help.",
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "version for pptx-toolkit")
	rootCmd.AddCommand(colorCmd)
	// Silence errors - subcommands print their own errors
	rootCmd.SilenceErrors = true
}

func main() {
	// Check for version flag before cobra processes it
	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--version" {
			fmt.Printf(versionBanner, Version)
			fmt.Println()
			return
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
