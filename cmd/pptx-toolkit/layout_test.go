package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testPPTX = "testdata/test.pptx"

func skipIfNoFixture(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(testPPTX); os.IsNotExist(err) {
		t.Skip("test.pptx fixture not found")
	}
}

// --- parseLayoutXML ---

func TestParseLayoutXML(t *testing.T) {
	tests := []struct {
		name             string
		xml              string
		wantName         string
		wantMatchingName string
	}{
		{
			name: "both fields present",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
             xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
             matchingName="Layout with matchName property">
  <p:cSld name="matchName-test"></p:cSld>
</p:sldLayout>`,
			wantName:         "matchName-test",
			wantMatchingName: "Layout with matchName property",
		},
		{
			name: "only cSld name present",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
             xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
             preserve="1" userDrawn="1">
  <p:cSld name="CODEX-TEST"></p:cSld>
</p:sldLayout>`,
			wantName:         "CODEX-TEST",
			wantMatchingName: "",
		},
		{
			name: "empty name attributes",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld name=""></p:cSld>
</p:sldLayout>`,
			wantName:         "",
			wantMatchingName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.CreateTemp("", "layout-*.xml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(f.Name())
			f.WriteString(tt.xml)
			f.Close()

			gotName, gotMatching, err := parseLayoutXML(f.Name())
			if err != nil {
				t.Fatalf("parseLayoutXML returned error: %v", err)
			}
			if gotName != tt.wantName {
				t.Errorf("name: got %q, want %q", gotName, tt.wantName)
			}
			if gotMatching != tt.wantMatchingName {
				t.Errorf("matchingName: got %q, want %q", gotMatching, tt.wantMatchingName)
			}
		})
	}
}

// --- layoutNumber ---

func TestLayoutNumber(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"slideLayout1.xml", 1, false},
		{"slideLayout12.xml", 12, false},
		{"slideLayout100.xml", 100, false},
		{"notALayout.xml", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := layoutNumber(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error: got %v, wantErr=%v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

// --- normalizeThemeFilter ---

func TestNormalizeThemeFilter(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		checks []string // keys expected to be true
	}{
		{
			name:   "without extension",
			input:  []string{"theme1"},
			checks: []string{"theme1.xml"},
		},
		{
			name:   "with extension",
			input:  []string{"theme1.xml"},
			checks: []string{"theme1.xml"},
		},
		{
			name:   "empty",
			input:  nil,
			checks: nil,
		},
		{
			name:   "trims whitespace and ignores empty entries",
			input:  []string{" theme1 ", "", " theme2.xml ", "   "},
			checks: []string{"theme1.xml", "theme2.xml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeThemeFilter(tt.input)
			for _, k := range tt.checks {
				if !got[k] {
					t.Errorf("expected key %q to be present", k)
				}
			}

			for k := range got {
				if k == ".xml" || k == " theme1 .xml" || k == " theme2.xml " {
					t.Errorf("unexpected unnormalized key %q", k)
				}
			}
		})
	}
}

// --- ReadLayouts (fixture-backed integration tests) ---

func TestReadLayouts_AllLayouts(t *testing.T) {
	skipIfNoFixture(t)

	layouts, err := ReadLayouts(testPPTX, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts failed: %v", err)
	}

	if len(layouts) == 0 {
		t.Fatal("expected at least one layout, got none")
	}

	// Layouts must be in ascending numeric order
	for i := 1; i < len(layouts); i++ {
		prev, _ := layoutNumber(layouts[i-1].FileName)
		curr, _ := layoutNumber(layouts[i].FileName)
		if curr <= prev {
			t.Errorf("layouts not sorted: %s before %s", layouts[i-1].FileName, layouts[i].FileName)
		}
	}
}

func TestReadLayouts_DivergentFieldsLayout(t *testing.T) {
	skipIfNoFixture(t)

	// slideLayout12 is the dedicated test layout with divergent fields
	layouts, err := ReadLayouts(testPPTX, LayoutFilters{LayoutID: "slideLayout12"})
	if err != nil {
		t.Fatalf("ReadLayouts failed: %v", err)
	}

	if len(layouts) != 1 {
		t.Fatalf("expected 1 layout, got %d", len(layouts))
	}

	l := layouts[0]

	if l.Name != "matchName-test" {
		t.Errorf("Name: got %q, want %q", l.Name, "matchName-test")
	}
	if l.MatchingName != "Layout with matchName property" {
		t.Errorf("MatchingName: got %q, want %q", l.MatchingName, "Layout with matchName property")
	}
	if l.LayoutID != "slideLayout12" {
		t.Errorf("LayoutID: got %q, want %q", l.LayoutID, "slideLayout12")
	}
	if l.Master == "" {
		t.Error("expected Master to be populated")
	}
	if l.Theme == "" {
		t.Error("expected Theme to be populated")
	}
}

func TestReadLayouts_NoMatchingNameReportsEmpty(t *testing.T) {
	skipIfNoFixture(t)

	layouts, err := ReadLayouts(testPPTX, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts failed: %v", err)
	}

	// At least some layouts should have no matching name (PowerPoint-native layouts)
	found := false
	for _, l := range layouts {
		if l.MatchingName == "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one layout with empty MatchingName")
	}
}

func TestReadLayouts_FilterByLayoutID(t *testing.T) {
	skipIfNoFixture(t)

	layouts, err := ReadLayouts(testPPTX, LayoutFilters{LayoutID: "slideLayout1"})
	if err != nil {
		t.Fatalf("ReadLayouts failed: %v", err)
	}

	if len(layouts) != 1 {
		t.Fatalf("expected 1 layout, got %d", len(layouts))
	}
	if layouts[0].LayoutID != "slideLayout1" {
		t.Errorf("got LayoutID %q, want %q", layouts[0].LayoutID, "slideLayout1")
	}
}

func TestReadLayouts_FilterByName(t *testing.T) {
	skipIfNoFixture(t)

	layouts, err := ReadLayouts(testPPTX, LayoutFilters{Name: "matchName-test"})
	if err != nil {
		t.Fatalf("ReadLayouts failed: %v", err)
	}

	if len(layouts) != 1 {
		t.Fatalf("expected 1 layout, got %d", len(layouts))
	}
	if layouts[0].Name != "matchName-test" {
		t.Errorf("got Name %q, want %q", layouts[0].Name, "matchName-test")
	}
}

func TestReadLayouts_FilterByMatchingName(t *testing.T) {
	skipIfNoFixture(t)

	layouts, err := ReadLayouts(testPPTX, LayoutFilters{MatchingName: "Layout with matchName property"})
	if err != nil {
		t.Fatalf("ReadLayouts failed: %v", err)
	}

	if len(layouts) != 1 {
		t.Fatalf("expected 1 layout, got %d", len(layouts))
	}
	if layouts[0].MatchingName != "Layout with matchName property" {
		t.Errorf("got MatchingName %q, want %q", layouts[0].MatchingName, "Layout with matchName property")
	}
}

func TestReadLayouts_FilterByTheme(t *testing.T) {
	skipIfNoFixture(t)

	// Without extension
	layouts, err := ReadLayouts(testPPTX, LayoutFilters{Theme: []string{"theme1"}})
	if err != nil {
		t.Fatalf("ReadLayouts failed: %v", err)
	}
	if len(layouts) == 0 {
		t.Fatal("expected layouts for theme1, got none")
	}
	for _, l := range layouts {
		if l.Theme != "theme1.xml" {
			t.Errorf("layout %s: Theme=%q, expected theme1.xml", l.LayoutID, l.Theme)
		}
	}
}

func TestReadLayouts_FilterNoMatch(t *testing.T) {
	skipIfNoFixture(t)

	layouts, err := ReadLayouts(testPPTX, LayoutFilters{LayoutID: "slideLayout9999"})
	if err != nil {
		t.Fatalf("ReadLayouts failed: %v", err)
	}
	if len(layouts) != 0 {
		t.Errorf("expected 0 layouts, got %d", len(layouts))
	}
}

func TestReadLayouts_InvalidFile(t *testing.T) {
	_, err := ReadLayouts("nonexistent.pptx", LayoutFilters{})
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestReadLayouts_UsedBySlides(t *testing.T) {
	skipIfNoFixture(t)

	layouts, err := ReadLayouts(testPPTX, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts failed: %v", err)
	}

	// At least one layout should be used by at least one slide
	hasUsedLayout := false
	for _, l := range layouts {
		if len(l.UsedBySlides) > 0 {
			hasUsedLayout = true
			// Slide numbers must be sorted
			for i := 1; i < len(l.UsedBySlides); i++ {
				if l.UsedBySlides[i] <= l.UsedBySlides[i-1] {
					t.Errorf("layout %s: slide numbers not sorted: %v", l.LayoutID, l.UsedBySlides)
				}
			}
		}
	}
	if !hasUsedLayout {
		t.Error("expected at least one layout with UsedBySlides populated")
	}
}

// --- buildSlideToLayoutMapping ---

func TestBuildSlideToLayoutMapping(t *testing.T) {
	skipIfNoFixture(t)

	tempDir := t.TempDir()
	if err := extractPPTX(testPPTX, tempDir); err != nil {
		t.Fatalf("extractPPTX failed: %v", err)
	}

	mapping, err := buildSlideToLayoutMapping(tempDir)
	if err != nil {
		t.Fatalf("buildSlideToLayoutMapping returned error: %v", err)
	}

	// Must be non-empty
	if len(mapping) == 0 {
		t.Fatal("expected non-empty slide→layout mapping")
	}

	// All values must be sorted
	for layoutFile, slides := range mapping {
		for i := 1; i < len(slides); i++ {
			if slides[i] <= slides[i-1] {
				t.Errorf("%s: slide numbers not sorted: %v", layoutFile, slides)
			}
		}
	}
}

// --- extractPPTX ---

func TestExtractPPTX(t *testing.T) {
	skipIfNoFixture(t)

	destDir := t.TempDir()
	if err := extractPPTX(testPPTX, destDir); err != nil {
		t.Fatalf("extractPPTX failed: %v", err)
	}

	// presentation.xml must exist after extraction
	presentationPath := filepath.Join(destDir, "ppt", "presentation.xml")
	if _, err := os.Stat(presentationPath); os.IsNotExist(err) {
		t.Error("presentation.xml not found after extraction")
	}

	// At least one layout file must exist
	layouts, err := filepath.Glob(filepath.Join(destDir, "ppt", "slideLayouts", "slideLayout*.xml"))
	if err != nil || len(layouts) == 0 {
		t.Error("no layout files found after extraction")
	}
}

func TestExtractPPTX_InvalidFile(t *testing.T) {
	destDir := t.TempDir()
	err := extractPPTX("nonexistent.pptx", destDir)
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLayoutListCommand_DoesNotLeakFiltersBetweenExecutions(t *testing.T) {
	skipIfNoFixture(t)

	origLayoutIDFilter := layoutIDFilter
	origLayoutNameFilter := layoutNameFilter
	origLayoutMatchFilter := layoutMatchFilter
	origLayoutThemeFilter := append([]string(nil), layoutThemeFilter...)
	defer func() {
		layoutIDFilter = origLayoutIDFilter
		layoutNameFilter = origLayoutNameFilter
		layoutMatchFilter = origLayoutMatchFilter
		layoutThemeFilter = origLayoutThemeFilter
		_ = layoutListCmd.Flags().Set("layout-id", origLayoutIDFilter)
		_ = layoutListCmd.Flags().Set("name", origLayoutNameFilter)
		_ = layoutListCmd.Flags().Set("matching-name", origLayoutMatchFilter)
		_ = layoutListCmd.Flags().Set("theme", strings.Join(origLayoutThemeFilter, ","))
	}()

	var firstOut bytes.Buffer
	var firstErr bytes.Buffer
	rootCmd.SetOut(&firstOut)
	rootCmd.SetErr(&firstErr)
	rootCmd.SetArgs([]string{"layout", "list", "--layout-id", "slideLayout12", testPPTX})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first execution failed: %v; stderr=%s", err, firstErr.String())
	}

	if !strings.Contains(firstOut.String(), "slideLayout12.xml") {
		t.Fatalf("expected filtered output to include slideLayout12.xml, got:\n%s", firstOut.String())
	}

	var secondOut bytes.Buffer
	var secondErr bytes.Buffer
	rootCmd.SetOut(&secondOut)
	rootCmd.SetErr(&secondErr)
	rootCmd.SetArgs([]string{"layout", "list", testPPTX})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second execution failed: %v; stderr=%s", err, secondErr.String())
	}

	if strings.Contains(secondOut.String(), "Found 1 layout(s)") {
		t.Fatalf("expected unfiltered output on second execution, got:\n%s", secondOut.String())
	}
}
