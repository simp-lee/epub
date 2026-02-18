package epub

import (
	"sort"
	"strconv"
	"strings"
)

// extractMetadata converts the raw OPF metadata into the public Metadata struct.
func extractMetadata(opf *opfPackage) Metadata {
	md := Metadata{
		Version: opf.Version,
	}
	om := &opf.Metadata

	// Build a refines lookup for ePub 3: "#id" → []opfMeta.
	refinesMap := buildRefinesMap(om.Metas)

	// Titles.
	md.Titles = extractTitles(om.Titles, refinesMap)

	// Authors (dc:creator).
	md.Authors = extractAuthors(om.Creators, refinesMap)

	// Languages.
	for _, l := range om.Languages {
		if v := strings.TrimSpace(l.Value); v != "" {
			md.Language = append(md.Language, v)
		}
	}

	// Identifiers.
	for _, id := range om.Identifiers {
		v := strings.TrimSpace(id.Value)
		if v == "" {
			continue
		}
		ident := Identifier{
			Value:  v,
			Scheme: id.Scheme,
			ID:     id.ID,
		}
		// ePub 3: check refines for scheme.
		if ident.Scheme == "" && id.ID != "" {
			if s, ok := findRefine(refinesMap, id.ID, "identifier-type"); ok {
				ident.Scheme = s
			}
		}
		md.Identifiers = append(md.Identifiers, ident)
	}

	// Publisher — take first non-empty.
	for _, p := range om.Publishers {
		if v := strings.TrimSpace(p.Value); v != "" {
			md.Publisher = v
			break
		}
	}

	// Date — take first non-empty.
	for _, d := range om.Dates {
		if v := strings.TrimSpace(d.Value); v != "" {
			md.Date = v
			break
		}
	}

	// Description — take first non-empty.
	for _, d := range om.Descriptions {
		if v := strings.TrimSpace(d.Value); v != "" {
			md.Description = v
			break
		}
	}

	// Subjects.
	for _, s := range om.Subjects {
		if v := strings.TrimSpace(s.Value); v != "" {
			md.Subjects = append(md.Subjects, v)
		}
	}

	// Rights — take first non-empty.
	for _, r := range om.Rights {
		if v := strings.TrimSpace(r.Value); v != "" {
			md.Rights = v
			break
		}
	}

	// Source — take first non-empty.
	for _, s := range om.Sources {
		if v := strings.TrimSpace(s.Value); v != "" {
			md.Source = v
			break
		}
	}

	return md
}

// buildRefinesMap builds a map from element ID (without "#") to the list of
// <meta refines="#id" ...> elements that refine it.
func buildRefinesMap(metas []opfMeta) map[string][]opfMeta {
	m := make(map[string][]opfMeta)
	for _, meta := range metas {
		ref := meta.Refines
		if ref == "" || !strings.HasPrefix(ref, "#") {
			continue
		}
		id := ref[1:] // strip leading "#"
		m[id] = append(m[id], meta)
	}
	return m
}

// findRefine looks up a single refining property value for the given element ID.
func findRefine(refinesMap map[string][]opfMeta, id, property string) (string, bool) {
	for _, m := range refinesMap[id] {
		if m.Property == property {
			v := strings.TrimSpace(m.Value)
			if v != "" {
				return v, true
			}
		}
	}
	return "", false
}

// extractTitles extracts titles from dc:title elements.
// For ePub 3, titles are ordered by display-seq from refines metadata.
func extractTitles(titles []opfDCElement, refinesMap map[string][]opfMeta) []string {
	if len(titles) == 0 {
		return nil
	}

	type titleEntry struct {
		value string
		seq   int
		index int // original order
	}

	entries := make([]titleEntry, 0, len(titles))
	hasSeq := false

	for i, t := range titles {
		v := strings.TrimSpace(t.Value)
		if v == "" {
			continue
		}
		e := titleEntry{value: v, seq: 0, index: i}
		if t.ID != "" {
			if seqStr, ok := findRefine(refinesMap, t.ID, "display-seq"); ok {
				if n, err := strconv.Atoi(seqStr); err == nil {
					e.seq = n
					hasSeq = true
				}
			}
		}
		entries = append(entries, e)
	}

	// Sort by display-seq if any title has one; otherwise preserve original order.
	if hasSeq {
		sort.SliceStable(entries, func(i, j int) bool {
			// Titles without seq (0) go after titles with seq.
			si, sj := entries[i].seq, entries[j].seq
			if si == 0 && sj == 0 {
				return entries[i].index < entries[j].index
			}
			if si == 0 {
				return false
			}
			if sj == 0 {
				return true
			}
			return si < sj
		})
	}

	result := make([]string, len(entries))
	for i, e := range entries {
		result[i] = e.value
	}
	return result
}

// extractAuthors extracts author information from dc:creator elements.
// ePub 2: uses opf:file-as and opf:role attributes directly on the element.
// ePub 3: uses <meta refines="..."> elements to express file-as and role.
func extractAuthors(creators []opfDCElement, refinesMap map[string][]opfMeta) []Author {
	if len(creators) == 0 {
		return nil
	}

	authors := make([]Author, 0, len(creators))
	for _, c := range creators {
		name := strings.TrimSpace(c.Value)
		if name == "" {
			continue
		}

		a := Author{
			Name:   name,
			FileAs: c.FileAs,
			Role:   c.Role,
		}

		// ePub 3: check refines for file-as and role if not set via attributes.
		if c.ID != "" {
			if a.FileAs == "" {
				if fa, ok := findRefine(refinesMap, c.ID, "file-as"); ok {
					a.FileAs = fa
				}
			}
			if a.Role == "" {
				if r, ok := findRefine(refinesMap, c.ID, "role"); ok {
					a.Role = r
				}
			}
		}

		authors = append(authors, a)
	}
	return authors
}
