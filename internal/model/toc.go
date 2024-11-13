package model

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/wot-oss/tmc/internal/utils"
)

type Index struct {
	Meta       IndexMeta     `json:"meta"`
	Data       []*IndexEntry `json:"data"`
	dataByName map[string]*IndexEntry
}

func (idx *Index) reindexData() {
	idx.dataByName = make(map[string]*IndexEntry)
	for _, v := range idx.Data {
		idx.dataByName[v.Name] = v
	}
}

type IndexMeta struct {
	Created time.Time `json:"created"`
}

type AttachmentContainer struct {
	Attachments []Attachment `json:"attachments,omitempty"`
}

func (ac *AttachmentContainer) FindAttachment(name string) (att Attachment, found bool) {
	if ac == nil {
		return Attachment{}, false
	}
	for _, a := range ac.Attachments {
		if a.Name == name {
			return a, true
		}
	}
	return Attachment{}, false
}

type IndexEntry struct {
	Name         string             `json:"name"`
	Manufacturer SchemaManufacturer `json:"schema:manufacturer" validate:"required"`
	Mpn          string             `json:"schema:mpn" validate:"required"`
	Author       SchemaAuthor       `json:"schema:author" validate:"required"`
	Versions     []*IndexVersion    `json:"versions"`
	AttachmentContainer
}

type Attachment struct {
	Name      string `json:"name"`
	MediaType string `json:"mediaType,omitempty"`
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
		slices.SortStableFunc(entry.Versions, func(a, b *IndexVersion) int {
			aid, _ := ParseTMID(a.TMID)
			bid, _ := ParseTMID(b.TMID)
			return -aid.Version.Compare(bid.Version)
		})
	}
	// sort entries ascending
	slices.SortFunc(idx.Data, func(a *IndexEntry, b *IndexEntry) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func (idx *Index) Filter(search *SearchParams) error {
	if search == nil {
		return nil
	}
	search.Sanitize()
	exclude := func(entry *IndexEntry) bool {
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
	if len(idx.Data) == 0 {
		return nil
	}

	if len(search.Query) > 0 {
		del, err := excludeBySimpleContentSearch(search.Query)
		if err != nil {
			return err
		}
		if del != nil {
			idx.Data = slices.DeleteFunc(idx.Data, del)
		}
	}
	return nil
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

func excludeBySimpleContentSearch(searchQuery string) (func(e *IndexEntry) bool, error) {
	return func(e *IndexEntry) bool {
		if e == nil {
			return true
		}
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
			return v
		}
	}
	return nil
}

// Insert uses ThingModel to add a version, either to an existing
// entry or as a new entry.
func (idx *Index) Insert(ctm *ThingModel) error {
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
	tv := &IndexVersion{
		Description: ctm.Description,
		TimeStamp:   tmid.Version.Timestamp,
		Version:     Version{Model: tmid.Version.Base.String()},
		TMID:        ctm.ID,
		ExternalID:  externalID,
		Digest:      tmid.Version.Hash,
		Links:       map[string]string{"content": tmid.String()},
	}
	if idx := slices.IndexFunc(idxEntry.Versions, func(version *IndexVersion) bool {
		return version.TMID == ctm.ID
	}); idx == -1 {
		idxEntry.Versions = append(idxEntry.Versions, tv)
	} else {
		idxEntry.Versions[idx] = tv
	}
	return nil
}

func (idx *Index) InsertAttachments(ref AttachmentContainerRef, atts ...Attachment) error {
	container, _, err := idx.FindAttachmentContainer(ref)
	if err != nil {
		return err
	}
	for _, att := range atts {
		found := false
		na := Attachment{
			Name:      att.Name,
			MediaType: att.MediaType,
		}
		for i, ea := range container.Attachments {
			if att.Name == ea.Name {
				container.Attachments[i] = na
				found = true
				break
			}
		}
		if !found {
			container.Attachments = append(container.Attachments, na)
		}
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
		idxEntry.Versions = slices.DeleteFunc(idxEntry.Versions, func(version *IndexVersion) bool {
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

func (idx *Index) FindAttachmentContainer(ref AttachmentContainerRef) (*AttachmentContainer, *IndexEntry, error) {
	k := ref.Kind()
	var tmName string
	switch k {
	case AttachmentContainerKindInvalid:
		return nil, nil, ErrInvalidIdOrName
	case AttachmentContainerKindTMID:
		id, err := ParseTMID(ref.TMID)
		if err != nil {
			return nil, nil, err
		}
		tmName = id.Name
	case AttachmentContainerKindTMName:
		fn, err := ParseFetchName(ref.TMName)
		if err != nil || fn.Semver != "" {
			return nil, nil, ErrInvalidIdOrName
		}
		tmName = ref.TMName
	}

	indexEntry := idx.FindByName(tmName)
	if indexEntry == nil {
		if ref.Kind() == AttachmentContainerKindTMID {
			return nil, nil, ErrTMNotFound
		} else {
			return nil, nil, ErrTMNameNotFound
		}
	}
	versions := indexEntry.Versions
	if k == AttachmentContainerKindTMID {
		for _, v := range versions {
			if v.TMID == ref.TMID {
				return &v.AttachmentContainer, indexEntry, nil
			}
		}
		return nil, nil, ErrTMNotFound
	}
	return &indexEntry.AttachmentContainer, indexEntry, nil
}

const AttachmentsDir = ".attachments"

// RelAttachmentsDir is a helper function which calculates the relative path of the attachments directory for
// given attachment container. That is, e.g. 'author/manufacturer/mpn/.attachments' for a TMName ref and
// 'author/manufacturer/mpn/.attachments/v1.0.0-20240108112117-2cd14601ef09' for a TMID ref
func RelAttachmentsDir(ref AttachmentContainerRef) (string, error) {
	var attDir string
	switch ref.Kind() {
	case AttachmentContainerKindInvalid:
		return "", fmt.Errorf("invalid attachment container reference: %w: %v", ErrInvalidIdOrName, ref)
	case AttachmentContainerKindTMID:
		id, err := ParseTMID(ref.TMID)
		if err != nil {
			return "", fmt.Errorf("invalid attachment container reference: %w: %v", err, ref)
		}
		attDir = fmt.Sprintf("%s/%s/%s", id.Name, AttachmentsDir, id.Version.String())
	case AttachmentContainerKindTMName:
		attDir = fmt.Sprintf("%s/%s", ref.TMName, AttachmentsDir)
	}
	return attDir, nil
}
