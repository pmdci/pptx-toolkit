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

func TestRemoveMatchingNameAttr(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantAbsent  string
		wantPresent string
	}{
		{
			name: "removes double-quoted attribute",
			input: `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" matchingName="Foo" preserve="1">` + "\n" +
				`  <p:cSld name="Bar"></p:cSld>` + "\n" +
				`</p:sldLayout>`,
			wantAbsent:  `matchingName=`,
			wantPresent: `preserve="1"`,
		},
		{
			name: "removes single-quoted attribute",
			input: `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" matchingName='Foo' preserve="1">` + "\n" +
				`  <p:cSld name="Bar"></p:cSld>` + "\n" +
				`</p:sldLayout>`,
			wantAbsent:  `matchingName=`,
			wantPresent: `preserve="1"`,
		},
		{
			name: "no-op when attribute absent",
			input: `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" preserve="1">` + "\n" +
				`  <p:cSld name="Bar"></p:cSld>` + "\n" +
				`</p:sldLayout>`,
			wantAbsent:  `matchingName=`,
			wantPresent: `preserve="1"`,
		},
		{
			name: "xml declaration does not fool the scanner",
			input: `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" matchingName="A&gt;B" preserve="1">` + "\n" +
				`  <p:cSld name="Bar"></p:cSld>` + "\n" +
				`</p:sldLayout>`,
			wantAbsent:  `matchingName=`,
			wantPresent: `preserve="1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := removeMatchingNameAttr([]byte(tt.input))
			if err != nil {
				t.Fatalf("removeMatchingNameAttr returned error: %v", err)
			}
			if tt.wantAbsent != "" && bytes.Contains(got, []byte(tt.wantAbsent)) {
				t.Errorf("expected %q to be absent, got:\n%s", tt.wantAbsent, got)
			}
			if tt.wantPresent != "" && !bytes.Contains(got, []byte(tt.wantPresent)) {
				t.Errorf("expected %q to be present, got:\n%s", tt.wantPresent, got)
			}
		})
	}
}

func TestSetLayoutProperty(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		target      string
		value       string
		wantContain string
	}{
		{
			name: "sets matching-name by replacing existing attribute",
			input: `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" matchingName="Old value" preserve="1">` + "\n" +
				`  <p:cSld name="Bar"></p:cSld>` + "\n" +
				`</p:sldLayout>`,
			target:      layoutPropertyMatchingName,
			value:       `New & Better`,
			wantContain: `matchingName="New &amp; Better"`,
		},
		{
			name: "sets matching-name by creating absent attribute",
			input: `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" preserve="1">` + "\n" +
				`  <p:cSld name="Bar"></p:cSld>` + "\n" +
				`</p:sldLayout>`,
			target:      layoutPropertyMatchingName,
			value:       `Created`,
			wantContain: `matchingName="Created"`,
		},
		{
			name: "sets cSld name by replacing existing attribute",
			input: `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" preserve="1">` + "\n" +
				`  <p:cSld name="Old"></p:cSld>` + "\n" +
				`</p:sldLayout>`,
			target:      layoutPropertyName,
			value:       `Fresh`,
			wantContain: `<p:cSld name="Fresh">`,
		},
		{
			name: "value with dollar sign is not corrupted",
			input: `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" matchingName="Old" preserve="1">` + "\n" +
				`  <p:cSld name="Bar"></p:cSld>` + "\n" +
				`</p:sldLayout>`,
			target:      layoutPropertyMatchingName,
			value:       `Price $100`,
			wantContain: `matchingName="Price $100"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := setLayoutProperty([]byte(tt.input), tt.target, tt.value)
			if err != nil {
				t.Fatalf("setLayoutProperty returned error: %v", err)
			}
			if !bytes.Contains(got, []byte(tt.wantContain)) {
				t.Fatalf("expected output to contain %q, got:\n%s", tt.wantContain, got)
			}
		})
	}
}

func TestSetLayoutProperty_CopiedMissingSourceBecomesNoOp(t *testing.T) {
	skipIfNoFixture(t)

	outFile := filepath.Join(t.TempDir(), "out.pptx")
	mapping := &LayoutSetMapping{
		SourceKind:     LayoutSetSourceProperty,
		SourceProperty: layoutPropertyMatchingName,
		TargetProperty: layoutPropertyName,
	}

	count, err := SetLayoutProperty(testPPTX, outFile, mapping, LayoutFilters{})
	if err != nil {
		t.Fatalf("SetLayoutProperty failed: %v", err)
	}
	if count == 0 {
		t.Skip("test.pptx has no layouts with matchingName")
	}

	originalLayouts, err := ReadLayouts(testPPTX, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts on input failed: %v", err)
	}
	resultLayouts, err := ReadLayouts(outFile, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts on output failed: %v", err)
	}

	originalByID := make(map[string]*LayoutInfo)
	for _, l := range originalLayouts {
		originalByID[l.LayoutID] = l
	}

	for _, l := range resultLayouts {
		orig := originalByID[l.LayoutID]
		if orig == nil {
			t.Fatalf("missing original layout for %s", l.LayoutID)
		}
		if orig.MatchingName == "" && l.Name != orig.Name {
			t.Fatalf("layout %s had no matching-name but name changed from %q to %q", l.LayoutID, orig.Name, l.Name)
		}
	}
}

func TestRemoveLayoutMatchingName(t *testing.T) {
	skipIfNoFixture(t)

	outFile := filepath.Join(t.TempDir(), "out.pptx")
	count, err := RemoveLayoutMatchingName(testPPTX, outFile, LayoutFilters{})
	if err != nil {
		t.Fatalf("RemoveLayoutMatchingName failed: %v", err)
	}
	if count == 0 {
		t.Skip("test.pptx has no layouts with matchingName — skipping removal check")
	}

	layouts, err := ReadLayouts(outFile, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts on output failed: %v", err)
	}
	for _, l := range layouts {
		if l.MatchingName != "" {
			t.Errorf("layout %s still has matchingName=%q after removal", l.FileName, l.MatchingName)
		}
	}
}

func TestRemoveLayoutMatchingName_FilteredRemoval(t *testing.T) {
	skipIfNoFixture(t)

	// Determine which layouts have matchingName set
	all, err := ReadLayouts(testPPTX, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts failed: %v", err)
	}
	var candidates []*LayoutInfo
	for _, l := range all {
		if l.MatchingName != "" {
			candidates = append(candidates, l)
		}
	}
	if len(candidates) == 0 {
		t.Skip("test.pptx has no layouts with matchingName")
	}

	target := candidates[0]
	outFile := filepath.Join(t.TempDir(), "out.pptx")
	count, err := RemoveLayoutMatchingName(testPPTX, outFile, LayoutFilters{LayoutID: target.LayoutID})
	if err != nil {
		t.Fatalf("RemoveLayoutMatchingName failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 layout modified, got %d", count)
	}

	result, err := ReadLayouts(outFile, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts on output failed: %v", err)
	}
	for _, l := range result {
		if l.LayoutID == target.LayoutID && l.MatchingName != "" {
			t.Errorf("target layout %s still has matchingName=%q", l.LayoutID, l.MatchingName)
		}
		if l.LayoutID != target.LayoutID && l.MatchingName == "" {
			// Only flag if the original had a matchingName (i.e., it was cleared unexpectedly)
			for _, orig := range all {
				if orig.LayoutID == l.LayoutID && orig.MatchingName != "" {
					t.Errorf("non-target layout %s had matchingName cleared unexpectedly", l.LayoutID)
				}
			}
		}
	}
}

func TestSetLayoutProperty_LiteralMatchingNameAllLayouts(t *testing.T) {
	skipIfNoFixture(t)

	outFile := filepath.Join(t.TempDir(), "out.pptx")
	mapping := &LayoutSetMapping{
		SourceKind:     LayoutSetSourceLiteral,
		SourceLiteral:  "Layout with matchName property",
		TargetProperty: layoutPropertyMatchingName,
	}

	count, err := SetLayoutProperty(testPPTX, outFile, mapping, LayoutFilters{})
	if err != nil {
		t.Fatalf("SetLayoutProperty failed: %v", err)
	}
	if count == 0 {
		t.Fatal("expected at least one layout to be updated")
	}

	layouts, err := ReadLayouts(outFile, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts on output failed: %v", err)
	}
	for _, l := range layouts {
		if l.MatchingName != "Layout with matchName property" {
			t.Fatalf("layout %s has MatchingName=%q, want %q", l.LayoutID, l.MatchingName, "Layout with matchName property")
		}
	}
}

func TestSetLayoutProperty_CopyNameToMatchingName(t *testing.T) {
	skipIfNoFixture(t)

	outFile := filepath.Join(t.TempDir(), "out.pptx")
	mapping := &LayoutSetMapping{
		SourceKind:     LayoutSetSourceProperty,
		SourceProperty: layoutPropertyName,
		TargetProperty: layoutPropertyMatchingName,
	}

	count, err := SetLayoutProperty(testPPTX, outFile, mapping, LayoutFilters{})
	if err != nil {
		t.Fatalf("SetLayoutProperty failed: %v", err)
	}
	if count == 0 {
		t.Fatal("expected at least one layout to be updated")
	}

	layouts, err := ReadLayouts(outFile, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts on output failed: %v", err)
	}
	for _, l := range layouts {
		if l.MatchingName != l.Name {
			t.Fatalf("layout %s has Name=%q MatchingName=%q, expected them to match", l.LayoutID, l.Name, l.MatchingName)
		}
	}
}

func TestSetLayoutProperty_FilteredByLayoutID(t *testing.T) {
	skipIfNoFixture(t)

	originalLayouts, err := ReadLayouts(testPPTX, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts on input failed: %v", err)
	}
	originalByID := make(map[string]*LayoutInfo)
	for _, l := range originalLayouts {
		originalByID[l.LayoutID] = l
	}

	targetID := "slideLayout12"
	outFile := filepath.Join(t.TempDir(), "out.pptx")
	mapping := &LayoutSetMapping{
		SourceKind:     LayoutSetSourceLiteral,
		SourceLiteral:  "Only target changed",
		TargetProperty: layoutPropertyMatchingName,
	}

	count, err := SetLayoutProperty(testPPTX, outFile, mapping, LayoutFilters{LayoutID: targetID})
	if err != nil {
		t.Fatalf("SetLayoutProperty failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 layout updated, got %d", count)
	}

	layouts, err := ReadLayouts(outFile, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts on output failed: %v", err)
	}
	for _, l := range layouts {
		if l.LayoutID == targetID {
			if l.MatchingName != "Only target changed" {
				t.Fatalf("target layout %s has MatchingName=%q", l.LayoutID, l.MatchingName)
			}
			continue
		}
		orig := originalByID[l.LayoutID]
		if orig == nil {
			t.Fatalf("missing original layout for %s", l.LayoutID)
		}
		if l.MatchingName != orig.MatchingName {
			t.Fatalf("non-target layout %s changed from %q to %q", l.LayoutID, orig.MatchingName, l.MatchingName)
		}
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

func TestLayoutRemoveCommand_DoesNotLeakFiltersBetweenExecutions(t *testing.T) {
	skipIfNoFixture(t)

	outDir := t.TempDir()
	firstOutput := filepath.Join(outDir, "filtered-out.pptx")
	secondOutput := filepath.Join(outDir, "unfiltered-out.pptx")

	originalLayouts, err := ReadLayouts(testPPTX, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts on fixture failed: %v", err)
	}
	originalWithMatchingName := 0
	for _, l := range originalLayouts {
		if l.MatchingName != "" {
			originalWithMatchingName++
		}
	}
	if originalWithMatchingName == 0 {
		t.Skip("test.pptx has no layouts with matchingName")
	}

	var firstOut bytes.Buffer
	var firstErr bytes.Buffer
	rootCmd.SetOut(&firstOut)
	rootCmd.SetErr(&firstErr)
	rootCmd.SetArgs([]string{"layout", "remove", "matching-name", "--layout-id", "slideLayout12", testPPTX, firstOutput})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first remove execution failed: %v; stderr=%s", err, firstErr.String())
	}

	firstLayouts, err := ReadLayouts(firstOutput, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts on first output failed: %v", err)
	}

	firstRemaining := 0
	for _, l := range firstLayouts {
		if l.MatchingName != "" {
			firstRemaining++
		}
	}
	if firstRemaining != originalWithMatchingName-1 {
		t.Fatalf("expected first execution to remove exactly one matching-name; before=%d after=%d", originalWithMatchingName, firstRemaining)
	}

	var secondOut bytes.Buffer
	var secondErr bytes.Buffer
	rootCmd.SetOut(&secondOut)
	rootCmd.SetErr(&secondErr)
	rootCmd.SetArgs([]string{"layout", "remove", "matching-name", testPPTX, secondOutput})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second remove execution failed: %v; stderr=%s", err, secondErr.String())
	}

	secondLayouts, err := ReadLayouts(secondOutput, LayoutFilters{})
	if err != nil {
		t.Fatalf("ReadLayouts on second output failed: %v", err)
	}
	for _, l := range secondLayouts {
		if l.MatchingName != "" {
			t.Fatalf("expected second execution to be unfiltered; layout %s still has matchingName=%q", l.LayoutID, l.MatchingName)
		}
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
