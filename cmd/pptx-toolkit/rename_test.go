package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	toolkitpptx "github.com/pmdci/pptx-toolkit/internal/pptx"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "empty", input: "", wantErr: "name cannot be empty"},
		{name: "basic", input: "Corporate Brand"},
		{name: "punctuation once blacklisted", input: `. / \ ? : *`},
		{name: "xml characters", input: `" ' < > &`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateName(%q) returned error: %v", tt.input, err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("ValidateName(%q) error = %v, want %q", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestSetColorSchemeName_EscapesAndRerename(t *testing.T) {
	skipIfNoFixture(t)

	const firstName = `. / \ ? : *   " ' < > &`
	const secondName = `Second & Final`

	firstOutput := filepath.Join(t.TempDir(), "first.pptx")
	count, err := SetColorSchemeName(testPPTX, firstOutput, firstName, []string{"theme1"})
	if err != nil {
		t.Fatalf("SetColorSchemeName first pass failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("first pass count = %d, want 1", count)
	}

	themes, err := ReadThemes(firstOutput)
	if err != nil {
		t.Fatalf("ReadThemes after first pass failed: %v", err)
	}
	if got := themeByFile(t, themes, "theme1.xml").ColorSchemeName; got != firstName {
		t.Fatalf("first pass scheme name = %q, want %q", got, firstName)
	}

	extracted := t.TempDir()
	if err := toolkitpptx.ExtractPPTX(firstOutput, extracted); err != nil {
		t.Fatalf("ExtractPPTX failed: %v", err)
	}
	rawTheme, err := os.ReadFile(filepath.Join(extracted, "ppt", "theme", "theme1.xml"))
	if err != nil {
		t.Fatalf("read extracted theme failed: %v", err)
	}
	wantEscaped := `name=". / \ ? : *   &quot; &apos; &lt; &gt; &amp;"`
	if !bytes.Contains(rawTheme, []byte(wantEscaped)) {
		t.Fatalf("expected escaped color scheme name %q in theme1.xml, got:\n%s", wantEscaped, rawTheme)
	}

	secondOutput := filepath.Join(t.TempDir(), "second.pptx")
	count, err = SetColorSchemeName(firstOutput, secondOutput, secondName, []string{"theme1"})
	if err != nil {
		t.Fatalf("SetColorSchemeName second pass failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("second pass count = %d, want 1", count)
	}

	themes, err = ReadThemes(secondOutput)
	if err != nil {
		t.Fatalf("ReadThemes after second pass failed: %v", err)
	}
	if got := themeByFile(t, themes, "theme1.xml").ColorSchemeName; got != secondName {
		t.Fatalf("second pass scheme name = %q, want %q", got, secondName)
	}
}

func TestSetColorSchemeName_SkipsUnnamedClrScheme(t *testing.T) {
	skipIfNoFixture(t)

	tempDir := t.TempDir()
	if err := toolkitpptx.ExtractPPTX(testPPTX, tempDir); err != nil {
		t.Fatalf("ExtractPPTX failed: %v", err)
	}

	themePath := filepath.Join(tempDir, "ppt", "theme", "theme1.xml")
	themeXML, err := os.ReadFile(themePath)
	if err != nil {
		t.Fatalf("read theme1.xml failed: %v", err)
	}
	themeXML = bytes.Replace(themeXML, []byte(`<a:clrScheme name="Office">`), []byte(`<a:clrScheme>`), 1)
	if err := os.WriteFile(themePath, themeXML, 0644); err != nil {
		t.Fatalf("write theme1.xml failed: %v", err)
	}

	inputPath := filepath.Join(t.TempDir(), "unnamed-clrscheme-input.pptx")
	if err := toolkitpptx.RepackPPTX(tempDir, inputPath); err != nil {
		t.Fatalf("RepackPPTX failed: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "out.pptx")
	_, err = SetColorSchemeName(inputPath, outputPath, "Should Not Apply", []string{"theme1"})
	if err == nil {
		t.Fatal("expected rename to skip unnamed clrScheme and return an error")
	}
	if err.Error() != "no themes were renamed (this might indicate an issue with the theme filter)" {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected no output file on skipped rename, stat err=%v", statErr)
	}
}

func themeByFile(t *testing.T, themes []*Theme, fileName string) *Theme {
	t.Helper()
	for _, theme := range themes {
		if theme.FileName == fileName {
			return theme
		}
	}
	t.Fatalf("theme %s not found", fileName)
	return nil
}
