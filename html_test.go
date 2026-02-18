package epub

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// preprocessHTMLEntities tests
// ---------------------------------------------------------------------------

func TestPreprocessHTMLEntities_BasicReplacements(t *testing.T) {
	input := []byte(`<title>Hello&nbsp;World &mdash; An&hellip; Introduction</title>`)
	got := preprocessHTMLEntities(input)
	want := `<title>Hello&#160;World &#8212; An&#8230; Introduction</title>`
	if string(got) != want {
		t.Errorf("preprocessHTMLEntities():\n got: %s\nwant: %s", got, want)
	}
}

func TestPreprocessHTMLEntities_QuotationMarks(t *testing.T) {
	input := []byte(`&ldquo;Hello&rdquo; &lsquo;World&rsquo;`)
	got := preprocessHTMLEntities(input)
	want := `&#8220;Hello&#8221; &#8216;World&#8217;`
	if string(got) != want {
		t.Errorf("preprocessHTMLEntities():\n got: %s\nwant: %s", got, want)
	}
}

func TestPreprocessHTMLEntities_Symbols(t *testing.T) {
	input := []byte(`&copy; 2024 &reg; Company&trade; &bull; Item &middot; Sub`)
	got := preprocessHTMLEntities(input)
	want := `&#169; 2024 &#174; Company&#8482; &#8226; Item &#183; Sub`
	if string(got) != want {
		t.Errorf("preprocessHTMLEntities():\n got: %s\nwant: %s", got, want)
	}
}

func TestPreprocessHTMLEntities_AccentedChars(t *testing.T) {
	input := []byte(`caf&eacute; na&iuml;ve r&eacute;sum&eacute;`)
	got := preprocessHTMLEntities(input)
	want := `caf&#233; na&#239;ve r&#233;sum&#233;`
	if string(got) != want {
		t.Errorf("preprocessHTMLEntities():\n got: %s\nwant: %s", got, want)
	}
}

func TestPreprocessHTMLEntities_PreservesXMLEntities(t *testing.T) {
	// &amp;, &lt;, &gt;, &quot;, &apos; are valid XML entities and must be preserved.
	input := []byte(`&amp; &lt; &gt; &quot; &apos;`)
	got := preprocessHTMLEntities(input)
	if string(got) != string(input) {
		t.Errorf("XML entities should be preserved:\n got: %s\nwant: %s", got, input)
	}
}

func TestPreprocessHTMLEntities_NoEntities(t *testing.T) {
	input := []byte(`<p>Plain text with no entities</p>`)
	got := preprocessHTMLEntities(input)
	if string(got) != string(input) {
		t.Errorf("Text without entities should be unchanged:\n got: %s\nwant: %s", got, input)
	}
}

func TestPreprocessHTMLEntities_Dashes(t *testing.T) {
	input := []byte(`2020&ndash;2024 &mdash; a range`)
	got := preprocessHTMLEntities(input)
	want := `2020&#8211;2024 &#8212; a range`
	if string(got) != want {
		t.Errorf("preprocessHTMLEntities():\n got: %s\nwant: %s", got, want)
	}
}

// ---------------------------------------------------------------------------
// extractText tests
// ---------------------------------------------------------------------------

func TestExtractText_SimpleParagraphs(t *testing.T) {
	input := []byte(`<html><body><p>First paragraph.</p><p>Second paragraph.</p></body></html>`)
	got, err := extractText(input)
	if err != nil {
		t.Fatalf("extractText() error: %v", err)
	}
	want := "First paragraph.\nSecond paragraph."
	if got != want {
		t.Errorf("extractText():\n got: %q\nwant: %q", got, want)
	}
}

func TestExtractText_LineBreaks(t *testing.T) {
	input := []byte(`<html><body><p>Line one<br/>Line two<br>Line three</p></body></html>`)
	got, err := extractText(input)
	if err != nil {
		t.Fatalf("extractText() error: %v", err)
	}
	want := "Line one\nLine two\nLine three"
	if got != want {
		t.Errorf("extractText():\n got: %q\nwant: %q", got, want)
	}
}

