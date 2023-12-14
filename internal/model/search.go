package model

import (
	"slices"
	"strings"
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
	FoundIn string
}

func NewSearchResultFromTOC(toc TOC, foundIn string) SearchResult {
	r := SearchResult{}
	var es []FoundEntry
	for _, e := range toc.Data {
		es = append(es, NewFoundEntryFromTOCEntry(e, foundIn))
	}
	r.Entries = es
	return r
}

func NewFoundEntryFromTOCEntry(e *TOCEntry, foundIn string) FoundEntry {
	return FoundEntry{
		Name:         e.Name,
		Manufacturer: e.Manufacturer,
		Mpn:          e.Mpn,
		Author:       e.Author,
		Versions:     toFoundVersions(e.Versions, foundIn),
	}
}

func mergeFoundVersions(vs1, vs2 []FoundVersion) []FoundVersion {
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
	r.Versions = mergeFoundVersions(r.Versions, other.Versions)
	return r
}

func toFoundVersions(versions []TOCVersion, fromRemote string) []FoundVersion {
	var r []FoundVersion
	for _, v := range versions {
		r = append(r, FoundVersion{
			TOCVersion: v,
			FoundIn:    fromRemote,
		})
	}
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
	ExternalID   []string
	Name         string
	Query        string
}
