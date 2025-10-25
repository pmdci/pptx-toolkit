package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessPPTX(t *testing.T) {
	// Path to test fixture
	testPPTX := filepath.Join("testdata", "test.pptx")

	// Check if fixture exists
	if _, err := os.Stat(testPPTX); os.IsNotExist(err) {
		t.Skip("test.pptx fixture not found")
	}

	t.Run("process without theme filter", func(t *testing.T) {
		// Create temp output file
		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Process with a simple mapping
		mapping := map[string]string{"accent1": "accent6"}
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, nil, "all", nil)

		if err != nil {
			t.Fatalf("ProcessPPTX failed: %v", err)
		}

		if filesProcessed == 0 {
			t.Error("expected to process some files, got 0")
		}

		// Verify output is a valid ZIP
		if _, err := zip.OpenReader(outputPath); err != nil {
			t.Errorf("output is not a valid ZIP: %v", err)
		}

		t.Logf("Processed %d files", filesProcessed)
	})

	t.Run("process with theme filter", func(t *testing.T) {
		// Create temp output file
		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Process only theme1
		mapping := map[string]string{"accent1": "accent6"}
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, []string{"theme1"}, "all", nil)

		if err != nil {
			t.Fatalf("ProcessPPTX failed: %v", err)
		}

		if filesProcessed == 0 {
			t.Error("expected to process some files, got 0")
		}

		// Verify output is a valid ZIP
		if _, err := zip.OpenReader(outputPath); err != nil {
			t.Errorf("output is not a valid ZIP: %v", err)
		}

		t.Logf("Processed %d files with theme filter", filesProcessed)
	})

	t.Run("process with multiple themes", func(t *testing.T) {
		// Create temp output file
		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Process theme1 and theme2
		mapping := map[string]string{"accent1": "accent6"}
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, []string{"theme1", "theme2"}, "all", nil)

		if err != nil {
			t.Fatalf("ProcessPPTX failed: %v", err)
		}

		if filesProcessed == 0 {
			t.Error("expected to process some files, got 0")
		}

		// Verify output is a valid ZIP
		if _, err := zip.OpenReader(outputPath); err != nil {
			t.Errorf("output is not a valid ZIP: %v", err)
		}

		t.Logf("Processed %d files with multiple theme filter", filesProcessed)
	})

	t.Run("atomic replacement in real file", func(t *testing.T) {
		// Create temp output file
		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Test atomic replacement: accent1→accent3, accent3→accent4
		mapping := map[string]string{
			"accent1": "accent3",
			"accent3": "accent4",
		}

		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, nil, "all", nil)

		if err != nil {
			t.Fatalf("ProcessPPTX failed: %v", err)
		}

		if filesProcessed == 0 {
			t.Error("expected to process some files, got 0")
		}

		// Verify output is a valid ZIP
		zipReader, err := zip.OpenReader(outputPath)
		if err != nil {
			t.Fatalf("output is not a valid ZIP: %v", err)
		}
		defer zipReader.Close()

		t.Logf("Processed %d files with atomic replacement", filesProcessed)
	})

	t.Run("process with content scope", func(t *testing.T) {
		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Process only content
		mapping := map[string]string{"accent1": "accent6"}
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, nil, "content", nil)

		if err != nil {
			t.Fatalf("ProcessPPTX failed: %v", err)
		}

		if filesProcessed == 0 {
			t.Error("expected to process some content files, got 0")
		}

		// Verify output is valid
		if _, err := zip.OpenReader(outputPath); err != nil {
			t.Errorf("output is not a valid ZIP: %v", err)
		}

		t.Logf("Processed %d content files", filesProcessed)
	})

	t.Run("process with master scope", func(t *testing.T) {
		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Process only master
		mapping := map[string]string{"accent1": "accent6"}
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, nil, "master", nil)

		if err != nil {
			t.Fatalf("ProcessPPTX failed: %v", err)
		}

		if filesProcessed == 0 {
			t.Error("expected to process some master files, got 0")
		}

		// Verify output is valid
		if _, err := zip.OpenReader(outputPath); err != nil {
			t.Errorf("output is not a valid ZIP: %v", err)
		}

		t.Logf("Processed %d master files", filesProcessed)
	})

	t.Run("scope and theme combination", func(t *testing.T) {
		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Process content in theme1 only
		mapping := map[string]string{"accent1": "accent6"}
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, []string{"theme1"}, "content", nil)

		if err != nil {
			t.Fatalf("ProcessPPTX failed: %v", err)
		}

		if filesProcessed == 0 {
			t.Error("expected to process some files, got 0")
		}

		// Verify output is valid
		if _, err := zip.OpenReader(outputPath); err != nil {
			t.Errorf("output is not a valid ZIP: %v", err)
		}

		t.Logf("Processed %d files with content scope + theme1 filter", filesProcessed)
	})

}


