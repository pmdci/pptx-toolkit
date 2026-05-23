package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadFontSchemes(t *testing.T) {
	skipIfNoFixture(t)

	schemes, err := ReadFontSchemes(testPPTX, nil)
	if err != nil {
		t.Fatalf("ReadFontSchemes failed: %v", err)
	}

	if len(schemes) != 5 {
		t.Fatalf("expected 5 theme font schemes, got %d", len(schemes))
	}

	expected := map[string]struct {
		themeName  string
		schemeName string
		major      string
		minor      string
	}{
		"theme1.xml": {themeName: "Office Theme Deck", schemeName: "Office", major: "Aptos Display", minor: "Aptos"},
		"theme2.xml": {themeName: "Blue II Deck", schemeName: "Office", major: "Aptos Display", minor: "Aptos"},
		"theme3.xml": {themeName: "Custom Theme Deck", schemeName: "Custom Fonts", major: "Space Grotesk", minor: "Inter"},
		"theme4.xml": {themeName: "Office Theme", schemeName: "Office", major: "Aptos Display", minor: "Aptos"},
		"theme5.xml": {themeName: "Office Theme", schemeName: "Office", major: "Aptos Display", minor: "Aptos"},
	}

	for _, scheme := range schemes {
		want, ok := expected[scheme.FileName]
		if !ok {
			t.Fatalf("unexpected theme file %q", scheme.FileName)
		}
		if scheme.ThemeName != want.themeName {
			t.Fatalf("%s: theme name = %q, want %q", scheme.FileName, scheme.ThemeName, want.themeName)
		}
		if scheme.SchemeName != want.schemeName {
			t.Fatalf("%s: scheme name = %q, want %q", scheme.FileName, scheme.SchemeName, want.schemeName)
		}
		if scheme.MajorTypeface != want.major {
			t.Fatalf("%s: major typeface = %q, want %q", scheme.FileName, scheme.MajorTypeface, want.major)
		}
		if scheme.MinorTypeface != want.minor {
			t.Fatalf("%s: minor typeface = %q, want %q", scheme.FileName, scheme.MinorTypeface, want.minor)
		}
	}
}

func TestSetFontScheme_MajorOnly(t *testing.T) {
	input := []byte(sampleThemeFontXML())

	output, err := SetFontScheme(input, FontSchemeUpdate{Major: "Arial"})
	if err != nil {
		t.Fatalf("SetFontScheme failed: %v", err)
	}

	assertContains(t, output, `majorFont><a:latin typeface="Arial" panose="02110004020202020204"/>`)
	assertContains(t, output, `minorFont><a:latin typeface="Aptos" panose="02110004020202020204"/>`)
	assertContains(t, output, `<a:font script="Jpan" typeface="游ゴシック Light"/>`)
}

func TestSetFontScheme_MinorOnly(t *testing.T) {
	input := []byte(sampleThemeFontXML())

	output, err := SetFontScheme(input, FontSchemeUpdate{Minor: "Times New Roman"})
	if err != nil {
		t.Fatalf("SetFontScheme failed: %v", err)
	}

	assertContains(t, output, `majorFont><a:latin typeface="Aptos Display" panose="02110004020202020204"/>`)
	assertContains(t, output, `minorFont><a:latin typeface="Times New Roman" panose="02110004020202020204"/>`)
}

func TestSetFontScheme_Both(t *testing.T) {
	input := []byte(sampleThemeFontXML())

	output, err := SetFontScheme(input, FontSchemeUpdate{
		Major: "Arial",
		Minor: "Times New Roman",
	})
	if err != nil {
		t.Fatalf("SetFontScheme failed: %v", err)
	}

	assertContains(t, output, `majorFont><a:latin typeface="Arial" panose="02110004020202020204"/>`)
	assertContains(t, output, `minorFont><a:latin typeface="Times New Roman" panose="02110004020202020204"/>`)
}

func TestSetFontScheme_SchemeName(t *testing.T) {
	input := []byte(sampleThemeFontXML())

	output, err := SetFontScheme(input, FontSchemeUpdate{SchemeName: "Corporate"})
	if err != nil {
		t.Fatalf("SetFontScheme failed: %v", err)
	}

	assertContains(t, output, `<a:fontScheme name="Corporate">`)
	assertContains(t, output, `majorFont><a:latin typeface="Aptos Display" panose="02110004020202020204"/>`)
	assertContains(t, output, `minorFont><a:latin typeface="Aptos" panose="02110004020202020204"/>`)
}

func TestSetFontScheme_Preserves47ScriptOverrides(t *testing.T) {
	input := []byte(sampleThemeFontXML())
	before := bytes.Count(input, []byte(`<a:font script=`))
	if before != 47 {
		t.Fatalf("fixture should contain 47 script overrides, got %d", before)
	}

	output, err := SetFontScheme(input, FontSchemeUpdate{Major: "Arial"})
	if err != nil {
		t.Fatalf("SetFontScheme failed: %v", err)
	}

	after := bytes.Count(output, []byte(`<a:font script=`))
	if after != 47 {
		t.Fatalf("expected 47 script overrides after update, got %d", after)
	}
}

