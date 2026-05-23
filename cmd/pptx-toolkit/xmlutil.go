package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var (
	nameAttrRe = regexp.MustCompile(`\s+name\s*=\s*(?:"[^"]*"|'[^']*')`)

	xmlAttributeEscaper = strings.NewReplacer(
		"&", "&amp;",
		`"`, "&quot;",
		"'", "&apos;",
		"<", "&lt;",
		">", "&gt;",
	)

	xmlTextEscaper = strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
)

func escapeXMLAttributeValue(value string) string {
	return xmlAttributeEscaper.Replace(value)
}

func escapeXMLTextContent(value string) string {
	return xmlTextEscaper.Replace(value)
}

func findStartElementRange(content []byte, matcher func(xml.StartElement, int) bool, context string) (int, int, error) {
	d := xml.NewDecoder(bytes.NewReader(content))
	depth := 0
	for {
		preOffset := int(d.InputOffset())
		tok, err := d.Token()
		if err == io.EOF {
			return -1, -1, nil
		}
		if err != nil {
			return 0, 0, fmt.Errorf("parsing %s: %w", context, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if matcher(t, depth) {
				return preOffset, int(d.InputOffset()), nil
			}
			depth++
		case xml.EndElement:
			if depth > 0 {
				depth--
			}
		}
	}
}

func replaceByteRange(content []byte, start, end int, replacement []byte) []byte {
	out := make([]byte, 0, len(content)-((end-start)-len(replacement)))
	out = append(out, content[:start]...)
	out = append(out, replacement...)
	out = append(out, content[end:]...)
	return out
}

func setAttrOnStartTag(startTag []byte, attrName, value string, attrRe *regexp.Regexp) []byte {
	replacement := []byte(fmt.Sprintf(` %s="%s"`, attrName, escapeXMLAttributeValue(value)))
	if attrRe.Match(startTag) {
		return attrRe.ReplaceAllLiteral(startTag, replacement)
	}

	insertPos := len(startTag) - 1
	if insertPos < 0 || startTag[insertPos] != '>' {
		return startTag
	}
	if insertPos > 0 && startTag[insertPos-1] == '/' {
		insertPos--
	}

	out := make([]byte, 0, len(startTag)+len(replacement))
	out = append(out, startTag[:insertPos]...)
	out = append(out, replacement...)
	out = append(out, startTag[insertPos:]...)
	return out
}
