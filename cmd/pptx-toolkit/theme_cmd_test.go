package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestThemeColorListCommand_ColourAlias(t *testing.T) {
	skipIfNoFixture(t)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"theme", "colour", "list", "--theme", "theme1", testPPTX})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list command via colour alias failed: %v; stderr=%s", err, errBuf.String())
	}

	output := out.String()
	if !strings.Contains(output, "━━━ theme1.xml ━━━") {
		t.Fatalf("expected theme1 block in output, got:\n%s", output)
	}
	if strings.Contains(output, "theme2.xml") {
		t.Fatalf("expected filtered output to omit theme2.xml, got:\n%s", output)
	}
}

func TestThemeColorSetCommand(t *testing.T) {
	skipIfNoFixture(t)

	outFile := filepath.Join(t.TempDir(), "out.pptx")

	var out bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"theme", "color", "set", testPPTX, outFile, "--scheme-name", `Scheme " ' < > &`, "--theme", "theme1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("set command failed: %v; stderr=%s", err, errBuf.String())
	}

	themes, err := ReadThemes(outFile)
	if err != nil {
		t.Fatalf("ReadThemes on output failed: %v", err)
	}
	if got := themeByFile(t, themes, "theme1.xml").ColorSchemeName; got != `Scheme " ' < > &` {
		t.Fatalf("theme1 color scheme name = %q", got)
	}
}

func TestThemeColorSetCommand_RequiresSchemeName(t *testing.T) {
	skipIfNoFixture(t)

	outFile := filepath.Join(t.TempDir(), "out.pptx")

	var out bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"theme", "color", "set", testPPTX, outFile})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing --scheme-name")
	}

	combined := out.String() + errBuf.String()
	if !strings.Contains(combined, "--scheme-name is required") {
		t.Fatalf("output = %q, want missing scheme-name error", combined)
	}
}
