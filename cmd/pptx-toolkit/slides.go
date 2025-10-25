package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/antchfx/xmlquery"
)

// ParseSlideRange parses a slide range string like "1,3,5-8" into a sorted slice of slide numbers
// Deduplicates silently and validates format
func ParseSlideRange(flag string) ([]int, error) {
	if flag == "" {
		return nil, nil
	}

	slides := make(map[int]bool)

	parts := strings.Split(flag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, "-") {
			// Range: "5-8"
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format '%s' (expected '1-5')", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid slide number '%s'", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid slide number '%s'", rangeParts[1])
			}

			if start < 1 {
				return nil, fmt.Errorf("invalid slide number %d (must be ≥ 1)", start)
			}

			if start > end {
				return nil, fmt.Errorf("invalid range %d-%d (start > end)", start, end)
			}

			for i := start; i <= end; i++ {
				slides[i] = true
			}
		} else {
			// Single slide: "3"
			slideNum, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid slide number '%s'", part)
			}

			if slideNum < 1 {
				return nil, fmt.Errorf("invalid slide number %d (must be ≥ 1)", slideNum)
			}

			slides[slideNum] = true
		}
	}

	if len(slides) == 0 {
		return nil, fmt.Errorf("no slides specified")
	}

	// Convert map to sorted slice
	result := make([]int, 0, len(slides))
	for slide := range slides {
		result = append(result, slide)
	}
	sort.Ints(result)

	return result, nil
}

// BuildSlideMapping creates a map of visual slide number to file path
// Parses presentation.xml for order (NOT file names)
func BuildSlideMapping(tempDir string) (map[int]string, error) {
	mapping := make(map[int]string)

	// Parse presentation.xml
	presentationPath := filepath.Join(tempDir, "ppt", "presentation.xml")
	presentationFile, err := os.Open(presentationPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open presentation.xml: %w", err)
	}
	defer presentationFile.Close()

	doc, err := xmlquery.Parse(presentationFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse presentation.xml: %w", err)
	}

	// Find slide IDs in order
	slideNodes := xmlquery.Find(doc, "//p:sldIdLst/p:sldId")
	if len(slideNodes) == 0 {
		// Try without namespace prefix
		slideNodes = xmlquery.Find(doc, "//sldIdLst/sldId")
	}

	if len(slideNodes) == 0 {
		return nil, fmt.Errorf("no slides found in presentation")
	}

	// Parse relationships file
	relsPath := filepath.Join(tempDir, "ppt", "_rels", "presentation.xml.rels")
	relsFile, err := os.Open(relsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open presentation.xml.rels: %w", err)
	}
	defer relsFile.Close()

	relsDoc, err := xmlquery.Parse(relsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse presentation.xml.rels: %w", err)
	}

	// Build mapping: visual slide number → file path
	for i, slideNode := range slideNodes {
		rId := slideNode.SelectAttr("r:id")
		if rId == "" {
			// Try without namespace prefix
			rId = slideNode.SelectAttr("id")
		}

		if rId == "" {
			continue
		}

		// Find relationship target
		xpath := fmt.Sprintf("//Relationship[@Id='%s']", rId)
		targetNode := xmlquery.FindOne(relsDoc, xpath)
		if targetNode == nil {
			continue
		}

		target := targetNode.SelectAttr("Target")
		if target == "" {
			continue
		}

		visualSlideNum := i + 1 // 1-indexed
		// target is like "slides/slide1.xml", prepend "ppt/"
		mapping[visualSlideNum] = filepath.Join("ppt", filepath.FromSlash(target))
	}

	return mapping, nil
}

// ValidateSlideNumbers checks if all requested slides exist in the presentation
// Reports all invalid slides together
func ValidateSlideNumbers(tempDir string, slideNums []int) error {
	if len(slideNums) == 0 {
		return nil
	}

	// Build slide mapping to get total count
	mapping, err := BuildSlideMapping(tempDir)
	if err != nil {
		return err
	}

	totalSlides := len(mapping)

	// Check each requested slide
	var invalid []int
	for _, slideNum := range slideNums {
		if slideNum > totalSlides {
			invalid = append(invalid, slideNum)
		}
	}

	if len(invalid) > 0 {
		if len(invalid) == 1 {
			return fmt.Errorf("slide %d does not exist (presentation has %d slides)", invalid[0], totalSlides)
		}
		// Multiple invalid slides
		invalidStrs := make([]string, len(invalid))
		for i, num := range invalid {
			invalidStrs[i] = fmt.Sprintf("%d", num)
		}
		return fmt.Errorf("slides %s do not exist (presentation has %d slides)",
			strings.Join(invalidStrs, ", "), totalSlides)
	}

	return nil
}

