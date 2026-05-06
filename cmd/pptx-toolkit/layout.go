package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/antchfx/xmlquery"
)

// LayoutInfo holds all inspectable fields for a single slide layout.
type LayoutInfo struct {
	FileName     string // e.g. "slideLayout12.xml"
	LayoutID     string // e.g. "slideLayout12"
	Name         string // p:cSld/@name
	MatchingName string // p:sldLayout/@matchingName, empty if absent
	Master       string // e.g. "slideMaster1.xml"
	Theme        string // e.g. "theme1.xml"
	UsedBySlides []int  // visual slide numbers, sorted
}

// LayoutFilters holds optional filter values for ReadLayouts.
type LayoutFilters struct {
	LayoutID     string
	Name         string
	MatchingName string
	Theme        []string
}

// ReadLayouts reads all slide layouts from a PPTX file and applies filters.
func ReadLayouts(pptxPath string, filters LayoutFilters) ([]*LayoutInfo, error) {
	tempDir, err := os.MkdirTemp("", "pptx-toolkit-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := extractPPTX(pptxPath, tempDir); err != nil {
		return nil, err
	}

	return readLayoutsFromDir(tempDir, filters)
}

func readLayoutsFromDir(tempDir string, filters LayoutFilters) ([]*LayoutInfo, error) {
	layoutToMaster, err := buildLayoutToMasterMapping(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read layout-to-master relationships: %w", err)
	}

	masterToTheme, err := buildThemeRelationships(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read master-to-theme relationships: %w", err)
	}

	slideToLayouts, err := buildSlideToLayoutMapping(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read slide-to-layout relationships: %w", err)
	}

	themeFiles := normalizeThemeFilter(filters.Theme)

	layoutsDir := filepath.Join(tempDir, "ppt", "slideLayouts")
	files, err := filepath.Glob(filepath.Join(layoutsDir, "slideLayout*.xml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list layout files: %w", err)
	}

	var layouts []*LayoutInfo
	for _, filePath := range files {
		fileName := filepath.Base(filePath)
		layoutID := strings.TrimSuffix(fileName, ".xml")

		name, matchingName, err := parseLayoutXML(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layout file %s: %w", fileName, err)
		}

		master := layoutToMaster[fileName]
		theme := masterToTheme[master]

		info := &LayoutInfo{
			FileName:     fileName,
			LayoutID:     layoutID,
			Name:         name,
			MatchingName: matchingName,
			Master:       master,
			Theme:        theme,
			UsedBySlides: slideToLayouts[fileName],
		}

		if !matchesLayoutFilters(info, filters, themeFiles) {
			continue
		}

		layouts = append(layouts, info)
	}

	sort.Slice(layouts, func(i, j int) bool {
		ni, _ := layoutNumber(layouts[i].FileName)
		nj, _ := layoutNumber(layouts[j].FileName)
		return ni < nj
	})

	return layouts, nil
}

func matchesLayoutFilters(info *LayoutInfo, filters LayoutFilters, themeFiles map[string]bool) bool {
	if filters.LayoutID != "" && info.LayoutID != filters.LayoutID {
		return false
	}
	if filters.Name != "" && info.Name != filters.Name {
		return false
	}
	if filters.MatchingName != "" && info.MatchingName != filters.MatchingName {
		return false
	}
	if len(themeFiles) > 0 && !themeFiles[info.Theme] {
		return false
	}
	return true
}

func normalizeThemeFilter(themes []string) map[string]bool {
	out := make(map[string]bool)
	for _, t := range themes {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if strings.HasSuffix(t, ".xml") {
			out[t] = true
		} else {
			out[t+".xml"] = true
		}
	}
	return out
}

// parseLayoutXML extracts name and matchingName from a slideLayout XML file.
func parseLayoutXML(filePath string) (name, matchingName string, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	doc, err := xmlquery.Parse(f)
	if err != nil {
		return "", "", err
	}

	root := xmlquery.FindOne(doc, "/*[local-name()='sldLayout']")
	if root != nil {
		matchingName = root.SelectAttr("matchingName")
	}

	cSld := xmlquery.FindOne(doc, "/*[local-name()='sldLayout']/*[local-name()='cSld']")
	if cSld != nil {
		name = cSld.SelectAttr("name")
	}

	return name, matchingName, nil
}

