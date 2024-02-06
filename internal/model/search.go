package model

import (
	"slices"
	"strings"
)

const (
	FullMatch FilterType = iota
	PrefixMatch
)

type SearchResult struct {
	Entries []FoundEntry
}
type FoundEntry struct {
	Name         string
	Manufacturer SchemaManufacturer
	Mpn          string
	Author       SchemaAuthor
	Versions     []FoundVersion
}
type FoundVersion struct {
	TOCVersion
	FoundIn FoundSource
}

type FoundSource struct {
	Directory  string
	RemoteName string
}

func (s FoundSource) String() string {
	if s.Directory != "" {
		return s.Directory
	}
	return "<" + s.RemoteName + ">"
}

func MergeFoundVersions(vs1, vs2 []FoundVersion) []FoundVersion {
	vs1 = append(vs1, vs2...)
	// whether the TMIDs are actually official or not is not important for these comparisons
	slices.SortStableFunc(vs1, func(a, b FoundVersion) int {
		tmid1, _ := ParseTMID(a.TMID, true)
		tmid2, _ := ParseTMID(b.TMID, true)
		if tmid1.Equals(tmid2) {
			return -strings.Compare(tmid1.Version.Timestamp, tmid2.Version.Timestamp) // sort in reverse chronological order within the same TMID
		}
		return strings.Compare(a.TMID, b.TMID)
	})
	return slices.CompactFunc(vs1, func(v1, v2 FoundVersion) bool {
		tmid1, _ := ParseTMID(v1.TMID, true)
		tmid2, _ := ParseTMID(v2.TMID, true)
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

type FilterType byte

type SearchOptions struct {
	// NameFilterType specifies whether SearchParams.Name must match a prefix or the full length of a TM name
	// Note that using FullMatch effectively limits the search result to at most one FoundEntry
	NameFilterType FilterType
}
