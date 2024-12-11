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
	indexPath   string
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
	// UseBleve indicates that the search query uses bleve query syntax
	UseBleve bool
}

func (sr *SearchResult) Filter(search *SearchParams) error {
	if search == nil {
		return nil
	}
	search.Sanitize()
	exclude := func(entry FoundEntry) bool {
		if !matchesNameFilter(search.Name, entry.Name, search.Options) {
			return true
		}

		if !matchesFilter(search.Author, entry.Author.Name) {
			return true
		}

		if !matchesFilter(search.Manufacturer, entry.Manufacturer.Name) {
			return true
		}

		if !matchesFilter(search.Mpn, entry.Mpn) {
			return true
		}

		if !matchesProtocolFilter(search.Protocol, entry) {
			return true
		}

		return false
	}
	sr.Entries = slices.DeleteFunc(sr.Entries, func(entry FoundEntry) bool {
		return exclude(entry)
	})
	if len(sr.Entries) == 0 {
		return nil
	}

	del, err := getSearchExclusionFunction(search, sr.indexPath)
	if err != nil {
		return err
	}
	if del != nil {
		sr.Entries = slices.DeleteFunc(sr.Entries, del)
	}
	return nil
}

func (sr *SearchResult) WithSearchIndex(indexPath string) *SearchResult {
	sr.indexPath = indexPath
	return sr
}

func getSearchExclusionFunction(search *SearchParams, indexPath string) (func(e FoundEntry) bool, error) {
	if search.Query == "" {
		return nil, nil
	}
	if search.Options.UseBleve {
		return excludeByContentSearch(search.Query, indexPath)
	} else {
		return excludeBySimpleContentSearch(search.Query)
	}
}

func excludeBySimpleContentSearch(searchQuery string) (func(e FoundEntry) bool, error) {
	return func(e FoundEntry) bool {
		searchQuery = utils.ToTrimmedLower(searchQuery)
		if strings.Contains(utils.ToTrimmedLower(e.Name), searchQuery) {
			return false
		}
		if strings.Contains(utils.ToTrimmedLower(e.Author.Name), searchQuery) {
			return false
		}
		if strings.Contains(utils.ToTrimmedLower(e.Manufacturer.Name), searchQuery) {
			return false
		}
		if strings.Contains(utils.ToTrimmedLower(e.Mpn), searchQuery) {
			return false
		}
		for _, version := range e.Versions {
			if strings.Contains(utils.ToTrimmedLower(version.Description), searchQuery) {
				return false
			}
			if strings.Contains(utils.ToTrimmedLower(version.ExternalID), searchQuery) {
				return false
			}
		}
		return true
	}, nil
}

func excludeByContentSearch(query, indexPath string) (func(e FoundEntry) bool, error) {
	_, err := os.Stat(indexPath)
	if err != nil {
		return nil, ErrSearchIndexNotFound
	}
	bleveIdx, errOpen := bleve.Open(indexPath)
	if errOpen != nil {
		return nil, fmt.Errorf("couldn't open bleve index: %w", errOpen)
	} else {
		defer bleveIdx.Close()
		query := bleve.NewQueryStringQuery(query)
		req := bleve.NewSearchRequestOptions(query, 100000, 0, true)
		sr, err := bleveIdx.Search(req)

		if err != nil {
			return nil, fmt.Errorf("error in content search: %w", err)
		}

		del := func(e FoundEntry) bool {
			if sr.Hits.Len() == 0 {
				return true
			}
			del := true
			for i, v := range e.Versions {
				for _, hv := range sr.Hits {
					parts := strings.Split(hv.ID, ":")
					if v.TMID == parts[0] {
						del = false
						e.Versions[i].SearchScore = float32(hv.Score)
					}
				}
			}
			return del
		}
		return del, nil
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

func matchesNameFilter(acceptedValue string, value string, options SearchOptions) bool {
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
