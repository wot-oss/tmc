package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/wot-oss/tmc/internal/utils"
)

const (
	FullMatch FilterType = iota
	PrefixMatch
)
const DefaultListSeparator = ","

var ErrSearchIndexNotFound = errors.New("search index not found. Use `tmc create-si` to create")

type SearchResult struct {
	LastUpdated time.Time
	Entries     []FoundEntry
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
	if other.LastUpdated.After(sr.LastUpdated) {
		sr.LastUpdated = other.LastUpdated
	}
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

type Filters struct {
	Author       []string
	Manufacturer []string
	Mpn          []string
	Protocol     []string
	Name         string
	Options      FilterOptions
}

func (p *Filters) Sanitize() {
	p.Author = sanitizeList(p.Author)
	p.Manufacturer = sanitizeList(p.Manufacturer)
	p.Mpn = sanitizeList(p.Mpn)
}

type FilterType byte

type FilterOptions struct {
	// NameFilterType specifies whether Filters.Name must match a prefix or the full length of a TM name
	// Note that using FullMatch effectively limits the search result to at most one FoundEntry
	NameFilterType FilterType
}

// Filter deletes all entries from this SearchResult that don't match the filters
func (sr *SearchResult) Filter(filters *Filters) error {
	if filters == nil {
		return nil
	}
	filters.Sanitize()
	exclude := func(entry FoundEntry) bool {
		if !matchesNameFilter(filters.Name, entry.Name, filters.Options) {
			return true
		}

		if !matchesFilter(filters.Author, entry.Author.Name) {
			return true
		}

		if !matchesFilter(filters.Manufacturer, entry.Manufacturer.Name) {
			return true
		}

		if !matchesFilter(filters.Mpn, entry.Mpn) {
			return true
		}

		if !matchesProtocolFilter(filters.Protocol, entry) {
			return true
		}

		return false
	}
	sr.Entries = slices.DeleteFunc(sr.Entries, func(entry FoundEntry) bool {
		return exclude(entry)
	})
	return nil
}

// TextSearch deletes all versions from this SearchResult that don't match the search query. The entries that remain
// are extended with information on matches' locations.
func (sr *SearchResult) TextSearch(query, indexPath string) error {
	if query == "" {
		return nil
	}
	matcher, err := getMatcherByBleveSearch(query, indexPath)
	if err != nil {
		return err
	}
	if matcher != nil {
		var newEntries []FoundEntry
		for _, entry := range sr.Entries {
			var newVersions []FoundVersion
			for _, version := range entry.Versions {
				if matcher(version) {
					newVersions = append(newVersions, version)
				}
			}
			if len(newVersions) > 0 {
				entry.Versions = newVersions
				newEntries = append(newEntries, entry)
			}
		}
		sr.Entries = newEntries
	}
	return nil
}

func getMatcherByBleveSearch(query, indexPath string) (func(e FoundVersion) bool, error) {
	_, err := os.Stat(indexPath)
	if err != nil {
		return nil, ErrSearchIndexNotFound
	}
	bleveIdx, errOpen := bleve.Open(indexPath)
	if errOpen != nil {
		return nil, fmt.Errorf("couldn't open bleve index: %w", errOpen)
	} else {
		defer bleveIdx.Close()
		q := bleve.NewQueryStringQuery(query)
		req := bleve.NewSearchRequestOptions(q, 100000, 0, false)
		req.IncludeLocations = true
		sr, err := bleveIdx.Search(req)

		if err != nil {
			return nil, fmt.Errorf("error in content search: %w", err)
		}

		scores := make(map[string]*search.DocumentMatch)
		for _, hit := range sr.Hits {
			scores[hit.ID] = hit
		}

		matcher := func(v FoundVersion) bool {
			if sr.Hits.Len() == 0 {
				return false
			}
			if hit, ok := scores[v.TMID]; ok {
				v.SearchScore = float32(hit.Score)
				var locs []string
				for field, _ := range hit.Locations {
					locs = append(locs, field)
				}
				slices.Sort(locs)
				v.MatchLocations = locs
				return true
			}
			return false
		}
		return matcher, nil
	}
}

func matchesProtocolFilter(protos []string, entry FoundEntry) bool {
	if len(protos) == 0 {
		return true
	}
	for _, v := range entry.Versions {
		for _, p := range protos {
			if slices.Contains(v.Protocols, p) {
				return true
			}
		}
	}
	return false
}

func matchesNameFilter(acceptedValue string, value string, options FilterOptions) bool {
	if len(acceptedValue) == 0 {
		return true
	}

	switch options.NameFilterType {
	case FullMatch:
		return value == acceptedValue
	case PrefixMatch:
		actualPathParts := strings.Split(value, "/")
		acceptedValue = strings.Trim(acceptedValue, "/")
		acceptedPathParts := strings.Split(acceptedValue, "/")
		if len(acceptedPathParts) > len(actualPathParts) {
			return false
		}
		return slices.Equal(actualPathParts[0:len(acceptedPathParts)], acceptedPathParts)
	default:
		panic(fmt.Sprintf("unsupported NameFilterType: %d", options.NameFilterType))
	}
}

func matchesFilter(acceptedValues []string, value string) bool {
	if len(acceptedValues) == 0 {
		return true
	}
	return slices.Contains(acceptedValues, utils.SanitizeName(value))
}

func ToFilters(author, manufacturer, mpn, protocol, name *string, opts *FilterOptions) *Filters {
	var search *Filters
	isSet := func(s *string) bool { return s != nil && *s != "" }
	if isSet(author) || isSet(manufacturer) || isSet(mpn) || isSet(protocol) || isSet(name) {
		search = &Filters{}
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
