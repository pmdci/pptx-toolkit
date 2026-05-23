package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	toolkitpptx "github.com/pmdci/pptx-toolkit/internal/pptx"
)

func TestFindThemeNameConflict_UsesRunes(t *testing.T) {
	themes := []*Theme{
		{FileName: "theme1.xml", ThemeName: "Alpha"},
		{FileName: "theme2.xml", ThemeName: "界界界界界界界界界界界界界界界界界界界界Z"},
	}

	if file, _, ok := findThemeNameConflict(themes, "theme1.xml", "界界界界界界界界界界界界界界界界界界界界Y"); !ok || file != "theme2.xml" {
		t.Fatalf("expected rune-based first-20-character conflict, got ok=%v file=%q", ok, file)
	}
}

func TestFindThemeNameConflict_IsCaseInsensitiveAndWhitespaceSensitive(t *testing.T) {
	themes := []*Theme{
		{FileName: "theme1.xml", ThemeName: "mytheme"},
		{FileName: "theme2.xml", ThemeName: "other"},
	}

	for _, candidate := range []string{"MyTheme", "MYTHEME"} {
		if file, _, ok := findThemeNameConflict(themes, "theme2.xml", candidate); !ok || file != "theme1.xml" {
			t.Fatalf("expected case-insensitive conflict for %q, got ok=%v file=%q", candidate, ok, file)
		}
	}

	for _, candidate := range []string{"mytheme ", " mytheme", "my theme"} {
		if _, _, ok := findThemeNameConflict(themes, "theme2.xml", candidate); ok {
			t.Fatalf("expected whitespace to remain significant for %q", candidate)
		}
	}
}

func TestReplaceTitleOfPartsThemeName_DecodesLogicalValue(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes">` +
		`<HeadingPairs><vt:vector size="2" baseType="variant"><vt:variant><vt:lpstr>Theme</vt:lpstr></vt:variant><vt:variant><vt:i4>1</vt:i4></vt:variant></vt:vector></HeadingPairs>` +
		`<TitlesOfParts><vt:vector size="1" baseType="lpstr"><vt:lpstr>Scheme &amp; &lt;One&gt;</vt:lpstr></vt:vector></TitlesOfParts>` +
		`</Properties>`)

	output, err := replaceTitleOfPartsThemeName(input, `Scheme & <One>`, `Second " ' < > &`)
	if err != nil {
		t.Fatalf("replaceTitleOfPartsThemeName failed: %v", err)
	}
	if !bytes.Contains(output, []byte(`Second " ' &lt; &gt; &amp;`)) {
		t.Fatalf("expected escaped replacement in output, got:\n%s", output)
	}

	values, err := titlesOfParts(output)
	if err != nil {
		t.Fatalf("titlesOfParts failed: %v", err)
	}
	if len(values) != 1 || values[0] != `Second " ' < > &` {
		t.Fatalf("titlesOfParts = %#v", values)
	}
}

