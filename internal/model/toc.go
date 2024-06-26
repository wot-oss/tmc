package model

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/wot-oss/tmc/internal/utils"
)

type Index struct {
	Meta       IndexMeta     `json:"meta"`
	Data       []*IndexEntry `json:"data"`
	dataByName map[string]*IndexEntry
}

func (i *Index) reindexData() {
	i.dataByName = make(map[string]*IndexEntry)
	for _, v := range i.Data {
		i.dataByName[v.Name] = v
	}
}

type IndexMeta struct {
	Created time.Time `json:"created"`
}

type AttachmentContainer struct {
	Attachments []Attachment `json:"attachments,omitempty"`
}

type IndexEntry struct {
	Name         string             `json:"name"`
	Manufacturer SchemaManufacturer `json:"schema:manufacturer" validate:"required"`
	Mpn          string             `json:"schema:mpn" validate:"required"`
	Author       SchemaAuthor       `json:"schema:author" validate:"required"`
	Versions     []IndexVersion     `json:"versions"`
	AttachmentContainer
}

type Attachment struct {
	Name string `json:"name"`
}

// AttachmentContainerRef contains a reference to an entity which can have file attachments
// Either TMName field must be not empty, or TMID. Never both and never none.
type AttachmentContainerRef struct {
	TMName string
	TMID   string
}

type AttachmentContainerKind byte

const (
	AttachmentContainerKindInvalid AttachmentContainerKind = iota
	AttachmentContainerKindTMName
	AttachmentContainerKindTMID
)

func NewTMIDAttachmentContainerRef(tmid string) AttachmentContainerRef {
	return AttachmentContainerRef{TMID: tmid}
}
func NewTMNameAttachmentContainerRef(tmName string) AttachmentContainerRef {
	return AttachmentContainerRef{TMName: tmName}
}
func (r AttachmentContainerRef) String() string {
	switch r.Kind() {
	case AttachmentContainerKindInvalid:
		return fmt.Sprintf("invalid AttachmentContainerRef (TMID=%s, TMName=%s)", r.TMID, r.TMName)
	case AttachmentContainerKindTMID:
		return fmt.Sprintf("TMID=%s", r.TMID)
	case AttachmentContainerKindTMName:
		return fmt.Sprintf("TMName=%s", r.TMName)
	default:
		return fmt.Sprintf("unknown container ref kind:%v", r.Kind())
	}
}

func (r AttachmentContainerRef) Kind() AttachmentContainerKind {
	if (r.TMID != "" && r.TMName != "") || (r.TMID == "" && r.TMName == "") {
		return AttachmentContainerKindInvalid
	}
	if r.TMName != "" {
		return AttachmentContainerKindTMName
	}
	return AttachmentContainerKindTMID
}

func (e *IndexEntry) MatchesSearchText(searchQuery string) bool {
	if e == nil {
		return false
	}
	searchQuery = utils.ToTrimmedLower(searchQuery)
	if strings.Contains(utils.ToTrimmedLower(e.Name), searchQuery) {
		return true
	}
	if strings.Contains(utils.ToTrimmedLower(e.Author.Name), searchQuery) {
		return true
	}
	if strings.Contains(utils.ToTrimmedLower(e.Manufacturer.Name), searchQuery) {
		return true
	}
	if strings.Contains(utils.ToTrimmedLower(e.Mpn), searchQuery) {
		return true
	}
	for _, version := range e.Versions {
		if strings.Contains(utils.ToTrimmedLower(version.Description), searchQuery) {
			return true
		}
		if strings.Contains(utils.ToTrimmedLower(version.ExternalID), searchQuery) {
			return true
		}
	}
	return false

}

const TMLinkRel = "content"

type IndexVersion struct {
	Description string            `json:"description"`
	Version     Version           `json:"version"`
	Links       map[string]string `json:"links"`
	TMID        string            `json:"tmID"`
	Digest      string            `json:"digest"`
	TimeStamp   string            `json:"timestamp,omitempty"`
	ExternalID  string            `json:"externalID"`
	AttachmentContainer
}

func (idx *Index) IsEmpty() bool {
	return len(idx.Data) == 0
}

