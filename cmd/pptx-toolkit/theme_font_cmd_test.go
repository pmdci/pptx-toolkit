package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestThemeFontSetCommand(t *testing.T) {
	skipIfNoFixture(t)

	original, err := ReadFontSchemes(testPPTX, nil)
	if err != nil {
		t.Fatalf("ReadFontSchemes on fixture failed: %v", err)
	}
	originalByFile := make(map[string]*FontScheme, len(original))
	for _, scheme := range original {
		originalByFile[scheme.FileName] = scheme
	}

	tests := []struct {
		name         string
		args         []string
		wantErr      string
		assertOutput func(t *testing.T, outFile string)
	}{
		{
			name: "major only",
			args: []string{"theme", "font", "set", "--major", "Arial", testPPTX},
			assertOutput: func(t *testing.T, outFile string) {
				assertFontSchemes(t, outFile, func(s *FontScheme) {
					if s.MajorTypeface != "Arial" {
						t.Fatalf("%s: major = %q, want %q", s.FileName, s.MajorTypeface, "Arial")
					}
					if s.MinorTypeface != originalByFile[s.FileName].MinorTypeface {
						t.Fatalf("%s: minor changed from %q to %q", s.FileName, originalByFile[s.FileName].MinorTypeface, s.MinorTypeface)
					}
				})
			},
		},
		{
			name: "minor only",
			args: []string{"theme", "font", "set", "--minor", "Times New Roman", testPPTX},
			assertOutput: func(t *testing.T, outFile string) {
				assertFontSchemes(t, outFile, func(s *FontScheme) {
					if s.MinorTypeface != "Times New Roman" {
						t.Fatalf("%s: minor = %q, want %q", s.FileName, s.MinorTypeface, "Times New Roman")
					}
					if s.MajorTypeface != originalByFile[s.FileName].MajorTypeface {
						t.Fatalf("%s: major changed from %q to %q", s.FileName, originalByFile[s.FileName].MajorTypeface, s.MajorTypeface)
					}
				})
			},
		},
		{
			name: "both",
			args: []string{"theme", "font", "set", "--major", "Arial", "--minor", "Times New Roman", testPPTX},
			assertOutput: func(t *testing.T, outFile string) {
				assertFontSchemes(t, outFile, func(s *FontScheme) {
					if s.MajorTypeface != "Arial" || s.MinorTypeface != "Times New Roman" {
						t.Fatalf("%s: got major=%q minor=%q", s.FileName, s.MajorTypeface, s.MinorTypeface)
					}
				})
			},
		},
		{
			name: "scheme name only",
			args: []string{"theme", "font", "set", "--scheme-name", "Corporate", testPPTX},
			assertOutput: func(t *testing.T, outFile string) {
				assertFontSchemes(t, outFile, func(s *FontScheme) {
					if s.SchemeName != "Corporate" {
						t.Fatalf("%s: scheme name = %q, want %q", s.FileName, s.SchemeName, "Corporate")
					}
					if s.MajorTypeface != originalByFile[s.FileName].MajorTypeface || s.MinorTypeface != originalByFile[s.FileName].MinorTypeface {
						t.Fatalf("%s: typefaces changed unexpectedly to major=%q minor=%q", s.FileName, s.MajorTypeface, s.MinorTypeface)
					}
				})
			},
		},
		{
			name: "theme filter",
			args: []string{"theme", "font", "set", "--major", "Arial", "--theme", "theme1", testPPTX},
			assertOutput: func(t *testing.T, outFile string) {
				assertFontSchemes(t, outFile, func(s *FontScheme) {
					if s.FileName == "theme1.xml" {
						if s.MajorTypeface != "Arial" {
							t.Fatalf("%s: major = %q, want %q", s.FileName, s.MajorTypeface, "Arial")
						}
						return
					}
					if s.MajorTypeface != originalByFile[s.FileName].MajorTypeface {
						t.Fatalf("%s: major changed from %q to %q", s.FileName, originalByFile[s.FileName].MajorTypeface, s.MajorTypeface)
					}
				})
			},
		},
		{
			name:    "no flags",
			args:    []string{"theme", "font", "set", testPPTX},
			wantErr: "at least one of",
		},
		{
			name:    "unknown theme filter",
			args:    []string{"theme", "font", "set", "--major", "Arial", "--theme", "NoSuchTheme", testPPTX},
			wantErr: "no themes matched",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "out.pptx")

			var out bytes.Buffer
			var errBuf bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&errBuf)

			args := append([]string(nil), tt.args...)
			args = append(args, outFile)
			rootCmd.SetArgs(args)

			err := rootCmd.Execute()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q", tt.wantErr)
				}
				combined := out.String() + errBuf.String()
				if !strings.Contains(combined, tt.wantErr) {
					t.Fatalf("output = %q, want substring %q", combined, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("command failed: %v; stderr=%s", err, errBuf.String())
			}

			tt.assertOutput(t, outFile)
		})
	}
}

