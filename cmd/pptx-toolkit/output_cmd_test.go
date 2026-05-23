package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestPrepareMutation_AbortStopsMutation(t *testing.T) {
	skipIfNoFixture(t)

	outFile := filepath.Join(t.TempDir(), "out.pptx")
	if err := os.WriteFile(outFile, []byte("existing"), 0644); err != nil {
		t.Fatalf("write output fixture failed: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("n\n"))
	var out bytes.Buffer
	cmd.SetOut(&out)

	err := PrepareMutation(cmd, testPPTX, outFile)
	if !errors.Is(err, errMutationAborted) {
		t.Fatalf("err = %v, want errMutationAborted", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("Aborted.")) {
		t.Fatalf("expected abort message, got %q", out.String())
	}
}
