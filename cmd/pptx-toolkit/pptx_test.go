package main

import (
	"archive/zip"
	"os"
	"path/filepath"
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
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, nil)

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
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, []string{"theme1"})

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
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, []string{"theme1", "theme2"})

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

		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, nil)

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

	t.Run("nonexistent theme filter", func(t *testing.T) {
		// Create temp output file
		outputFile, err := os.CreateTemp("", "output-*.pptx")
		if err != nil {
			t.Fatal(err)
		}
		outputPath := outputFile.Name()
		outputFile.Close()
		defer os.Remove(outputPath)

		// Process with non-existent theme
		mapping := map[string]string{"accent1": "accent6"}
		filesProcessed, err := ProcessPPTX(testPPTX, outputPath, mapping, []string{"theme999"})

		if err != nil {
			t.Fatalf("ProcessPPTX failed: %v", err)
		}

		// Should still create valid output, but process few/no files
		if _, err := zip.OpenReader(outputPath); err != nil {
			t.Errorf("output is not a valid ZIP: %v", err)
		}

		t.Logf("Processed %d files with nonexistent theme filter", filesProcessed)
	})
}

func TestProcessPPTX_Errors(t *testing.T) {
	t.Run("nonexistent input file", func(t *testing.T) {
		_, err := ProcessPPTX("/nonexistent/file.pptx", "/tmp/output.pptx", map[string]string{"accent1": "accent2"}, nil)
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
		_, err := ProcessPPTX(testPPTX, "/invalid/path/output.pptx", map[string]string{"accent1": "accent2"}, nil)
		if err == nil {
			t.Error("expected error for invalid output path, got nil")
		}
	})
}