func TestReplaceTitleOfPartsThemeName_IgnoresWrongTitlesOfPartsNamespace(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes">` +
		`<TitlesOfParts xmlns="urn:not-extended-properties"><vt:vector size="1" baseType="lpstr"><vt:lpstr>Office Theme Deck</vt:lpstr></vt:vector></TitlesOfParts>` +
		`</Properties>`)

	_, err := replaceTitleOfPartsThemeName(input, "Office Theme Deck", "Renamed")
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsAll(err.Error(), "no TitlesOfParts entry matched theme name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetThemeName_UpdatesThemeAndTitlesOfPartsOnly(t *testing.T) {
	skipIfNoFixture(t)

	originalTitles := titlesOfPartsFromPPTX(t, testPPTX)
	outFile := filepath.Join(t.TempDir(), "renamed.pptx")

	count, err := SetThemeName(testPPTX, outFile, "Blue II Deck RENAMED", []string{"theme2"})
	if err != nil {
		t.Fatalf("SetThemeName failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}

	themes, err := ReadThemes(outFile)
	if err != nil {
		t.Fatalf("ReadThemes failed: %v", err)
	}
	if got := themeByFile(t, themes, "theme2.xml").ThemeName; got != "Blue II Deck RENAMED" {
		t.Fatalf("theme2 name = %q", got)
	}

	gotTitles := titlesOfPartsFromPPTX(t, outFile)
	wantTitles := append([]string(nil), originalTitles...)
	replaceOneString(t, wantTitles, "Blue II Deck", "Blue II Deck RENAMED")
	assertStringSlicesEqual(t, gotTitles, wantTitles)
}

func TestSetThemeName_NoOpSucceeds(t *testing.T) {
	skipIfNoFixture(t)

	outFile := filepath.Join(t.TempDir(), "noop.pptx")
	count, err := SetThemeName(testPPTX, outFile, "Office Theme Deck", []string{"theme1"})
	if err != nil {
		t.Fatalf("SetThemeName failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Fatalf("expected output file, stat err=%v", err)
	}
}

func TestSetThemeName_EscapesAndRerename(t *testing.T) {
	skipIfNoFixture(t)

	firstName := `. / \ ? : *   " ' < > &`
	secondName := `Second & Final`

	firstOutput := filepath.Join(t.TempDir(), "first.pptx")
	count, err := SetThemeName(testPPTX, firstOutput, firstName, []string{"theme1"})
	if err != nil {
		t.Fatalf("SetThemeName first pass failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("first pass count = %d, want 1", count)
	}

	themes, err := ReadThemes(firstOutput)
	if err != nil {
		t.Fatalf("ReadThemes first pass failed: %v", err)
	}
	if got := themeByFile(t, themes, "theme1.xml").ThemeName; got != firstName {
		t.Fatalf("first pass theme name = %q, want %q", got, firstName)
	}

	extracted := t.TempDir()
	if err := toolkitpptx.ExtractPPTX(firstOutput, extracted); err != nil {
		t.Fatalf("ExtractPPTX failed: %v", err)
	}
	themeXML, err := os.ReadFile(filepath.Join(extracted, "ppt", "theme", "theme1.xml"))
	if err != nil {
		t.Fatalf("read theme1.xml failed: %v", err)
	}
	if !bytes.Contains(themeXML, []byte(`name=". / \ ? : *   &quot; &apos; &lt; &gt; &amp;"`)) {
		t.Fatalf("expected escaped theme name in theme XML, got:\n%s", themeXML)
	}

	appXML, err := os.ReadFile(filepath.Join(extracted, "docProps", "app.xml"))
	if err != nil {
		t.Fatalf("read app.xml failed: %v", err)
	}
	if !bytes.Contains(appXML, []byte(`. / \ ? : *   " ' &lt; &gt; &amp;`)) {
		t.Fatalf("expected escaped theme name in app.xml, got:\n%s", appXML)
	}
	if bytes.Contains(appXML, []byte(`&quot;`)) || bytes.Contains(appXML, []byte(`&apos;`)) {
		t.Fatalf("expected app.xml text content not to escape quotes, got:\n%s", appXML)
	}

	secondOutput := filepath.Join(t.TempDir(), "second.pptx")
	count, err = SetThemeName(firstOutput, secondOutput, secondName, []string{"theme1"})
	if err != nil {
		t.Fatalf("SetThemeName second pass failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("second pass count = %d, want 1", count)
	}

	themes, err = ReadThemes(secondOutput)
	if err != nil {
		t.Fatalf("ReadThemes second pass failed: %v", err)
	}
	if got := themeByFile(t, themes, "theme1.xml").ThemeName; got != secondName {
		t.Fatalf("second pass theme name = %q, want %q", got, secondName)
	}
}

func TestSetThemeName_RejectsNonSlideMasterThemes(t *testing.T) {
	skipIfNoFixture(t)

	for _, theme := range []string{"theme4", "theme5.xml"} {
		t.Run(theme, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "out.pptx")
			_, err := SetThemeName(testPPTX, outFile, "Should Fail", []string{theme})
			if err == nil {
				t.Fatal("expected error")
			}
			if !containsAll(err.Error(), "slide-master theme") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSetThemeName_AppXMLMissingSucceeds(t *testing.T) {
	skipIfNoFixture(t)

	inputPath := extractedFixtureToPPTX(t, func(tempDir string) {
		if err := os.Remove(filepath.Join(tempDir, "docProps", "app.xml")); err != nil {
			t.Fatalf("remove app.xml failed: %v", err)
		}
	})

	outFile := filepath.Join(t.TempDir(), "out.pptx")
	count, err := SetThemeName(inputPath, outFile, "Office Theme Deck RENAMED", []string{"theme1"})
	if err != nil {
		t.Fatalf("SetThemeName failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	themes, err := ReadThemes(outFile)
	if err != nil {
		t.Fatalf("ReadThemes failed: %v", err)
	}
	if got := themeByFile(t, themes, "theme1.xml").ThemeName; got != "Office Theme Deck RENAMED" {
		t.Fatalf("theme1 name = %q", got)
	}
}

func TestSetThemeName_AppXMLNoMatchFails(t *testing.T) {
	skipIfNoFixture(t)

	inputPath := extractedFixtureToPPTX(t, func(tempDir string) {
		appPath := filepath.Join(tempDir, "docProps", "app.xml")
		content, err := os.ReadFile(appPath)
		if err != nil {
			t.Fatalf("read app.xml failed: %v", err)
		}
		content = bytes.Replace(content, []byte("Office Theme Deck"), []byte("Other Theme Name"), 1)
		if err := os.WriteFile(appPath, content, 0644); err != nil {
			t.Fatalf("write app.xml failed: %v", err)
		}
	})

	outFile := filepath.Join(t.TempDir(), "out.pptx")
	_, err := SetThemeName(inputPath, outFile, "Renamed", []string{"theme1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsAll(err.Error(), "no TitlesOfParts entry matched theme name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetThemeName_AppXMLDuplicateMatchFails(t *testing.T) {
	skipIfNoFixture(t)

	inputPath := extractedFixtureToPPTX(t, func(tempDir string) {
		appPath := filepath.Join(tempDir, "docProps", "app.xml")
		content, err := os.ReadFile(appPath)
		if err != nil {
			t.Fatalf("read app.xml failed: %v", err)
		}
		content = bytes.Replace(content, []byte("Blue II Deck"), []byte("Office Theme Deck"), 1)
		if err := os.WriteFile(appPath, content, 0644); err != nil {
			t.Fatalf("write app.xml failed: %v", err)
		}
	})

	outFile := filepath.Join(t.TempDir(), "out.pptx")
	_, err := SetThemeName(inputPath, outFile, "Renamed", []string{"theme1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsAll(err.Error(), "multiple TitlesOfParts entries matched theme name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetThemeName_EmptyThemeNameFailsEarly(t *testing.T) {
	skipIfNoFixture(t)

	inputPath := extractedFixtureToPPTX(t, func(tempDir string) {
		themePath := filepath.Join(tempDir, "ppt", "theme", "theme1.xml")
		content, err := os.ReadFile(themePath)
		if err != nil {
			t.Fatalf("read theme1.xml failed: %v", err)
		}
		content = bytes.Replace(content, []byte(`name="Office Theme Deck"`), []byte(`name=""`), 1)
		if err := os.WriteFile(themePath, content, 0644); err != nil {
			t.Fatalf("write theme1.xml failed: %v", err)
		}
	})

	outFile := filepath.Join(t.TempDir(), "out.pptx")
	_, err := SetThemeName(inputPath, outFile, "Renamed", []string{"theme1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsAll(err.Error(), "theme1.xml", "empty name attribute") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetThemeName_MissingThemeNameAttributeFailsEarly(t *testing.T) {
	skipIfNoFixture(t)

	inputPath := extractedFixtureToPPTX(t, func(tempDir string) {
		themePath := filepath.Join(tempDir, "ppt", "theme", "theme1.xml")
		content, err := os.ReadFile(themePath)
		if err != nil {
			t.Fatalf("read theme1.xml failed: %v", err)
		}
		content = bytes.Replace(content, []byte(` name="Office Theme Deck"`), []byte(""), 1)
		if err := os.WriteFile(themePath, content, 0644); err != nil {
			t.Fatalf("write theme1.xml failed: %v", err)
		}
	})

	outFile := filepath.Join(t.TempDir(), "out.pptx")
	_, err := SetThemeName(inputPath, outFile, "Renamed", []string{"theme1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !containsAll(err.Error(), "theme1.xml", "no name attribute") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func titlesOfPartsFromPPTX(t *testing.T, pptxPath string) []string {
	t.Helper()

	tempDir := t.TempDir()
	if err := toolkitpptx.ExtractPPTX(pptxPath, tempDir); err != nil {
		t.Fatalf("ExtractPPTX failed: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tempDir, "docProps", "app.xml"))
	if err != nil {
		t.Fatalf("read app.xml failed: %v", err)
	}
	values, err := titlesOfParts(content)
	if err != nil {
		t.Fatalf("titlesOfParts failed: %v", err)
	}
	return values
}

func extractedFixtureToPPTX(t *testing.T, mutate func(string)) string {
	t.Helper()

	tempDir := extractFixtureToTempDir(t)
	mutate(tempDir)
	inputPath := filepath.Join(t.TempDir(), "input.pptx")
	if err := toolkitpptx.RepackPPTX(tempDir, inputPath); err != nil {
		t.Fatalf("RepackPPTX failed: %v", err)
	}
	return inputPath
}

func replaceOneString(t *testing.T, values []string, oldValue, newValue string) {
	t.Helper()
	for i, value := range values {
		if value == oldValue {
			values[i] = newValue
			return
		}
	}
	t.Fatalf("value %q not found", oldValue)
}

func assertStringSlicesEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(got)=%d len(want)=%d\n got=%q\nwant=%q", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("value[%d]=%q want %q\n got=%q\nwant=%q", i, got[i], want[i], got, want)
		}
	}
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !bytes.Contains([]byte(s), []byte(part)) {
			return false
		}
	}
	return true
}
