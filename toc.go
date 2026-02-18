package epub

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

// parseTOC determines the TOC source (nav document or NCX), parses it,
// assigns spine indices, and stores results in b.toc and b.landmarks.
// This is called during initBook after the OPF has been parsed.
func (b *Book) parseTOC() {
	// Build a map from file path (without fragment) → spine index.
	spineMap := make(map[string]int, len(b.spine))
	for i, si := range b.spine {
		// Resolve spine item href relative to OPF directory to get ZIP-internal path.
		href := b.resolveOPFPath(si.Href)
		spineMap[href] = i
	}

	isEPub3 := strings.HasPrefix(b.opf.Version, "3")

	spineLen := len(b.spine)

	if isEPub3 {
		// ePub 3: prefer nav document, fall back to NCX.
		if toc, landmarks, ok := b.parseNavTOC(spineMap); ok {
			b.toc = toc
			b.landmarks = landmarks
			computeSpineRanges(b.toc, spineLen)
			return
		}
	}

	// ePub 2 or ePub 3 without nav document: use NCX.
	if toc, ok := b.parseNCXTOC(spineMap); ok {
		b.toc = toc
		computeSpineRanges(b.toc, spineLen)
		return
	}

	// No TOC found — expose empty TOC/landmarks slices to callers.
	b.toc = []TOCItem{}
	b.landmarks = nil
}

// parseNavTOC finds and parses the nav document, assigns spine indices,
// and returns (toc, landmarks, true). Returns (nil, nil, false) if no nav document is found.
func (b *Book) parseNavTOC(spineMap map[string]int) ([]TOCItem, []TOCItem, bool) {
	// Find the manifest item with properties containing "nav".
	// Iterate the OPF slice (not the map) to get deterministic document order.
	var navItem *manifestItem
	for _, raw := range b.opf.Manifest.Items {
		for _, prop := range strings.Fields(raw.Properties) {
			if prop == "nav" {
				navItem = b.manifestByID[raw.ID]
				break
			}
		}
		if navItem != nil {
			break
		}
	}
	if navItem == nil {
		return nil, nil, false
	}

	// Resolve nav document path relative to OPF directory.
	navPath := b.resolveOPFPath(navItem.Href)

	f := b.findFile(navPath)
	if f == nil {
		return nil, nil, false
	}

	data, err := readZipFile(f)
	if err != nil {
		b.warnings = append(b.warnings, fmt.Sprintf("failed to read nav document: %v", err))
		return nil, nil, false
	}

	toc, landmarks, err := parseNavDocument(data, navPath)
	if err != nil {
		b.warnings = append(b.warnings, fmt.Sprintf("failed to parse nav document: %v", err))
		return nil, nil, false
	}

	assignSpineIndices(toc, spineMap)
	assignSpineIndices(landmarks, spineMap)

	return toc, landmarks, true
}

// parseNCXTOC finds and parses the NCX file, assigns spine indices,
// and returns (toc, true). Returns (nil, false) if no NCX is found.
func (b *Book) parseNCXTOC(spineMap map[string]int) ([]TOCItem, bool) {
	tocID := b.opf.Spine.Toc
	if tocID == "" {
		return nil, false
	}

	ncxItem, ok := b.manifestByID[tocID]
	if !ok {
		return nil, false
	}

	// Resolve NCX path relative to OPF directory.
	ncxPath := b.resolveOPFPath(ncxItem.Href)

	f := b.findFile(ncxPath)
	if f == nil {
		return nil, false
	}

	data, err := readZipFile(f)
	if err != nil {
		b.warnings = append(b.warnings, fmt.Sprintf("failed to read NCX file: %v", err))
		return nil, false
	}

	toc, err := parseNCX(data, ncxPath)
	if err != nil {
		b.warnings = append(b.warnings, fmt.Sprintf("failed to parse NCX file: %v", err))
		return nil, false
	}

	assignSpineIndices(toc, spineMap)

	return toc, true
}