func TestSetFontScheme_PreservesPanose(t *testing.T) {
	input := []byte(sampleThemeFontXML())

	output, err := SetFontScheme(input, FontSchemeUpdate{Minor: "Times New Roman"})
	if err != nil {
		t.Fatalf("SetFontScheme failed: %v", err)
	}

	assertContains(t, output, `majorFont><a:latin typeface="Aptos Display" panose="02110004020202020204"/>`)
	assertContains(t, output, `minorFont><a:latin typeface="Times New Roman" panose="02110004020202020204"/>`)
}

func TestSetFontScheme_DollarSignInName(t *testing.T) {
	input := []byte(sampleThemeFontXML())

	tricky := []string{"Foo$1Bar", "$$", "Times${New}Roman"}
	for _, name := range tricky {
		output, err := SetFontScheme(input, FontSchemeUpdate{Major: name})
		if err != nil {
			t.Fatalf("SetFontScheme failed for %q: %v", name, err)
		}
		assertContains(t, output, `typeface="`+name+`"`)
	}
}

func sampleThemeFontXML() string {
	overrides := []string{
		`<a:font script="Jpan" typeface="游ゴシック Light"/>`,
		`<a:font script="Hang" typeface="맑은 고딕"/>`,
		`<a:font script="Hans" typeface="等线 Light"/>`,
		`<a:font script="Hant" typeface="新細明體"/>`,
		`<a:font script="Arab" typeface="Times New Roman"/>`,
		`<a:font script="Hebr" typeface="Times New Roman"/>`,
		`<a:font script="Thai" typeface="Angsana New"/>`,
		`<a:font script="Ethi" typeface="Nyala"/>`,
		`<a:font script="Beng" typeface="Vrinda"/>`,
		`<a:font script="Gujr" typeface="Shruti"/>`,
		`<a:font script="Khmr" typeface="MoolBoran"/>`,
		`<a:font script="Knda" typeface="Tunga"/>`,
		`<a:font script="Guru" typeface="Raavi"/>`,
		`<a:font script="Cans" typeface="Euphemia"/>`,
		`<a:font script="Cher" typeface="Plantagenet Cherokee"/>`,
		`<a:font script="Yiii" typeface="Microsoft Yi Baiti"/>`,
		`<a:font script="Tibt" typeface="Microsoft Himalaya"/>`,
		`<a:font script="Thaa" typeface="MV Boli"/>`,
		`<a:font script="Deva" typeface="Mangal"/>`,
		`<a:font script="Telu" typeface="Gautami"/>`,
		`<a:font script="Taml" typeface="Latha"/>`,
		`<a:font script="Syrc" typeface="Estrangelo Edessa"/>`,
		`<a:font script="Orya" typeface="Kalinga"/>`,
		`<a:font script="Mlym" typeface="Kartika"/>`,
		`<a:font script="Laoo" typeface="DokChampa"/>`,
		`<a:font script="Sinh" typeface="Iskoola Pota"/>`,
		`<a:font script="Mong" typeface="Mongolian Baiti"/>`,
		`<a:font script="Viet" typeface="Times New Roman"/>`,
		`<a:font script="Uigh" typeface="Microsoft Uighur"/>`,
		`<a:font script="Geor" typeface="Sylfaen"/>`,
		`<a:font script="Armn" typeface="Arial"/>`,
		`<a:font script="Bugi" typeface="Leelawadee UI"/>`,
		`<a:font script="Bopo" typeface="Microsoft JhengHei"/>`,
		`<a:font script="Java" typeface="Javanese Text"/>`,
		`<a:font script="Lisu" typeface="Segoe UI"/>`,
		`<a:font script="Mymr" typeface="Myanmar Text"/>`,
		`<a:font script="Nkoo" typeface="Ebrima"/>`,
		`<a:font script="Olck" typeface="Nirmala UI"/>`,
		`<a:font script="Osma" typeface="Ebrima"/>`,
		`<a:font script="Phag" typeface="Phagspa"/>`,
		`<a:font script="Syrn" typeface="Estrangelo Edessa"/>`,
		`<a:font script="Syrj" typeface="Estrangelo Edessa"/>`,
		`<a:font script="Syre" typeface="Estrangelo Edessa"/>`,
		`<a:font script="Sora" typeface="Nirmala UI"/>`,
		`<a:font script="Tale" typeface="Microsoft Tai Le"/>`,
		`<a:font script="Talu" typeface="Microsoft New Tai Lue"/>`,
		`<a:font script="Tfng" typeface="Ebrima"/>`,
	}

	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="Office Theme Deck">` +
		`<a:themeElements>` +
		`<a:clrScheme name="Office"></a:clrScheme>` +
		`<a:fontScheme name="Office">` +
		`<a:majorFont><a:latin typeface="Aptos Display" panose="02110004020202020204"/><a:ea typeface=""/><a:cs typeface=""/>` +
		strings.Join(overrides, "") +
		`</a:majorFont>` +
		`<a:minorFont><a:latin typeface="Aptos" panose="02110004020202020204"/><a:ea typeface=""/><a:cs typeface=""/></a:minorFont>` +
		`</a:fontScheme>` +
		`<a:fmtScheme name="Office"></a:fmtScheme>` +
		`</a:themeElements>` +
		`</a:theme>`
}

func assertContains(t *testing.T, body []byte, needle string) {
	t.Helper()
	if !bytes.Contains(body, []byte(needle)) {
		t.Fatalf("expected output to contain %q, got:\n%s", needle, body)
	}
}
