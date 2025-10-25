package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseSlideRange(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr bool
	}{
		{
			name:  "single slide",
			input: "1",
			want:  []int{1},
		},
		{
			name:  "multiple slides",
			input: "1,3,5",
			want:  []int{1, 3, 5},
		},
		{
			name:  "simple range",
			input: "1-5",
			want:  []int{1, 2, 3, 4, 5},
		},
		{
			name:  "mixed format",
			input: "1,3,5-8,10",
			want:  []int{1, 3, 5, 6, 7, 8, 10},
		},
		{
			name:  "duplicates deduped",
			input: "1,1,3,3",
			want:  []int{1, 3},
		},
		{
			name:  "spaces trimmed",
			input: " 1 , 3 , 5-8 ",
			want:  []int{1, 3, 5, 6, 7, 8},
		},
		{
			name:    "empty string",
			input:   "",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "invalid range format",
			input:   "1-",
			wantErr: true,
		},
		{
			name:    "reverse range",
			input:   "5-2",
			wantErr: true,
		},
		{
			name:    "zero slide",
			input:   "0",
			wantErr: true,
		},
		{
			name:    "negative slide",
			input:   "-1",
			wantErr: true,
		},
		{
			name:    "invalid range with text",
			input:   "1-a",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSlideRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSlideRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseSlideRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildSlideMapping(t *testing.T) {
	// Use test.pptx fixture
	testPPTX := filepath.Join("testdata", "test.pptx")

	if _, err := os.Stat(testPPTX); os.IsNotExist(err) {
		t.Skip("test.pptx fixture not found")
	}

	// Extract to temp directory
	tempDir, err := os.MkdirTemp("", "slides-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Extract PPTX
	zipReader, err := zip.OpenReader(testPPTX)
	if err != nil {
		t.Fatal(err)
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		filePath := filepath.Join(tempDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			t.Fatal(err)
		}

		outFile, err := os.Create(filePath)
		if err != nil {
			t.Fatal(err)
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			t.Fatal(err)
		}

		_, err = outFile.ReadFrom(rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			t.Fatal(err)
		}
	}

	// Build mapping
	mapping, err := BuildSlideMapping(tempDir)
	if err != nil {
		t.Fatalf("BuildSlideMapping() error = %v", err)
	}

	// Verify we have 12 slides (from research doc)
	if len(mapping) != 12 {
		t.Errorf("Expected 12 slides, got %d", len(mapping))
	}

	// Verify slide 1 maps to a path in ppt/slides/
	slide1Path := mapping[1]
	if !filepath.IsAbs(slide1Path) {
		slide1Path = filepath.Join(tempDir, slide1Path)
	}

	if !filepath.HasPrefix(slide1Path, filepath.Join(tempDir, "ppt", "slides")) {
		t.Errorf("Slide 1 path doesn't have expected prefix: %s", slide1Path)
	}

	// Verify all slide numbers are sequential (1-12)
	for i := 1; i <= 12; i++ {
		if _, exists := mapping[i]; !exists {
			t.Errorf("Slide %d not found in mapping", i)
		}
	}
}

func TestValidateSlideNumbers(t *testing.T) {
	testPPTX := filepath.Join("testdata", "test.pptx")

	if _, err := os.Stat(testPPTX); os.IsNotExist(err) {
		t.Skip("test.pptx fixture not found")
	}

	// Extract to temp directory
	tempDir, err := os.MkdirTemp("", "slides-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Extract PPTX
	zipReader, err := zip.OpenReader(testPPTX)
	if err != nil {
		t.Fatal(err)
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		filePath := filepath.Join(tempDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			t.Fatal(err)
		}

		outFile, err := os.Create(filePath)
		if err != nil {
			t.Fatal(err)
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			t.Fatal(err)
		}

		_, err = outFile.ReadFrom(rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name    string
		slides  []int
		wantErr bool
	}{
		{
			name:    "valid slides",
			slides:  []int{1, 3, 5},
			wantErr: false,
		},
		{
			name:    "all slides valid",
			slides:  []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			wantErr: false,
		},
		{
			name:    "slide beyond range",
			slides:  []int{1, 15},
			wantErr: true,
		},
		{
			name:    "multiple slides beyond range",
			slides:  []int{15, 99},
			wantErr: true,
		},
		{
			name:    "empty slice",
			slides:  []int{},
			wantErr: false,
		},
		{
			name:    "nil slice",
			slides:  nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSlideNumbers(tempDir, tt.slides)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSlideNumbers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetSlideContent(t *testing.T) {
	testPPTX := filepath.Join("testdata", "test.pptx")

	if _, err := os.Stat(testPPTX); os.IsNotExist(err) {
		t.Skip("test.pptx fixture not found")
	}

	// Extract to temp directory
	tempDir, err := os.MkdirTemp("", "slides-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Extract PPTX
	zipReader, err := zip.OpenReader(testPPTX)
	if err != nil {
		t.Fatal(err)
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		filePath := filepath.Join(tempDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			t.Fatal(err)
		}

		outFile, err := os.Create(filePath)
		if err != nil {
			t.Fatal(err)
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			t.Fatal(err)
		}

		_, err = outFile.ReadFrom(rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("slide with diagram", func(t *testing.T) {
		// Slide 3 has a diagram (from research doc)
		files, err := GetSlideContent(tempDir, []int{3})
		if err != nil {
			t.Fatalf("GetSlideContent() error = %v", err)
		}

		// Should include slide3.xml + 5 diagram files
		expectedFiles := []string{
			"ppt/slides/slide3.xml",
			"ppt/diagrams/data1.xml",
			"ppt/diagrams/layout1.xml",
			"ppt/diagrams/colors1.xml",
			"ppt/diagrams/quickStyle1.xml",
			"ppt/diagrams/drawing1.xml",
		}

		for _, expected := range expectedFiles {
			if !files[expected] {
				t.Errorf("Expected file %s not found in result", expected)
			}
		}

		t.Logf("Slide 3 content: %d files", len(files))
	})

	t.Run("slide with chart", func(t *testing.T) {
		// Slide 4 has a chart (from research doc)
		files, err := GetSlideContent(tempDir, []int{4})
		if err != nil {
			t.Fatalf("GetSlideContent() error = %v", err)
		}

		// Should include slide4.xml + chart + chart sub-files
		expectedFiles := []string{
			"ppt/slides/slide4.xml",
			"ppt/charts/chart1.xml",
			"ppt/charts/colors1.xml",
			"ppt/charts/style1.xml",
		}

		for _, expected := range expectedFiles {
			if !files[expected] {
				t.Errorf("Expected file %s not found in result", expected)
			}
		}

		t.Logf("Slide 4 content: %d files", len(files))
	})

	t.Run("multiple slides", func(t *testing.T) {
		// Slides 3 and 4 (diagram + chart)
		files, err := GetSlideContent(tempDir, []int{3, 4})
		if err != nil {
			t.Fatalf("GetSlideContent() error = %v", err)
		}

		// Should include both slides + their embedded content
		minExpected := 10 // 2 slides + 5 diagram files + 3 chart files
		if len(files) < minExpected {
			t.Errorf("Expected at least %d files, got %d", minExpected, len(files))
		}

		t.Logf("Slides 3,4 content: %d files", len(files))
	})

	t.Run("empty slice", func(t *testing.T) {
		files, err := GetSlideContent(tempDir, []int{})
		if err != nil {
			t.Fatalf("GetSlideContent() error = %v", err)
		}

		if files != nil {
			t.Errorf("Expected nil for empty slice, got %v", files)
		}
	})
}

func TestResolveRelativePath(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		target   string
		want     string
	}{
		{
			name:     "chart from slide",
			basePath: "/tmp/ppt/slides/slide1.xml",
			target:   "../charts/chart1.xml",
			want:     filepath.Clean("/tmp/ppt/charts/chart1.xml"),
		},
		{
			name:     "diagram from slide",
			basePath: "/tmp/ppt/slides/slide3.xml",
			target:   "../diagrams/data1.xml",
			want:     filepath.Clean("/tmp/ppt/diagrams/data1.xml"),
		},
		{
			name:     "colors from chart",
			basePath: "/tmp/ppt/charts/chart1.xml",
			target:   "colors1.xml",
			want:     filepath.Clean("/tmp/ppt/charts/colors1.xml"),
		},
		{
			name:     "layout from slide",
			basePath: "/tmp/ppt/slides/slide1.xml",
			target:   "../slideLayouts/slideLayout1.xml",
			want:     filepath.Clean("/tmp/ppt/slideLayouts/slideLayout1.xml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveRelativePath(tt.basePath, tt.target)
			if got != tt.want {
				t.Errorf("resolveRelativePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
