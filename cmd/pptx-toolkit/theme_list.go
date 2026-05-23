package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/antchfx/xmlquery"
	toolkitpptx "github.com/pmdci/pptx-toolkit/internal/pptx"
)

const (
	masterTypeSlide   = "slide"
	masterTypeNotes   = "notes"
	masterTypeHandout = "handout"
)

type ThemeSummary struct {
	Theme    *Theme
	Bindings []MasterBinding
}

type MasterBinding struct {
	MasterType string
	FileName   string
}

type masterReference struct {
	MasterType string
	PartPath   string
}

type relEntry struct {
	Target string
	Type   string
}

func ReadThemeSummaries(pptxPath string, themeFilter []string) ([]*ThemeSummary, error) {
	tempDir, err := os.MkdirTemp("", "pptx-toolkit-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := toolkitpptx.ExtractPPTX(pptxPath, tempDir); err != nil {
		return nil, err
	}

	return readThemeSummariesFromDir(tempDir, themeFilter)
}

func readThemeSummariesFromDir(tempDir string, themeFilter []string) ([]*ThemeSummary, error) {
	themes, err := readThemesFromDirStrict(tempDir)
	if err != nil {
		return nil, err
	}

	themes, err = filterThemes(themes, themeFilter)
	if err != nil {
		return nil, err
	}

	themeFiles := make(map[string]bool, len(themes))
	summaries := make([]*ThemeSummary, 0, len(themes))
	for _, theme := range themes {
		themeFiles[theme.FileName] = true
		summaries = append(summaries, &ThemeSummary{Theme: theme})
	}

	bindingMap, err := buildThemeBindingMap(tempDir, themeFiles)
	if err != nil {
		return nil, err
	}
	for _, summary := range summaries {
		summary.Bindings = bindingMap[summary.Theme.FileName]
		sortBindings(summary.Bindings)
	}

	return summaries, nil
}

func readThemesFromDirStrict(tempDir string) ([]*Theme, error) {
	themeFiles, err := filepath.Glob(filepath.Join(tempDir, "ppt", "theme", "*.xml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list theme files: %w", err)
	}
	sort.Strings(themeFiles)

	if len(themeFiles) == 0 {
		return nil, fmt.Errorf("no themes found")
	}

	themes := make([]*Theme, 0, len(themeFiles))
	for _, themeFile := range themeFiles {
		content, err := os.ReadFile(themeFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filepath.Base(themeFile), err)
		}

		theme, err := parseThemeXML(content, filepath.Base(themeFile))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(themeFile), err)
		}
		if theme.FontSchemeName == "" {
			return nil, fmt.Errorf("%s: no fontScheme element found", filepath.Base(themeFile))
		}

		themes = append(themes, theme)
	}

	return themes, nil
}

func buildThemeBindingMap(tempDir string, themeFiles map[string]bool) (map[string][]MasterBinding, error) {
	out := make(map[string][]MasterBinding)

	masterRefs, err := readMasterReferences(tempDir)
	if err != nil {
		return nil, err
	}

	for _, ref := range masterRefs {
		themeFileName, err := readMasterThemeBinding(tempDir, ref.PartPath)
		if err != nil {
			return nil, err
		}
		if themeFileName == "" || !themeFiles[themeFileName] {
			continue
		}

		out[themeFileName] = append(out[themeFileName], MasterBinding{
			MasterType: ref.MasterType,
			FileName:   path.Base(ref.PartPath),
		})
	}

	return out, nil
}

func readMasterReferences(tempDir string) ([]masterReference, error) {
	presentationPath := filepath.Join(tempDir, "ppt", "presentation.xml")
	relsPath := filepath.Join(tempDir, "ppt", "_rels", "presentation.xml.rels")

	presentation, err := parseXMLFile(presentationPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read presentation.xml: %w", err)
	}

	rels, err := parseRelationshipsFile(relsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read presentation relationships: %w", err)
	}

	var refs []masterReference
	refs = append(refs, extractMasterReferences(presentation, rels, masterTypeSlide, "sldMasterIdLst", "sldMasterId")...)
	refs = append(refs, extractMasterReferences(presentation, rels, masterTypeNotes, "notesMasterIdLst", "notesMasterId")...)
	refs = append(refs, extractMasterReferences(presentation, rels, masterTypeHandout, "handoutMasterIdLst", "handoutMasterId")...)

	return refs, nil
}

func extractMasterReferences(doc *xmlquery.Node, rels map[string]relEntry, masterType, listName, itemName string) []masterReference {
	xpath := fmt.Sprintf("/*[local-name()='presentation']/*[local-name()='%s']/*[local-name()='%s']", listName, itemName)
	nodes := xmlquery.Find(doc, xpath)

	var refs []masterReference
	for _, node := range nodes {
		relID := selectRelationshipID(node)
		if relID == "" {
			continue
		}

		entry, ok := rels[relID]
		if !ok {
			continue
		}

		refs = append(refs, masterReference{
			MasterType: masterType,
			PartPath:   entry.Target,
		})
	}

	return refs
}