// GetSlideContent returns all files that belong to the specified slides
// Includes: slide files, charts + sub-files, diagrams (all 5 files), notes
func GetSlideContent(tempDir string, slideNums []int) (map[string]bool, error) {
	if len(slideNums) == 0 {
		return nil, nil
	}

	filesToProcess := make(map[string]bool)

	// Build slide mapping
	slideMapping, err := BuildSlideMapping(tempDir)
	if err != nil {
		return nil, err
	}

	// For each requested slide
	for _, slideNum := range slideNums {
		slideRelPath, exists := slideMapping[slideNum]
		if !exists {
			continue
		}

		// Store relative path for matching
		relPath := filepath.ToSlash(slideRelPath)
		filesToProcess[relPath] = true

		// Build absolute path for file operations
		slidePath := filepath.Join(tempDir, slideRelPath)

		// Find slide's relationships
		slideDir := filepath.Dir(slidePath)
		slideName := filepath.Base(slidePath)
		relsPath := filepath.Join(slideDir, "_rels", slideName+".rels")

		if _, err := os.Stat(relsPath); os.IsNotExist(err) {
			continue
		}

		// Parse relationships
		relsFile, err := os.Open(relsPath)
		if err != nil {
			continue
		}
		relsDoc, err := xmlquery.Parse(relsFile)
		relsFile.Close()
		if err != nil {
			continue
		}

		// Find all relationships
		rels := xmlquery.Find(relsDoc, "//Relationship")

		for _, rel := range rels {
			relType := rel.SelectAttr("Type")
			target := rel.SelectAttr("Target")

			if target == "" {
				continue
			}

			// Process charts
			if strings.HasSuffix(relType, "/chart") {
				chartPath := resolveRelativePath(slidePath, target)
				chartRelPath, _ := filepath.Rel(tempDir, chartPath)
				chartRelPath = filepath.ToSlash(chartRelPath)
				filesToProcess[chartRelPath] = true

				// Include chart sub-files (colors, style)
				chartDir := filepath.Dir(chartPath)
				chartName := filepath.Base(chartPath)
				chartRelsPath := filepath.Join(chartDir, "_rels", chartName+".rels")

				if _, err := os.Stat(chartRelsPath); err == nil {
					chartRelsFile, err := os.Open(chartRelsPath)
					if err == nil {
						chartRelsDoc, err := xmlquery.Parse(chartRelsFile)
						chartRelsFile.Close()
						if err == nil {
							subRels := xmlquery.Find(chartRelsDoc, "//Relationship")
							for _, subRel := range subRels {
								subTarget := subRel.SelectAttr("Target")
								if subTarget != "" {
									subPath := resolveRelativePath(chartPath, subTarget)
									// Only include XML files (not embedded Excel data)
									if strings.HasSuffix(subPath, ".xml") {
										subRelPath, _ := filepath.Rel(tempDir, subPath)
										subRelPath = filepath.ToSlash(subRelPath)
										filesToProcess[subRelPath] = true
									}
								}
							}
						}
					}
				}
			}

			// Process diagrams (all 5 types)
			diagramTypes := []string{
				"/diagramData",
				"/diagramLayout",
				"/diagramColors",
				"/diagramQuickStyle",
				"/diagramDrawing",
			}

			for _, diagType := range diagramTypes {
				if strings.HasSuffix(relType, diagType) {
					diagPath := resolveRelativePath(slidePath, target)
					diagRelPath, _ := filepath.Rel(tempDir, diagPath)
					diagRelPath = filepath.ToSlash(diagRelPath)
					filesToProcess[diagRelPath] = true
					break
				}
			}

			// Process notes slides
			if strings.HasSuffix(relType, "/notesSlide") {
				notesPath := resolveRelativePath(slidePath, target)
				notesRelPath, _ := filepath.Rel(tempDir, notesPath)
				notesRelPath = filepath.ToSlash(notesRelPath)
				filesToProcess[notesRelPath] = true
			}
		}
	}

	return filesToProcess, nil
}

// resolveRelativePath resolves a relative path like "../charts/chart1.xml"
// from a base path like "/tmp/ppt/slides/slide1.xml"
func resolveRelativePath(basePath, target string) string {
	baseDir := filepath.Dir(basePath)
	targetPath := filepath.Join(baseDir, filepath.FromSlash(target))
	return filepath.Clean(targetPath)
}
