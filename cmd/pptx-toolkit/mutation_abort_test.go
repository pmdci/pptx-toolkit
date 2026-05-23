package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestColorSwapCommand_OverwriteDeclineIsNotError(t *testing.T) {
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
	rootCmd.SetArgs([]string{"color", "swap", "accent1:accent2", testPPTX, outFile})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected nil error on user abort, got %v", err)
	}
	if !strings.Contains(out.String(), "Aborted.") {
		t.Fatalf("expected abort message, got:\n%s", out.String())
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

func TestThemeFontSetCommand_OverwriteDeclineIsNotError(t *testing.T) {
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
	rootCmd.SetArgs([]string{"theme", "font", "set", testPPTX, outFile, "--major", "Arial"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected nil error on user abort, got %v", err)
	}
	if !strings.Contains(out.String(), "Aborted.") {
		t.Fatalf("expected abort message, got:\n%s", out.String())
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
