package pptx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ExtractPPTX extracts a PPTX zip archive into destDir.
func ExtractPPTX(pptxPath, destDir string) error {
	r, err := zip.OpenReader(pptxPath)
	if err != nil {
		return fmt.Errorf("failed to open PPTX: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		dest := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
			return err
		}

		out, err := os.Create(dest)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}

		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// RepackPPTX repacks an extracted PPTX tree into a .pptx archive.
func RepackPPTX(tempDir, outputPath string) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	return filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}
		zipFile, err := zipWriter.Create(filepath.ToSlash(relPath))
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(zipFile, bytes.NewReader(content))
		return err
	})
}
