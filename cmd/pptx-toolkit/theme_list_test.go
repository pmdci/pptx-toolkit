package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	toolkitpptx "github.com/pmdci/pptx-toolkit/internal/pptx"
)

func TestReadThemeSummaries(t *testing.T) {
	skipIfNoFixture(t)

	summaries, err := ReadThemeSummaries(testPPTX, nil)
	if err != nil {
		t.Fatalf("ReadThemeSummaries failed: %v", err)
	}

	if len(summaries) != 5 {
		t.Fatalf("expected 5 theme summaries, got %d", len(summaries))
	}

	byFile := make(map[string]*ThemeSummary, len(summaries))
	for _, summary := range summaries {
		byFile[summary.Theme.FileName] = summary
		if summary.Theme.FontSchemeName == "" {
			t.Fatalf("%s: missing font scheme name", summary.Theme.FileName)
		}
	}

	assertBindings(t, byFile["theme1.xml"], masterTypeSlide, []string{"slideMaster1.xml"})
	assertBindings(t, byFile["theme2.xml"], masterTypeSlide, []string{"slideMaster2.xml"})
	assertBindings(t, byFile["theme3.xml"], masterTypeSlide, []string{"slideMaster3.xml"})
	assertBindings(t, byFile["theme4.xml"], masterTypeNotes, []string{"notesMaster1.xml"})
	assertBindings(t, byFile["theme5.xml"], masterTypeHandout, []string{"handoutMaster1.xml"})
}

func TestReadThemeSummaries_Filter(t *testing.T) {
	skipIfNoFixture(t)

	summaries, err := ReadThemeSummaries(testPPTX, []string{"theme4"})
	if err != nil {
		t.Fatalf("ReadThemeSummaries with filter failed: %v", err)
	}

	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].Theme.FileName != "theme4.xml" {
		t.Fatalf("expected theme4.xml, got %s", summaries[0].Theme.FileName)
	}
	assertBindings(t, summaries[0], masterTypeNotes, []string{"notesMaster1.xml"})
}

func TestThemeListCommand(t *testing.T) {
	skipIfNoFixture(t)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"theme", "list", testPPTX})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("theme list failed: %v; stderr=%s", err, errBuf.String())
	}

	output := out.String()
	for _, want := range []string{
		"━━━ theme1.xml ━━━",
		"Font Scheme:",
		"Slide master:   slideMaster1.xml",
		"Notes master:   notesMaster1.xml",
		"Handout master: handoutMaster1.xml",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in output, got:\n%s", want, output)
		}
	}
}

func TestThemeListCommand_Filter(t *testing.T) {
	skipIfNoFixture(t)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"theme", "list", "--theme", "theme4", testPPTX})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("theme list with filter failed: %v; stderr=%s", err, errBuf.String())
	}

	output := out.String()
	if !strings.Contains(output, "━━━ theme4.xml ━━━") {
		t.Fatalf("expected theme4 block in output, got:\n%s", output)
	}
	if strings.Contains(output, "theme1.xml") {
		t.Fatalf("expected filtered output to omit theme1.xml, got:\n%s", output)
	}
	if !strings.Contains(output, "Notes master:   notesMaster1.xml") {
		t.Fatalf("expected notes binding in output, got:\n%s", output)
	}
}