func TestProcessPPTX_Errors(t *testing.T) {
	t.Run("nonexistent input file", func(t *testing.T) {
		_, err := ProcessPPTX("/nonexistent/file.pptx", "/tmp/output.pptx", map[string]string{"accent1": "accent2"}, nil, "all", nil)
		if err == nil {
			t.Error("expected error for nonexistent file, got nil")
		}
	})

	t.Run("invalid output path", func(t *testing.T) {
		testPPTX := filepath.Join("testdata", "test.pptx")

		if _, err := os.Stat(testPPTX); os.IsNotExist(err) {
			t.Skip("test.pptx fixture not found")
		}

		// Try to write to invalid path
		_, err := ProcessPPTX(testPPTX, "/invalid/path/output.pptx", map[string]string{"accent1": "accent2"}, nil, "all", nil)
		if err == nil {
			t.Error("expected error for invalid output path, got nil")
		}
	})

	t.Run("nonexistent theme filter", func(t *testing.T) {
		testPPTX := filepath.Join("testdata", "test.pptx")

		if _, err := os.Stat(testPPTX); os.IsNotExist(err) {
			t.Skip("test.pptx fixture not found")
		}

		// Create temp output file
		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Process with non-existent theme - should error
		mapping := map[string]string{"accent1": "accent6"}
		_, err = ProcessPPTX(testPPTX, outputPath, mapping, []string{"theme999"}, "all", nil)

		if err == nil {
			t.Error("expected error for nonexistent theme, got nil")
		}

		// Should contain helpful error message
		expectedMsg := "theme(s) not found"
		if err != nil && !strings.Contains(err.Error(), expectedMsg) {
			t.Errorf("expected error to contain '%s', got: %v", expectedMsg, err)
		}
	})

	t.Run("invalid scope", func(t *testing.T) {
		testPPTX := filepath.Join("testdata", "test.pptx")

		if _, err := os.Stat(testPPTX); os.IsNotExist(err) {
			t.Skip("test.pptx fixture not found")
		}

		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Invalid scope should error
		mapping := map[string]string{"accent1": "accent6"}
		_, err = ProcessPPTX(testPPTX, outputPath, mapping, nil, "invalid", nil)

		if err == nil {
			t.Error("expected error for invalid scope, got nil")
		}

		if !strings.Contains(err.Error(), "invalid scope") {
			t.Errorf("expected 'invalid scope' in error, got: %v", err)
		}
	})

	t.Run("nonexistent slide", func(t *testing.T) {
		testPPTX := filepath.Join("testdata", "test.pptx")

		if _, err := os.Stat(testPPTX); os.IsNotExist(err) {
			t.Skip("test.pptx fixture not found")
		}

		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Process with non-existent slide - should error
		mapping := map[string]string{"accent1": "accent6"}
		_, err = ProcessPPTX(testPPTX, outputPath, mapping, nil, "content", []int{99})

		if err == nil {
			t.Error("expected error for nonexistent slide, got nil")
		}

		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("expected 'does not exist' in error, got: %v", err)
		}
	})
}