// assignSpineIndices recursively sets SpineIndex on each TOCItem by matching
// its Href (without fragment) against the spine map.
func assignSpineIndices(items []TOCItem, spineMap map[string]int) {
	for i := range items {
		if items[i].Href != "" {
			filePath := hrefWithoutFragment(items[i].Href)
			if idx, ok := spineMap[filePath]; ok {
				items[i].SpineIndex = idx
			}
		}
		if len(items[i].Children) > 0 {
			assignSpineIndices(items[i].Children, spineMap)
		}
	}
}

// hrefWithoutFragment returns the href with the fragment (#...) removed.
func hrefWithoutFragment(href string) string {
	if idx := strings.IndexByte(href, '#'); idx >= 0 {
		return href[:idx]
	}
	return href
}

// computeSpineRanges sets SpineEndIndex on each TOCItem so that the entry
// covers spine[SpineIndex:SpineEndIndex]. Items with SpineIndex == -1 get
// SpineEndIndex == -1. For the last entry (by SpineIndex order), SpineEndIndex
// equals spineLen.
func computeSpineRanges(items []TOCItem, spineLen int) {
	if len(items) == 0 {
		return
	}

	// Flatten all TOC items into a slice of pointers.
	var flat []*TOCItem
	flattenTOCItems(&flat, items)

	// Collect unique spine indices.
	seen := make(map[int]bool, len(flat))
	var indices []int
	for _, item := range flat {
		if item.SpineIndex >= 0 && !seen[item.SpineIndex] {
			seen[item.SpineIndex] = true
			indices = append(indices, item.SpineIndex)
		}
	}

	if len(indices) == 0 {
		return
	}

	sort.Ints(indices)

	// Build mapping: SpineIndex → SpineEndIndex.
	endMap := make(map[int]int, len(indices))
	for i, idx := range indices {
		if i+1 < len(indices) {
			endMap[idx] = indices[i+1]
		} else {
			endMap[idx] = spineLen
		}
	}

	// Apply SpineEndIndex to all items.
	for _, item := range flat {
		if item.SpineIndex >= 0 {
			item.SpineEndIndex = endMap[item.SpineIndex]
		} else {
			item.SpineEndIndex = -1
		}
	}
}

// flattenTOCItems collects pointers to all TOCItem nodes (including nested
// children) into flat.
func flattenTOCItems(flat *[]*TOCItem, items []TOCItem) {
	for i := range items {
		*flat = append(*flat, &items[i])
		if len(items[i].Children) > 0 {
			flattenTOCItems(flat, items[i].Children)
		}
	}
}

// --- NCX XML decoding structs (ePub 2) ---

// ncxDocument represents the root <ncx> element of an NCX file.
type ncxDocument struct {
	XMLName xml.Name  `xml:"ncx"`
	NavMap  ncxNavMap `xml:"navMap"`
}

// ncxNavMap represents the <navMap> element containing top-level navPoints.
type ncxNavMap struct {
	NavPoints []ncxNavPoint `xml:"navPoint"`
}

// ncxNavPoint represents a <navPoint> element which may contain nested navPoints.
type ncxNavPoint struct {
	ID        string        `xml:"id,attr"`
	PlayOrder string        `xml:"playOrder,attr"`
	Label     ncxNavLabel   `xml:"navLabel"`
	Content   ncxContent    `xml:"content"`
	Children  []ncxNavPoint `xml:"navPoint"`
}

// ncxNavLabel represents the <navLabel> element containing the display text.
type ncxNavLabel struct {
	Text string `xml:"text"`
}

// ncxContent represents the <content> element with its src attribute.
type ncxContent struct {
	Src string `xml:"src,attr"`
}

// parseNCX parses NCX (ePub 2) data and returns a tree of TOCItem.
// ncxPath is the ZIP-internal path to the NCX file (e.g., "OEBPS/toc.ncx"),
// used to resolve relative hrefs to ZIP root-relative paths.
func parseNCX(data []byte, ncxPath string) ([]TOCItem, error) {
	data = preprocessHTMLEntities(data)
	data = stripBOM(data)

	var doc ncxDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("epub: parse NCX: %w", err)
	}

	items := convertNavPoints(doc.NavMap.NavPoints, ncxPath)
	return items, nil
}

