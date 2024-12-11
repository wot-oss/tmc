package model

import (
	"encoding/json"
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
	FoundIn      FoundSource
	AttachmentContainer
}
type FoundVersion struct {
	*IndexVersion
	FoundIn FoundSource
}

type FoundAttachment struct {
	Attachment
	FoundIn FoundSource `json:"repo"`
}

type FoundSource struct {
	Directory string
	RepoName  string
}

func (s FoundSource) String() string {
	if s.Directory != "" {
		return "<" + s.Directory + ">"
	}
	return s.RepoName
}

func (s FoundSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func MergeFoundVersions(vs1, vs2 []FoundVersion) []FoundVersion {
	vs1 = append(vs1, vs2...)
	slices.SortStableFunc(vs1, func(a, b FoundVersion) int {
		tmid1, _ := ParseTMID(a.TMID)
		tmid2, _ := ParseTMID(b.TMID)
		nc := strings.Compare(tmid1.Name, tmid2.Name)
		if nc != 0 {
			return nc
		}
		vc := -tmid1.Version.Compare(tmid2.Version) // sort in descending order within the same TM name
		if vc != 0 {
			return vc
		}
		return strings.Compare(a.FoundIn.RepoName, b.FoundIn.RepoName)
	})
	return vs1
}

func (sr *SearchResult) Merge(other *SearchResult) {
	sr.Entries = mergeFoundEntries(sr.Entries, other.Entries)
}

func mergeFoundEntries(e1, e2 []FoundEntry) []FoundEntry {
	e1 = append(e1, e2...)
	slices.SortStableFunc(e1, func(a, b FoundEntry) int {
		nc := strings.Compare(a.Name, b.Name)
		if nc != 0 {
			return nc
		}
		return strings.Compare(a.FoundIn.RepoName, b.FoundIn.RepoName)
	})
	return e1
}

type SearchParams struct {
	Author       []string
	Manufacturer []string
	Mpn          []string
	Protocol     []string
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

func ToSearchParams(author, manufacturer, mpn, protocol, name, query *string, opts *SearchOptions) *SearchParams {
	var search *SearchParams
	isSet := func(s *string) bool { return s != nil && *s != "" }
	if isSet(author) || isSet(manufacturer) || isSet(mpn) || isSet(protocol) || isSet(name) || isSet(query) {
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
		if isSet(protocol) {
			search.Protocol = strings.Split(*protocol, DefaultListSeparator)
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