var matchingNameRe = regexp.MustCompile(`\s+matchingName\s*=\s*(?:"[^"]*"|'[^']*')`)

// removeMatchingNameAttr removes the matchingName attribute from the sldLayout
// start tag using the Decoder.InputOffset approach: parse only to locate the
// exact byte range of the start tag, then apply a regex to that slice only.
// The rest of the file is byte-identical to the input.
func removeMatchingNameAttr(content []byte) ([]byte, error) {
	d := xml.NewDecoder(bytes.NewReader(content))
	for {
		preOffset := d.InputOffset()
		tok, err := d.Token()
		if err == io.EOF {
			return content, nil
		}
		if err != nil {
			return nil, fmt.Errorf("parsing layout XML: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local == "sldLayout" &&
			se.Name.Space == "http://schemas.openxmlformats.org/presentationml/2006/main" {
			postOffset := d.InputOffset()
			startTag := matchingNameRe.ReplaceAll(content[preOffset:postOffset], nil)
			out := make([]byte, 0, len(content))
			out = append(out, content[:preOffset]...)
			out = append(out, startTag...)
			out = append(out, content[postOffset:]...)
			return out, nil
		}
	}
}

// RemoveLayoutMatchingName removes the matchingName attribute from layouts
// matching the given filters. Writes the modified PPTX to outputPath even
// when no layouts are changed. Returns the number of layouts modified.
func RemoveLayoutMatchingName(inputPath, outputPath string, filters LayoutFilters) (int, error) {
	tempDir, err := os.MkdirTemp("", "pptx-toolkit-*")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := extractPPTX(inputPath, tempDir); err != nil {
		return 0, err
	}

	layouts, err := readLayoutsFromDir(tempDir, filters)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, l := range layouts {
		if l.MatchingName == "" {
			continue
		}
		filePath := filepath.Join(tempDir, "ppt", "slideLayouts", l.FileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return 0, fmt.Errorf("failed to read %s: %w", l.FileName, err)
		}
		modified, err := removeMatchingNameAttr(content)
		if err != nil {
			return 0, fmt.Errorf("failed to process %s: %w", l.FileName, err)
		}
		if err := os.WriteFile(filePath, modified, 0644); err != nil {
			return 0, fmt.Errorf("failed to write %s: %w", l.FileName, err)
		}
		count++
	}

	if err := repackPPTX(tempDir, outputPath); err != nil {
		return 0, err
	}
	return count, nil
}

// buildSlideToLayoutMapping returns a map of layout filename → sorted visual slide numbers.
func buildSlideToLayoutMapping(tempDir string) (map[string][]int, error) {
	result := make(map[string][]int)

	slideMapping, err := BuildSlideMapping(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read slide order from presentation.xml: %w", err)
	}

	for visualNum, slideRelPath := range slideMapping {
		slidePath := filepath.Join(tempDir, slideRelPath)
		slideName := filepath.Base(slidePath)
		relsPath := filepath.Join(filepath.Dir(slidePath), "_rels", slideName+".rels")

		f, err := os.Open(relsPath)
		if err != nil {
			continue
		}
		doc, err := xmlquery.Parse(f)
		f.Close()
		if err != nil {
			continue
		}

		node := xmlquery.FindOne(doc, "//Relationship[@Type='http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout']")
		if node == nil {
			continue
		}

		layoutFileName := filepath.Base(node.SelectAttr("Target"))
		result[layoutFileName] = append(result[layoutFileName], visualNum)
	}

	for k := range result {
		sort.Ints(result[k])
	}

	return result, nil
}

var layoutNumRe = regexp.MustCompile(`slideLayout(\d+)\.xml$`)

func layoutNumber(fileName string) (int, error) {
	m := layoutNumRe.FindStringSubmatch(fileName)
	if len(m) < 2 {
		return 0, fmt.Errorf("cannot parse layout number from %q", fileName)
	}
	return strconv.Atoi(m[1])
}

// extractPPTX extracts a PPTX zip archive into destDir.
func extractPPTX(pptxPath, destDir string) error {
	r, err := zip.OpenReader(pptxPath)
	if err != nil {
		return fmt.Errorf("failed to open PPTX: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		dest := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(dest, os.ModePerm)
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