func TestProcessPPTX_SlideFiltering(t *testing.T) {
	testPPTX := filepath.Join("testdata", "test.pptx")

	if _, err := os.Stat(testPPTX); os.IsNotExist(err) {
		t.Skip("test.pptx fixture not found")
	}

	t.Run("filter specific slides", func(t *testing.T) {
		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Process only slides 3,4 (diagram + chart from research doc)
		mapping := map[string]string{"accent1": "accent6"}
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, nil, "content", []int{3, 4})

		if err != nil {
			t.Fatalf("ProcessPPTX failed: %v", err)
		}

		// Should process: slide3.xml, slide4.xml, + embedded content
		// From research: slide3 has 5 diagram files, slide4 has chart + 2 sub-files
		// Total: 2 slides + 5 diagram + 3 chart = 10 files minimum
		if filesProcessed < 10 {
			t.Errorf("Expected at least 10 files processed, got %d", filesProcessed)
		}

		// Verify output is valid
		if _, err := zip.OpenReader(outputPath); err != nil {
			t.Errorf("output is not a valid ZIP: %v", err)
		}

		t.Logf("Processed %d files for slides 3,4", filesProcessed)
	})

	t.Run("filter single slide", func(t *testing.T) {
		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Process only slide 1 (simple slide, no embedded content)
		mapping := map[string]string{"accent1": "accent6"}
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, nil, "content", []int{1})

		if err != nil {
			t.Fatalf("ProcessPPTX failed: %v", err)
		}

		// Should process just slide1.xml (1 file)
		if filesProcessed != 1 {
			t.Errorf("Expected 1 file processed, got %d", filesProcessed)
		}

		t.Logf("Processed %d file for slide 1", filesProcessed)
	})

	t.Run("filter slides with theme", func(t *testing.T) {
		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Process slides 1-4 with theme1 filter
		mapping := map[string]string{"accent1": "accent6"}
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, []string{"theme1"}, "content", []int{1, 2, 3, 4})

		if err != nil {
			t.Fatalf("ProcessPPTX failed: %v", err)
		}

		// Should process intersection of slides 1-4 AND theme1
		// From research: slides 1-4 are all theme1
		if filesProcessed == 0 {
			t.Error("expected to process some files, got 0")
		}

		t.Logf("Processed %d files for slides 1-4 with theme1 filter", filesProcessed)
	})

	t.Run("slide range", func(t *testing.T) {
		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Process slides 1-3 (range)
		mapping := map[string]string{"accent1": "accent6"}
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, nil, "content", []int{1, 2, 3})

		if err != nil {
			t.Fatalf("ProcessPPTX failed: %v", err)
		}

		// Should process slides 1,2,3 + slide 3's diagram
		// Minimum: 3 slides + 5 diagram files = 8
		if filesProcessed < 8 {
			t.Errorf("Expected at least 8 files processed, got %d", filesProcessed)
		}

		t.Logf("Processed %d files for slides 1-3", filesProcessed)
	})
}

func TestValidateScope(t *testing.T) {
	tests := []struct {
		name    string
		scope   string
		wantErr bool
	}{
		{"valid all", "all", false},
		{"valid content", "content", false},
		{"valid master", "master", false},
		{"invalid scope", "invalid", true},
		{"empty scope", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateScope(tt.scope)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateScope() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetXMLPatterns(t *testing.T) {
	tests := []struct {
		name         string
		scope        Scope
		wantContains []string
		wantExcludes []string
	}{
		{
			name:         "all scope",
			scope:        ScopeAll,
			wantContains: []string{"ppt/slides/", "ppt/slideMasters/", "ppt/charts/", "ppt/slideLayouts/"},
		},
		{
			name:         "content scope",
			scope:        ScopeContent,
			wantContains: []string{"ppt/slides/", "ppt/charts/", "ppt/diagrams/", "ppt/notesSlides/"},
			wantExcludes: []string{"ppt/slideMasters/", "ppt/slideLayouts/"},
		},
		{
			name:         "master scope",
			scope:        ScopeMaster,
			wantContains: []string{"ppt/slideMasters/", "ppt/slideLayouts/", "ppt/notesMasters/", "ppt/handoutMasters/"},
			wantExcludes: []string{"ppt/slides/", "ppt/charts/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := getXMLPatterns(tt.scope)

			for _, want := range tt.wantContains {
				if !containsString(patterns, want) {
					t.Errorf("getXMLPatterns(%s) missing %s", tt.scope, want)
				}
			}

			for _, exclude := range tt.wantExcludes {
				if containsString(patterns, exclude) {
					t.Errorf("getXMLPatterns(%s) should not contain %s", tt.scope, exclude)
				}
			}
		})
	}
}

// Helper function
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