func TestExtractText_Headings(t *testing.T) {
	input := []byte(`<html><body><h1>Title</h1><p>Content</p><h2>Subtitle</h2><p>More</p></body></html>`)
	got, err := extractText(input)
	if err != nil {
		t.Fatalf("extractText() error: %v", err)
	}
	want := "Title\nContent\nSubtitle\nMore"
	if got != want {
		t.Errorf("extractText():\n got: %q\nwant: %q", got, want)
	}
}

func TestExtractText_SkipScriptAndStyle(t *testing.T) {
	input := []byte(`<html>
<head><style>body { color: red; }</style></head>
<body>
<p>Visible text</p>
<script>alert("hidden");</script>
<p>Also visible</p>
</body></html>`)
	got, err := extractText(input)
	if err != nil {
		t.Fatalf("extractText() error: %v", err)
	}
	if strings.Contains(got, "alert") {
		t.Errorf("script content should be skipped, got: %q", got)
	}
	if strings.Contains(got, "color") {
		t.Errorf("style content should be skipped, got: %q", got)
	}
	if !strings.Contains(got, "Visible text") || !strings.Contains(got, "Also visible") {
		t.Errorf("visible text should be present, got: %q", got)
	}
}

func TestExtractText_SelfClosingScriptAndStyle(t *testing.T) {
	input := []byte(`<html><body><p>Before</p><script/><style/><p>After</p></body></html>`)
	got, err := extractText(input)
	if err != nil {
		t.Fatalf("extractText() error: %v", err)
	}
	want := "Before\nAfter"
	if got != want {
		t.Errorf("extractText():\n got: %q\nwant: %q", got, want)
	}
}

func TestExtractText_DivAndList(t *testing.T) {
	input := []byte(`<html><body><div>Block one</div><div>Block two</div><ul><li>Item A</li><li>Item B</li></ul></body></html>`)
	got, err := extractText(input)
	if err != nil {
		t.Fatalf("extractText() error: %v", err)
	}
	want := "Block one\nBlock two\nItem A\nItem B"
	if got != want {
		t.Errorf("extractText():\n got: %q\nwant: %q", got, want)
	}
}

func TestExtractText_EmptyInput(t *testing.T) {
	got, err := extractText([]byte(""))
	if err != nil {
		t.Fatalf("extractText() error: %v", err)
	}
	if got != "" {
		t.Errorf("extractText(empty) = %q; want empty", got)
	}
}