const themeRelationshipType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme"

func readMasterThemeBinding(tempDir, masterPartPath string) (string, error) {
	masterRelsPath, ok := joinUnderBase(tempDir, relsPathForPart(masterPartPath))
	if !ok {
		return "", fmt.Errorf("master relationship path escapes package root for %s", path.Base(masterPartPath))
	}

	rels, err := parseRelationshipsFile(masterRelsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read relationships for %s: %w", path.Base(masterPartPath), err)
	}

	for _, entry := range rels {
		if entry.Type == themeRelationshipType {
			return path.Base(entry.Target), nil
		}
	}

	return "", nil
}

func parseXMLFile(filePath string) (*xmlquery.Node, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return xmlquery.Parse(file)
}

func parseRelationshipsFile(filePath string) (map[string]relEntry, error) {
	doc, err := parseXMLFile(filePath)
	if err != nil {
		return nil, err
	}

	ownerPart := ownerPartPathFromRelsPath(filepath.ToSlash(filePath))
	if ownerPart == "" {
		return nil, fmt.Errorf("cannot determine owner part for %s", filePath)
	}

	rels := make(map[string]relEntry)
	for _, node := range xmlquery.Find(doc, "/*[local-name()='Relationships']/*[local-name()='Relationship']") {
		relID := node.SelectAttr("Id")
		target := node.SelectAttr("Target")
		if relID == "" || target == "" {
			continue
		}

		resolvedTarget, ok := resolveRelationshipTarget(ownerPart, target)
		if !ok {
			continue
		}

		rels[relID] = relEntry{
			Target: resolvedTarget,
			Type:   node.SelectAttr("Type"),
		}
	}

	return rels, nil
}

func ownerPartPathFromRelsPath(filePath string) string {
	idx := strings.LastIndex(filePath, "/ppt/")
	if idx == -1 {
		return ""
	}

	pkgPath := strings.TrimPrefix(filePath[idx+1:], "ppt/")
	dir := path.Dir(pkgPath)
	base := strings.TrimSuffix(path.Base(pkgPath), ".rels")
	if path.Base(dir) == "_rels" {
		dir = path.Dir(dir)
	}

	return path.Join("ppt", dir, base)
}

func resolveRelationshipTarget(ownerPartPath, target string) (string, bool) {
	var resolved string
	if strings.HasPrefix(target, "/") {
		resolved = path.Clean(strings.TrimPrefix(target, "/"))
	} else {
		baseDir := path.Dir(ownerPartPath)
		resolved = path.Clean(path.Join(baseDir, target))
	}

	if resolved == "." || resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", false
	}

	return resolved, true
}

func relsPathForPart(partPath string) string {
	dir := path.Dir(partPath)
	base := path.Base(partPath)
	return path.Join(dir, "_rels", base+".rels")
}

func selectRelationshipID(node *xmlquery.Node) string {
	for _, attr := range node.Attr {
		if attr.Name.Local == "id" && attr.Name.Space != "" {
			return attr.Value
		}
	}
	return ""
}

func joinUnderBase(baseDir, relPath string) (string, bool) {
	cleanBase := filepath.Clean(baseDir)
	joined := filepath.Clean(filepath.Join(cleanBase, filepath.FromSlash(relPath)))

	rel, err := filepath.Rel(cleanBase, joined)
	if err != nil {
		return "", false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}

	return joined, true
}

func sortBindings(bindings []MasterBinding) {
	sort.Slice(bindings, func(i, j int) bool {
		oi := bindingTypeOrder(bindings[i].MasterType)
		oj := bindingTypeOrder(bindings[j].MasterType)
		if oi != oj {
			return oi < oj
		}
		return bindings[i].FileName < bindings[j].FileName
	})
}

func bindingTypeOrder(masterType string) int {
	switch masterType {
	case masterTypeSlide:
		return 0
	case masterTypeNotes:
		return 1
	case masterTypeHandout:
		return 2
	default:
		return 3
	}
}

func groupedBindings(bindings []MasterBinding) (slides, notes, handouts, unknown []string) {
	for _, binding := range bindings {
		switch binding.MasterType {
		case masterTypeSlide:
			slides = append(slides, binding.FileName)
		case masterTypeNotes:
			notes = append(notes, binding.FileName)
		case masterTypeHandout:
			handouts = append(handouts, binding.FileName)
		default:
			unknown = append(unknown, binding.FileName)
		}
	}
	return slides, notes, handouts, unknown
}
