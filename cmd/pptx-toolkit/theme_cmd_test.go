package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestThemeColorListCommand_ColourAlias(t *testing.T) {
	skipIfNoFixture(t)

	var out bytes.Buffer
	var errBuf bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{"theme", "colour", "list", "--theme", "theme1", testPPTX})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list command via colour alias failed: %v; stderr=%s", err, errBuf.String())
	}

	output := out.String()
	if !strings.Contains(output, "━━━ theme1.xml ━━━") {
		t.Fatalf("expected theme1 block in output, got:\n%s", output)
	}
	if strings.Contains(output, "theme2.xml") {
		t.Fatalf("expected filtered output to omit theme2.xml, got:\n%s", output)
	}
}