func TestExtractText_InlineElements(t *testing.T) {
	input := []byte(`<html><body><p>This is <b>bold</b> and <i>italic</i> text.</p></body></html>`)
	got, err := extractText(input)
	if err != nil {
		t.Fatalf("extractText() error: %v", err)
	}
	want := "This is bold and italic text."
	if got != want {
		t.Errorf("extractText():\n got: %q\nwant: %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// extractBodyHTML tests
// ---------------------------------------------------------------------------

func TestExtractBodyHTML_BasicBody(t *testing.T) {
	input := []byte(`<html><head><title>Test</title><style>h1{color:red}</style></head><body><h1>Hello</h1><p>World</p></body></html>`)
	got, err := extractBodyHTML(input)
	if err != nil {
		t.Fatalf("extractBodyHTML() error: %v", err)
	}
	if strings.Contains(got, "<head>") || strings.Contains(got, "<title>") || strings.Contains(got, "<style>") {
		t.Errorf("head/title/style should be stripped, got: %q", got)
	}
	if !strings.Contains(got, "<h1>Hello</h1>") {
		t.Errorf("body content should be preserved, got: %q", got)
	}
	if !strings.Contains(got, "<p>World</p>") {
		t.Errorf("body content should be preserved, got: %q", got)
	}
}

func TestExtractBodyHTML_StripsScriptAndStyle(t *testing.T) {
	input := []byte(`<html><body><p>Keep</p><script>alert("x")</script><style>.hide{display:none}</style><p>Also keep</p></body></html>`)
	got, err := extractBodyHTML(input)
	if err != nil {
		t.Fatalf("extractBodyHTML() error: %v", err)
	}
	if strings.Contains(got, "<script>") || strings.Contains(got, "alert") {
		t.Errorf("script should be stripped, got: %q", got)
	}
	if strings.Contains(got, "<style>") || strings.Contains(got, "display") {
		t.Errorf("style should be stripped, got: %q", got)
	}
	if !strings.Contains(got, "<p>Keep</p>") || !strings.Contains(got, "<p>Also keep</p>") {
		t.Errorf("non-script content should be kept, got: %q", got)
	}
}

func TestExtractBodyHTML_StripsEventAttributes(t *testing.T) {
	input := []byte(`<html><body><div onclick="evil()" onmouseover="track()"><p onload="init()">Text</p></div></body></html>`)
	got, err := extractBodyHTML(input)
	if err != nil {
		t.Fatalf("extractBodyHTML() error: %v", err)
	}
	if strings.Contains(got, "onclick") || strings.Contains(got, "onmouseover") || strings.Contains(got, "onload") {
		t.Errorf("event attributes should be stripped, got: %q", got)
	}
	if !strings.Contains(got, "<p>Text</p>") {
		t.Errorf("text content should be preserved, got: %q", got)
	}
}

func TestExtractBodyHTML_NoBody(t *testing.T) {
	input := []byte(`<html><head><title>No Body</title></head></html>`)
	got, err := extractBodyHTML(input)
	if err != nil {
		t.Fatalf("extractBodyHTML() error: %v", err)
	}
	if got != "" {
		t.Errorf("extractBodyHTML(no body) = %q; want empty", got)
	}
}

func TestExtractBodyHTML_PreservesAttributes(t *testing.T) {
	input := []byte(`<html><body><a href="link.html" class="nav">Click</a></body></html>`)
	got, err := extractBodyHTML(input)
	if err != nil {
		t.Fatalf("extractBodyHTML() error: %v", err)
	}
	if !strings.Contains(got, `href="link.html"`) {
		t.Errorf("non-event attributes should be preserved, got: %q", got)
	}
	if !strings.Contains(got, `class="nav"`) {
		t.Errorf("class attribute should be preserved, got: %q", got)
	}
}

func TestExtractBodyHTML_StripsDangerousURIProtocols(t *testing.T) {
	input := []byte(`<html><body>
		<a href="javascript:alert(1)">Bad JS</a>
		<a href="vbscript:msgbox(1)">Bad VB</a>
		<img src="data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg=="/>
	</body></html>`)

	got, err := extractBodyHTML(input)
	if err != nil {
		t.Fatalf("extractBodyHTML() error: %v", err)
	}

	if strings.Contains(strings.ToLower(got), "javascript:") {
		t.Errorf("javascript: URI should be stripped, got: %q", got)
	}
	if strings.Contains(strings.ToLower(got), "vbscript:") {
		t.Errorf("vbscript: URI should be stripped, got: %q", got)
	}
	if strings.Contains(strings.ToLower(got), "data:text/html") {
		t.Errorf("data:text/html URI should be stripped, got: %q", got)
	}
}

func TestExtractBodyHTML_AllowsSafeURIProtocols(t *testing.T) {
	input := []byte(`<html><body>
		<a href="https://example.com">HTTPS</a>
		<a href="mailto:test@example.com">Mail</a>
		<a href="#section">Fragment</a>
		<a href="chapter1.xhtml">Relative</a>
		<img src="data:image/png;base64,AAA"/>
	</body></html>`)

	got, err := extractBodyHTML(input)
	if err != nil {
		t.Fatalf("extractBodyHTML() error: %v", err)
	}

	checks := []string{
		`href="https://example.com"`,
		`href="mailto:test@example.com"`,
		`href="#section"`,
		`href="chapter1.xhtml"`,
		`src="data:image/png;base64,AAA"`,
	}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("expected safe URI to be preserved (%s), got: %q", want, got)
		}
	}
}

// ---------------------------------------------------------------------------
// rewriteImagePaths tests
// ---------------------------------------------------------------------------

func TestRewriteImagePaths_ImgSrc(t *testing.T) {
	input := []byte(`<html><body><img src="../images/cover.jpg"/></body></html>`)
	got := rewriteImagePaths(input, "OEBPS/text/chapter1.xhtml")
	gotStr := string(got)
	if !strings.Contains(gotStr, `src="OEBPS/images/cover.jpg"`) {
		t.Errorf("img src should be rewritten to absolute path, got: %s", gotStr)
	}
}

func TestRewriteImagePaths_SameDirectory(t *testing.T) {
	input := []byte(`<html><body><img src="image.png"/></body></html>`)
	got := rewriteImagePaths(input, "OEBPS/chapter1.xhtml")
	gotStr := string(got)
	if !strings.Contains(gotStr, `src="OEBPS/image.png"`) {
		t.Errorf("img src should resolve to same directory, got: %s", gotStr)
	}
}

func TestRewriteImagePaths_AbsoluteURLsUnchanged(t *testing.T) {
	input := []byte(`<html><body><img src="https://example.com/img.png"/></body></html>`)
	got := rewriteImagePaths(input, "OEBPS/chapter1.xhtml")
	gotStr := string(got)
	if !strings.Contains(gotStr, `src="https://example.com/img.png"`) {
		t.Errorf("absolute URLs should not be rewritten, got: %s", gotStr)
	}
}

func TestRewriteImagePaths_DataURIsUnchanged(t *testing.T) {
	input := []byte(`<html><body><img src="data:image/png;base64,ABC"/></body></html>`)
	got := rewriteImagePaths(input, "OEBPS/chapter1.xhtml")
	gotStr := string(got)
	if !strings.Contains(gotStr, `src="data:image/png;base64,ABC"`) {
		t.Errorf("data URIs should not be rewritten, got: %s", gotStr)
	}
}

func TestRewriteImagePaths_SVGImage(t *testing.T) {
	input := []byte(`<html><body><svg><image xlink:href="../images/pic.svg"/></svg></body></html>`)
	got := rewriteImagePaths(input, "OEBPS/text/page.xhtml")
	gotStr := string(got)
	if !strings.Contains(gotStr, "OEBPS/images/pic.svg") {
		t.Errorf("SVG image xlink:href should be rewritten, got: %s", gotStr)
	}
}

func TestRewriteImagePaths_MultipleImages(t *testing.T) {
	input := []byte(`<html><body><img src="a.jpg"/><img src="../b.png"/></body></html>`)
	got := rewriteImagePaths(input, "OEBPS/text/ch.xhtml")
	gotStr := string(got)
	if !strings.Contains(gotStr, `src="OEBPS/text/a.jpg"`) {
		t.Errorf("first image should be resolved, got: %s", gotStr)
	}
	if !strings.Contains(gotStr, `src="OEBPS/b.png"`) {
		t.Errorf("second image should be resolved, got: %s", gotStr)
	}
}

func TestRewriteImagePaths_EmptySrc(t *testing.T) {
	input := []byte(`<html><body><img src=""/></body></html>`)
	got := rewriteImagePaths(input, "OEBPS/chapter1.xhtml")
	gotStr := string(got)
	// Empty src should remain empty.
	if !strings.Contains(gotStr, `src=""`) {
		t.Errorf("empty src should remain empty, got: %s", gotStr)
	}
}

func TestRewriteImagePaths_InvalidHTML(t *testing.T) {
	// Badly formed HTML should not cause a panic; input should be returned.
	input := []byte(`<html><body><img src="ok.jpg"`)
	got := rewriteImagePaths(input, "OEBPS/chapter.xhtml")
	// x/net/html is lenient, so it may still parse. Just ensure no panic.
	if len(got) == 0 {
		t.Error("rewriteImagePaths should return non-empty output even for malformed input")
	}
}
