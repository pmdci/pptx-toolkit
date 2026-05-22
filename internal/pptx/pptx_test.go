package pptx

import (
	"os"
	"path/filepath"
	"testing"
)

var testPPTX = filepath.Join("..", "..", "cmd", "pptx-toolkit", "testdata", "test.pptx")

func skipIfNoFixture(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(testPPTX); os.IsNotExist(err) {
		t.Skip("test.pptx fixture not found")
	}
}

func TestExtractPPTX(t *testing.T) {
	skipIfNoFixture(t)

	destDir := t.TempDir()
	if err := ExtractPPTX(testPPTX, destDir); err != nil {
		t.Fatalf("ExtractPPTX failed: %v", err)
	}

	presentationPath := filepath.Join(destDir, "ppt", "presentation.xml")
	if _, err := os.Stat(presentationPath); os.IsNotExist(err) {
		t.Error("presentation.xml not found after extraction")
	}

	layouts, err := filepath.Glob(filepath.Join(destDir, "ppt", "slideLayouts", "slideLayout*.xml"))
	if err != nil || len(layouts) == 0 {
		t.Error("no layout files found after extraction")
	}
}

func TestExtractPPTX_InvalidFile(t *testing.T) {
	destDir := t.TempDir()
	err := ExtractPPTX("nonexistent.pptx", destDir)
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}