// convertNavPoints recursively converts ncxNavPoint elements into TOCItem entries.
func convertNavPoints(points []ncxNavPoint, ncxPath string) []TOCItem {
	if len(points) == 0 {
		return nil
	}

	items := make([]TOCItem, 0, len(points))
	for _, np := range points {
		item := TOCItem{
			Title:         strings.TrimSpace(np.Label.Text),
			SpineIndex:    -1,
			SpineEndIndex: -1,
		}

		// Resolve href relative to the NCX file location.
		src := strings.TrimSpace(np.Content.Src)
		if src != "" {
			if resolved := resolveRelativePath(ncxPath, src); resolved != "" {
				item.Href = resolved
			}
		}

		// Recursively process nested navPoints.
		item.Children = convertNavPoints(np.Children, ncxPath)

		items = append(items, item)
	}

	return items
}

// --- Nav Document parsing (ePub 3) ---

// parseNavDocument parses an ePub 3 XHTML nav document and returns toc and landmarks.
// basePath is the ZIP-internal path of the nav document file (for resolving relative hrefs).
func parseNavDocument(data []byte, basePath string) (toc []TOCItem, landmarks []TOCItem, err error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("epub: parse nav document: %w", err)
	}

	// Collect all <nav> elements from the document.
	var navNodes []*html.Node
	var findNavs func(*html.Node)
	findNavs = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "nav" {
			navNodes = append(navNodes, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findNavs(c)
		}
	}
	findNavs(doc)

	for _, nav := range navNodes {
		if hasEpubType(nav, "toc") {
			if ol := findFirstChildElement(nav, "ol"); ol != nil {
				toc = parseNavOL(ol, basePath)
			}
		} else if hasEpubType(nav, "landmarks") {
			if ol := findFirstChildElement(nav, "ol"); ol != nil {
				landmarks = parseNavOL(ol, basePath)
			}
		}
	}

	return toc, landmarks, nil
}

// parseNavOL processes an <ol> element and returns its <li> children as TOCItem entries.
func parseNavOL(ol *html.Node, basePath string) []TOCItem {
	var items []TOCItem
	for c := ol.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "li" {
			item := parseNavLI(c, basePath)
			items = append(items, item)
		}
	}
	return items
}

// parseNavLI processes a single <li> element.
// It looks for <a> (or <span> fallback) for title/href and nested <ol> for children.
func parseNavLI(li *html.Node, basePath string) TOCItem {
	item := TOCItem{SpineIndex: -1, SpineEndIndex: -1}

	for c := li.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		switch c.Data {
		case "a":
			// Keep the first <a> (per ePub 3 nav spec, each <li> has exactly one).
			if item.Href == "" {
				href := navGetAttr(c, "href")
				if href != "" {
					if resolved := resolveRelativePath(basePath, href); resolved != "" {
						item.Href = resolved
					}
				}
				item.Title = strings.TrimSpace(nodeTextContent(c))
			}
		case "span":
			// Use <span> text only if no <a> has been found yet.
			if item.Title == "" {
				item.Title = strings.TrimSpace(nodeTextContent(c))
			}
		case "ol":
			item.Children = parseNavOL(c, basePath)
		}
	}

	return item
}

// hasEpubType checks whether n has an epub:type attribute containing the given token
// (space-separated token matching).
func hasEpubType(n *html.Node, typeName string) bool {
	val := navGetAttr(n, "epub:type")
	for _, t := range strings.Fields(val) {
		if t == typeName {
			return true
		}
	}
	return false
}

// navGetAttr returns the value of the attribute with the given key on n.
func navGetAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// findFirstChildElement performs a depth-first search for the first descendant
// element with the given tag name.
func findFirstChildElement(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
		if found := findFirstChildElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

// nodeTextContent recursively collects all text content within a node.
func nodeTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(nodeTextContent(c))
	}
	return sb.String()
}