func (idx *Index) Sort() {
	if idx.IsEmpty() {
		return
	}
	// sort versions of each entry descending
	for _, entry := range idx.Data {
		slices.SortFunc(entry.Versions, func(a IndexVersion, b IndexVersion) int {
			av := semver.MustParse(a.Version.Model)
			bv := semver.MustParse(b.Version.Model)
			vc := bv.Compare(av)
			if vc != 0 {
				return vc
			}
			vc = strings.Compare(b.TimeStamp, a.TimeStamp) // our timestamps can be compared lexicographically
			if vc != 0 {
				return vc
			}
			return strings.Compare(b.TMID, a.TMID) // in case of semVer and timestamp equality, use complete ID to ensure stable order
		})
	}
	// sort entries ascending
	slices.SortFunc(idx.Data, func(a *IndexEntry, b *IndexEntry) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func (idx *Index) Filter(search *SearchParams) {
	if search == nil {
		return
	}
	search.Sanitize()
	exclude := func(entry *IndexEntry) bool {
		if !entry.MatchesSearchText(search.Query) {
			return true
		}

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

		return false
	}
	idx.Data = slices.DeleteFunc(idx.Data, func(entry *IndexEntry) bool {
		e := exclude(entry)
		if e && idx.dataByName != nil {
			delete(idx.dataByName, entry.Name)
		}
		return e
	})

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

// FindByName searches by TM name and returns a pointer to the IndexEntry if found
func (idx *Index) FindByName(name string) *IndexEntry {
	if idx.dataByName == nil {
		idx.reindexData()
	}
	return idx.dataByName[name]
}

// FindByTMID searches by TM name and returns a pointer to the IndexVersion if found.
// returns nil if tmID is not valid or not found in
func (idx *Index) FindByTMID(tmID string) *IndexVersion {
	id, err := ParseTMID(tmID)
	if err != nil {
		return nil
	}
	e := idx.FindByName(id.Name)
	if e == nil {
		return nil
	}
	for _, v := range e.Versions {
		if v.TMID == tmID {
			return &v
		}
	}
	return nil
}

func mapAttachments(atts []string) []Attachment {
	var res []Attachment
	for _, a := range atts {
		res = append(res, Attachment{Name: a})
	}
	return res
}

func (idx *Index) SetEntryAttachments(name string, attachmentNames []string) {
	entry := idx.FindByName(name)
	if entry != nil {
		entry.Attachments = mapAttachments(attachmentNames)
	}
}

// Insert uses ThingModel to add a version, either to an existing
// entry or as a new entry.
func (idx *Index) Insert(ctm *ThingModel, tmAttachments []string) error {
	mapAttachments := func(atts []string) []Attachment {
		var res []Attachment
		for _, a := range atts {
			res = append(res, Attachment{Name: a})
		}
		return res
	}

	tmid, err := ParseTMID(ctm.ID)
	if err != nil {
		return err
	}
	// find the right entry, or create if it doesn't exist
	idxEntry := idx.FindByName(tmid.Name)
	if idxEntry == nil {
		idxEntry = &IndexEntry{
			Name:         tmid.Name,
			Manufacturer: SchemaManufacturer{Name: ctm.Manufacturer.Name},
			Mpn:          ctm.Mpn,
			Author:       SchemaAuthor{Name: ctm.Author.Name},
		}
		idx.Data = append(idx.Data, idxEntry)
		idx.dataByName[idxEntry.Name] = idxEntry
	}
	// TODO: check if id already exists?
	// Append version information to entry
	externalID := ""
	original := ctm.Links.FindLink("original")
	if original != nil {
		externalID = original.HRef
	}
	tv := IndexVersion{
		Description:         ctm.Description,
		TimeStamp:           tmid.Version.Timestamp,
		Version:             Version{Model: tmid.Version.Base.String()},
		TMID:                ctm.ID,
		ExternalID:          externalID,
		Digest:              tmid.Version.Hash,
		Links:               map[string]string{"content": tmid.String()},
		AttachmentContainer: AttachmentContainer{mapAttachments(tmAttachments)},
	}
	if idx := slices.IndexFunc(idxEntry.Versions, func(version IndexVersion) bool {
		return version.TMID == ctm.ID
	}); idx == -1 {
		idxEntry.Versions = append(idxEntry.Versions, tv)
	} else {
		idxEntry.Versions[idx] = tv
	}
	return nil
}

// Delete deletes the record for the given id. Returns TM name to be removed from names file if no more versions are left
func (idx *Index) Delete(id string) (updated bool, deletedName string, err error) {
	var idxEntry *IndexEntry

	name, found := strings.CutSuffix(id, "/"+filepath.Base(id))
	if !found {
		return false, "", ErrInvalidId
	}
	idxEntry = idx.FindByName(name)
	if idxEntry != nil {
		idxEntry.Versions = slices.DeleteFunc(idxEntry.Versions, func(version IndexVersion) bool {
			fnd := version.TMID == id
			if fnd {
				updated = true
			}
			return fnd
		})
		if len(idxEntry.Versions) == 0 {
			idx.Data = slices.DeleteFunc(idx.Data, func(entry *IndexEntry) bool {
				return entry.Name == name
			})
			delete(idx.dataByName, name)
			return updated, name, nil
		}
	}
	return updated, "", nil
}

const AttachmentsDir = ".attachments"

// RelAttachmentsDir is a helper function which calculates the relative path of the attachments directory for
// given attachment container. That is, e.g. 'author/manufacturer/mpn/.attachments' for a TMName ref and
// 'author/manufacturer/mpn/.attachments/v1.0.0-20240108112117-2cd14601ef09' for a TMID ref
func RelAttachmentsDir(ref AttachmentContainerRef) (string, error) {
	var attDir string
	switch ref.Kind() {
	case AttachmentContainerKindInvalid:
		return "", fmt.Errorf("%w: %v", ErrInvalidIdOrName, ref)
	case AttachmentContainerKindTMID:
		id, err := ParseTMID(ref.TMID)
		if err != nil {
			return "", err
		}
		attDir = fmt.Sprintf("%s/%s/%s", id.Name, AttachmentsDir, id.Version.String())
	case AttachmentContainerKindTMName:
		attDir = fmt.Sprintf("%s/%s", ref.TMName, AttachmentsDir)
	}
	slog.Default().Debug("attachments dir for ref calculated", "container", ref, "attDir", attDir)
	return attDir, nil

}