func TestThemeFontListCommand_ThemeFilter(t *testing.T) {
	skipIfNoFixture(t)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"theme", "font", "list", "--theme", "theme1", testPPTX})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list command failed: %v; stderr=%s", err, errBuf.String())
	}

	output := out.String()
	if !strings.Contains(output, "━━━ theme1.xml ━━━") {
		t.Fatalf("expected theme1 block in output, got:\n%s", output)
	}
	if strings.Contains(output, "theme2.xml") {
		t.Fatalf("expected filtered output to omit theme2.xml, got:\n%s", output)
	}
	if !strings.Contains(output, "major  (headings):  Aptos Display") {
		t.Fatalf("expected font output, got:\n%s", output)
	}
}

func TestThemeFontListSetList_RoundTrip(t *testing.T) {
	skipIfNoFixture(t)

	outFile := filepath.Join(t.TempDir(), "roundtrip.pptx")

	var firstOut bytes.Buffer
	var firstErr bytes.Buffer
	rootCmd.SetOut(&firstOut)
	rootCmd.SetErr(&firstErr)
	rootCmd.SetArgs([]string{"theme", "font", "list", testPPTX})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("initial list failed: %v; stderr=%s", err, firstErr.String())
	}
	if !strings.Contains(firstOut.String(), "major  (headings):  Aptos Display") {
		t.Fatalf("expected initial list output to show original major font, got:\n%s", firstOut.String())
	}

	var setOut bytes.Buffer
	var setErr bytes.Buffer
	rootCmd.SetOut(&setOut)
	rootCmd.SetErr(&setErr)
	rootCmd.SetArgs([]string{"theme", "font", "set", "--major", "Arial", "--minor", "Times New Roman", testPPTX, outFile})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("set command failed: %v; stderr=%s", err, setErr.String())
	}

	var secondOut bytes.Buffer
	var secondErr bytes.Buffer
	rootCmd.SetOut(&secondOut)
	rootCmd.SetErr(&secondErr)
	rootCmd.SetArgs([]string{"theme", "font", "list", outFile})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second list failed: %v; stderr=%s", err, secondErr.String())
	}

	output := secondOut.String()
	if !strings.Contains(output, "major  (headings):  Arial") {
		t.Fatalf("expected updated major font in output, got:\n%s", output)
	}
	if !strings.Contains(output, "minor  (body):      Times New Roman") {
		t.Fatalf("expected updated minor font in output, got:\n%s", output)
	}
}

func assertFontSchemes(t *testing.T, pptxPath string, assert func(*FontScheme)) {
	t.Helper()

	schemes, err := ReadFontSchemes(pptxPath, nil)
	if err != nil {
		t.Fatalf("ReadFontSchemes failed: %v", err)
	}
	for _, scheme := range schemes {
		assert(scheme)
	}
}
