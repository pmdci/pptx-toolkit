package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/antchfx/xmlquery"
	toolkitpptx "github.com/pmdci/pptx-toolkit/internal/pptx"
)

const presentationMLNamespace = "http://schemas.openxmlformats.org/presentationml/2006/main"

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

	if err := toolkitpptx.ExtractPPTX(pptxPath, tempDir); err != nil {
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

var (
	matchingNameRe = regexp.MustCompile(`\s+matchingName\s*=\s*(?:"[^"]*"|'[^']*')`)
)

// removeMatchingNameAttr removes the matchingName attribute from the sldLayout
// start tag using the Decoder.InputOffset approach: parse only to locate the
// exact byte range of the start tag, then apply a regex to that slice only.
// The rest of the file is byte-identical to the input.
func removeMatchingNameAttr(content []byte) ([]byte, error) {
	preOffset, postOffset, err := findLayoutStartTagRange(content)
	if err != nil {
		return nil, err
	}
	if preOffset == -1 {
		return content, nil
	}
	startTag := matchingNameRe.ReplaceAll(content[preOffset:postOffset], nil)
	return replaceByteRange(content, preOffset, postOffset, startTag), nil
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

	if err := toolkitpptx.ExtractPPTX(inputPath, tempDir); err != nil {
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

	if err := toolkitpptx.RepackPPTX(tempDir, outputPath); err != nil {
		return 0, err
	}
	return count, nil
}

func findLayoutStartTagRange(content []byte) (int, int, error) {
	return findStartElementRange(content, func(se xml.StartElement, _ int) bool {
		return se.Name.Local == "sldLayout" && se.Name.Space == presentationMLNamespace
	}, "layout XML")
}

func findLayoutCSldStartTagRange(content []byte) (int, int, error) {
	foundLayout := false
	return findStartElementRange(content, func(se xml.StartElement, depth int) bool {
		if se.Name.Local == "sldLayout" && se.Name.Space == presentationMLNamespace {
			foundLayout = true
			return false
		}
		return foundLayout && depth == 1 && se.Name.Local == "cSld" && se.Name.Space == presentationMLNamespace
	}, "layout XML")
}

func setLayoutProperty(content []byte, targetProperty, value string) ([]byte, error) {
	var (
		start int
		end   int
		err   error
	)

	switch targetProperty {
	case layoutPropertyMatchingName:
		start, end, err = findLayoutStartTagRange(content)
		if err != nil {
			return nil, err
		}
		if start == -1 {
			return nil, fmt.Errorf("layout root element not found")
		}
		startTag := setAttrOnStartTag(content[start:end], "matchingName", value, matchingNameRe)
		return replaceByteRange(content, start, end, startTag), nil
	case layoutPropertyName:
		start, end, err = findLayoutCSldStartTagRange(content)
		if err != nil {
			return nil, err
		}
		if start == -1 {
			return nil, fmt.Errorf("layout cSld element not found")
		}
		startTag := setAttrOnStartTag(content[start:end], "name", value, nameAttrRe)
		return replaceByteRange(content, start, end, startTag), nil
	default:
		return nil, fmt.Errorf("unsupported target property: %s", targetProperty)
	}
}

func layoutPropertyValue(info *LayoutInfo, property string) string {
	switch property {
	case layoutPropertyName:
		return info.Name
	case layoutPropertyMatchingName:
		return info.MatchingName
	default:
		return ""
	}
}

// SetLayoutProperty applies a mapping to all layouts matching the given filters.
// Property-to-property copies are a no-op when the source property is absent.
// The output PPTX is always written, even when no layouts are changed.
func SetLayoutProperty(inputPath, outputPath string, mapping *LayoutSetMapping, filters LayoutFilters) (int, error) {
	if mapping == nil {
		return 0, fmt.Errorf("layout set mapping cannot be nil")
	}
	if mapping.SourceKind == LayoutSetSourceLiteral {
		if err := ValidateName(mapping.SourceLiteral); err != nil {
			return 0, err
		}
	}

	tempDir, err := os.MkdirTemp("", "pptx-toolkit-*")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := toolkitpptx.ExtractPPTX(inputPath, tempDir); err != nil {
		return 0, err
	}

	layouts, err := readLayoutsFromDir(tempDir, filters)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, l := range layouts {
		var value string
		switch mapping.SourceKind {
		case LayoutSetSourceProperty:
			value = layoutPropertyValue(l, mapping.SourceProperty)
			if value == "" {
				continue
			}
		case LayoutSetSourceLiteral:
			value = mapping.SourceLiteral
		default:
			return 0, fmt.Errorf("unsupported source kind: %s", mapping.SourceKind)
		}

		filePath := filepath.Join(tempDir, "ppt", "slideLayouts", l.FileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return 0, fmt.Errorf("failed to read %s: %w", l.FileName, err)
		}

		modified, err := setLayoutProperty(content, mapping.TargetProperty, value)
		if err != nil {
			return 0, fmt.Errorf("failed to process %s: %w", l.FileName, err)
		}
		if bytes.Equal(modified, content) {
			continue
		}
		if err := os.WriteFile(filePath, modified, 0644); err != nil {
			return 0, fmt.Errorf("failed to write %s: %w", l.FileName, err)
		}
		count++
	}

	if err := toolkitpptx.RepackPPTX(tempDir, outputPath); err != nil {
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
