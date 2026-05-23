package main

import (
	"bytes"
	"os"
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

func TestThemeSetCommand(t *testing.T) {
	skipIfNoFixture(t)

	outFile := filepath.Join(t.TempDir(), "out.pptx")

	var out bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"theme", "set", testPPTX, outFile, "--theme", "theme2", "--name", `Deck " ' < > &`})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("theme set command failed: %v; stderr=%s", err, errBuf.String())
	}

	themes, err := ReadThemes(outFile)
	if err != nil {
		t.Fatalf("ReadThemes on output failed: %v", err)
	}
	if got := themeByFile(t, themes, "theme2.xml").ThemeName; got != `Deck " ' < > &` {
		t.Fatalf("theme2 theme name = %q", got)
	}

	output := out.String()
	for _, want := range []string{
		"Theme target:  theme2",
		`Theme name:    Deck " ' < > &`,
		"Modified 1 theme(s).",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in output, got:\n%s", want, output)
		}
	}
}

func TestThemeSetCommand_Validation(t *testing.T) {
	skipIfNoFixture(t)

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing name",
			args:    []string{"theme", "set", testPPTX, filepath.Join(t.TempDir(), "missing-name.pptx"), "--theme", "theme1"},
			wantErr: "--name is required",
		},
		{
			name:    "missing theme",
			args:    []string{"theme", "set", testPPTX, filepath.Join(t.TempDir(), "missing-theme.pptx"), "--name", "Renamed"},
			wantErr: "--theme is required when using theme set",
		},
		{
			name:    "notes master xml target",
			args:    []string{"theme", "set", testPPTX, filepath.Join(t.TempDir(), "notes-theme.pptx"), "--theme", "theme4.xml", "--name", "Renamed"},
			wantErr: "slide-master theme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			var errBuf bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&errBuf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()
			if err == nil {
				t.Fatal("expected error")
			}

			combined := out.String() + errBuf.String()
			if !strings.Contains(combined, tt.wantErr) {
				t.Fatalf("output = %q, want substring %q", combined, tt.wantErr)
			}
		})
	}
}

func TestThemeSetCommand_OverwriteDeclineIsNotError(t *testing.T) {
	skipIfNoFixture(t)

	outFile := filepath.Join(t.TempDir(), "out.pptx")
	if err := os.WriteFile(outFile, []byte("existing"), 0644); err != nil {
		t.Fatalf("write existing output failed: %v", err)
	}

	var out bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetIn(bytes.NewBufferString("n\n"))
	rootCmd.SetArgs([]string{"theme", "set", testPPTX, outFile, "--theme", "theme1", "--name", "Renamed"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected nil error on user abort, got %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Aborted.") {
		t.Fatalf("expected abort message, got:\n%s", output)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", errBuf.String())
	}

	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file failed: %v", err)
	}
	if string(got) != "existing" {
		t.Fatalf("expected existing file to remain untouched, got %q", string(got))
	}
}
