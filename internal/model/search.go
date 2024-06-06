package model

import (
	"slices"
	"strings"

	"github.com/wot-oss/tmc/internal/utils"
)

const (
	FullMatch FilterType = iota
	PrefixMatch
)
const DefaultListSeparator = ","

type SearchResult struct {
	Entries []FoundEntry
}
type FoundEntry struct {
	Name         string
	Manufacturer SchemaManufacturer
	Mpn          string
	Author       SchemaAuthor
	Versions     []FoundVersion
	AttachmentContainer
}
type FoundVersion struct {
	IndexVersion
	FoundIn FoundSource
}

type FoundSource struct {
	Directory string
	RepoName  string
}

func (s FoundSource) String() string {
	if s.Directory != "" {
		return s.Directory
	}
	return "<" + s.RepoName + ">"
}

func MergeFoundVersions(vs1, vs2 []FoundVersion) []FoundVersion {
	vs1 = append(vs1, vs2...)
	slices.SortStableFunc(vs1, func(a, b FoundVersion) int {
		tmid1, _ := ParseTMID(a.TMID)
		tmid2, _ := ParseTMID(b.TMID)
		if tmid1.Equals(tmid2) {
			return -strings.Compare(tmid1.Version.Timestamp, tmid2.Version.Timestamp) // sort in reverse chronological order within the same TMID
		}
		return strings.Compare(a.TMID, b.TMID)
	})
	return slices.CompactFunc(vs1, func(v1, v2 FoundVersion) bool {
		tmid1, _ := ParseTMID(v1.TMID)
		tmid2, _ := ParseTMID(v2.TMID)
		return tmid1.Equals(tmid2)
	})
}

func (r FoundEntry) Merge(other FoundEntry) FoundEntry {
	if r.Name == "" {
		return FoundEntry{
			Name:         other.Name,
			Manufacturer: other.Manufacturer,
			Mpn:          other.Mpn,
			Author:       other.Author,
			Versions:     other.Versions,
		}
	}
	r.Versions = MergeFoundVersions(r.Versions, other.Versions)
	return r
}

func (sr *SearchResult) Merge(other *SearchResult) {
	sr.Entries = mergeFoundEntries(sr.Entries, other.Entries)
}

func mergeFoundEntries(e1, e2 []FoundEntry) []FoundEntry {
	e1 = append(e1, e2...)
	slices.SortStableFunc(e1, func(a, b FoundEntry) int {
		return strings.Compare(a.Name, b.Name)
	})
	if len(e1) < 2 {
		return e1
	}
	i := 1
	for k := 1; k < len(e1); k++ {
		if e1[k].Name != e1[k-1].Name {
			if i != k {
				e1[i] = e1[k]
			}
			i++
		} else {
			e1[k-1] = e1[k-1].Merge(e1[k])
		}
	}
	return e1[:i]
}

type SearchParams struct {
	Author       []string
	Manufacturer []string
	Mpn          []string
	Name         string
	Query        string
	Options      SearchOptions
}

func (p *SearchParams) Sanitize() {
	p.Author = sanitizeList(p.Author)
	p.Manufacturer = sanitizeList(p.Manufacturer)
	p.Mpn = sanitizeList(p.Mpn)
}

type FilterType byte

type SearchOptions struct {
	// NameFilterType specifies whether SearchParams.Name must match a prefix or the full length of a TM name
	// Note that using FullMatch effectively limits the search result to at most one FoundEntry
	NameFilterType FilterType
}

func ToSearchParams(author, manufacturer, mpn, name, query *string, opts *SearchOptions) *SearchParams {
	var search *SearchParams
	isSet := func(s *string) bool { return s != nil && *s != "" }
	if isSet(author) || isSet(manufacturer) || isSet(mpn) || isSet(name) || isSet(query) {
		search = &SearchParams{}
		if isSet(author) {
			search.Author = strings.Split(*author, DefaultListSeparator)
		}
		if isSet(manufacturer) {
			search.Manufacturer = strings.Split(*manufacturer, DefaultListSeparator)
		}
		if isSet(mpn) {
			search.Mpn = strings.Split(*mpn, DefaultListSeparator)
		}
		if isSet(query) {
			search.Query = *query
		}
		if isSet(name) {
			search.Name = *name
		}
		if opts != nil {
			search.Options = *opts
		}
	}
	return search
}

func sanitizeList(l []string) []string {
	for i, v := range l {
		l[i] = utils.SanitizeName(v)
	}
	return l
}