func TestResolveRelationshipTarget(t *testing.T) {
	tests := []struct {
		name   string
		owner  string
		target string
		want   string
		wantOK bool
	}{
		{
			name:   "relative target",
			owner:  "ppt/slideMasters/slideMaster1.xml",
			target: "../theme/theme1.xml",
			want:   "ppt/theme/theme1.xml",
			wantOK: true,
		},
		{
			name:   "package absolute target",
			owner:  "ppt/slideMasters/slideMaster1.xml",
			target: "/ppt/theme/theme1.xml",
			want:   "ppt/theme/theme1.xml",
			wantOK: true,
		},
		{
			name:   "path traversal rejected",
			owner:  "ppt/slideMasters/slideMaster1.xml",
			target: "../../../../etc/passwd",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveRelationshipTarget(tt.owner, tt.target)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOwnerPartPathFromRelsPath_UsesLastPptSegment(t *testing.T) {
	filePath := filepath.ToSlash(filepath.Join("/tmp", "ppt-cache", "nested", "ppt", "workspace", "ppt", "slideMasters", "_rels", "slideMaster1.xml.rels"))
	got := ownerPartPathFromRelsPath(filePath)
	want := "ppt/slideMasters/slideMaster1.xml"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestJoinUnderBaseRejectsEscapes(t *testing.T) {
	base := filepath.Join(string(filepath.Separator), "tmp", "pptx-toolkit")

	if _, ok := joinUnderBase(base, "ppt/slideMasters/_rels/slideMaster1.xml.rels"); !ok {
		t.Fatal("expected in-base path to be accepted")
	}

	if _, ok := joinUnderBase(base, "../../etc/passwd"); ok {
		t.Fatal("expected escaping path to be rejected")
	}
}

func TestGroupedBindingsIncludesUnknown(t *testing.T) {
	slides, notes, handouts, unknown := groupedBindings([]MasterBinding{
		{MasterType: masterTypeSlide, FileName: "slideMaster1.xml"},
		{MasterType: "custom", FileName: "mystery.xml"},
	})

	if len(slides) != 1 || slides[0] != "slideMaster1.xml" {
		t.Fatalf("unexpected slide bindings: %v", slides)
	}
	if len(notes) != 0 || len(handouts) != 0 {
		t.Fatalf("unexpected known binding groups: notes=%v handouts=%v", notes, handouts)
	}
	if len(unknown) != 1 || unknown[0] != "mystery.xml" {
		t.Fatalf("unexpected unknown bindings: %v", unknown)
	}
}

func TestReadThemeSummariesFromDir_MissingMasterRelsIsError(t *testing.T) {
	skipIfNoFixture(t)

	tempDir := extractFixtureToTempDir(t)
	target := filepath.Join(tempDir, "ppt", "slideMasters", "_rels", "slideMaster1.xml.rels")
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove %s: %v", target, err)
	}

	_, err := readThemeSummariesFromDir(tempDir, nil)
	if err == nil {
		t.Fatal("expected error for missing master rels")
	}
	if !strings.Contains(err.Error(), "slideMaster1.xml") {
		t.Fatalf("expected master context in error, got %v", err)
	}
}

func TestReadThemeSummariesFromDir_CorruptMasterRelsIsError(t *testing.T) {
	skipIfNoFixture(t)

	tempDir := extractFixtureToTempDir(t)
	target := filepath.Join(tempDir, "ppt", "slideMasters", "_rels", "slideMaster1.xml.rels")
	if err := os.WriteFile(target, []byte("<Relationships"), 0644); err != nil {
		t.Fatalf("write %s: %v", target, err)
	}

	_, err := readThemeSummariesFromDir(tempDir, nil)
	if err == nil {
		t.Fatal("expected error for corrupt master rels")
	}
	if !strings.Contains(err.Error(), "slideMaster1.xml") {
		t.Fatalf("expected master context in error, got %v", err)
	}
}

func TestReadThemeSummariesFromDir_CorruptPresentationRelsIsError(t *testing.T) {
	skipIfNoFixture(t)

	tempDir := extractFixtureToTempDir(t)
	target := filepath.Join(tempDir, "ppt", "_rels", "presentation.xml.rels")
	if err := os.WriteFile(target, []byte("<Relationships"), 0644); err != nil {
		t.Fatalf("write %s: %v", target, err)
	}

	_, err := readThemeSummariesFromDir(tempDir, nil)
	if err == nil {
		t.Fatal("expected error for corrupt presentation rels")
	}
	if !strings.Contains(err.Error(), "presentation relationships") {
		t.Fatalf("expected presentation relationship context in error, got %v", err)
	}
}

func TestReadThemeSummariesFromDir_ReadableMasterWithoutThemeIsUnbound(t *testing.T) {
	skipIfNoFixture(t)

	tempDir := extractFixtureToTempDir(t)
	target := filepath.Join(tempDir, "ppt", "notesMasters", "_rels", "notesMaster1.xml.rels")
	content := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`
	if err := os.WriteFile(target, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", target, err)
	}

	summaries, err := readThemeSummariesFromDir(tempDir, []string{"theme4"})
	if err != nil {
		t.Fatalf("expected no error for readable but unbound master rels: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if len(summaries[0].Bindings) != 0 {
		t.Fatalf("expected theme4 to be unbound, got %v", summaries[0].Bindings)
	}
}

func extractFixtureToTempDir(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	if err := toolkitpptx.ExtractPPTX(testPPTX, tempDir); err != nil {
		t.Fatalf("ExtractPPTX failed: %v", err)
	}
	return tempDir
}

func assertBindings(t *testing.T, summary *ThemeSummary, masterType string, want []string) {
	t.Helper()

	if summary == nil {
		t.Fatal("summary is nil")
	}

	var got []string
	for _, binding := range summary.Bindings {
		if binding.MasterType == masterType {
			got = append(got, binding.FileName)
		}
	}

	if len(got) != len(want) {
		t.Fatalf("%s: got %v, want %v", summary.Theme.FileName, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: got %v, want %v", summary.Theme.FileName, got, want)
		}
	}
}
