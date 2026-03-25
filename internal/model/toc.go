package model

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type Index struct {
	Meta                    IndexMeta     `json:"meta"`
	Data                    []*IndexEntry `json:"data"`
	dataByName              map[string]*IndexEntry
	authorAttachments       map[string]*IndexEntry
	manufacturerAttachments map[string]*IndexEntry
}

func (idx *Index) reindexData() {
	idx.dataByName = make(map[string]*IndexEntry)
	idx.authorAttachments = make(map[string]*IndexEntry)
	idx.manufacturerAttachments = make(map[string]*IndexEntry)
	for _, v := range idx.Data {
		idx.dataByName[v.Name] = v
		parts := strings.Split(v.Name, "/")
		if len(parts) >= 1 {
			authorName := parts[0]
			if _, exists := idx.authorAttachments[authorName]; !exists {
				idx.authorAttachments[authorName] = v
			}
		}
		if len(parts) >= 2 {
			manufacturerName := parts[0] + "/" + parts[1]
			if _, exists := idx.manufacturerAttachments[manufacturerName]; !exists {
				idx.manufacturerAttachments[manufacturerName] = v
			}
		}
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
	Manufacturer SchemaManufacturer `json:"schema:manufacturer"`
	Mpn          string             `json:"schema:mpn"`
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
	Author       string
	Manufacturer string
	TMName       string
	TMID         string
}

type AttachmentContainerKind byte

const (
	AttachmentContainerKindInvalid AttachmentContainerKind = iota
	AttachmentContainerKindAuthor
	AttachmentContainerKindManufacturer
	AttachmentContainerKindTMName
	AttachmentContainerKindTMID
)

func NewAuthorAttachmentContainerRef(author string) AttachmentContainerRef {
	return AttachmentContainerRef{TMName: author, Author: author}
}
func NewManufacturerAttachmentContainerRef(manufacturer string) AttachmentContainerRef {
	return AttachmentContainerRef{TMName: manufacturer, Manufacturer: manufacturer}
}
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
	case AttachmentContainerKindAuthor:
		return fmt.Sprintf("Author=%s", r.Author)
	case AttachmentContainerKindManufacturer:
		return fmt.Sprintf("Manufacturer=%s", r.Manufacturer)
	case AttachmentContainerKindTMID:
		return fmt.Sprintf("TMID=%s", r.TMID)
	case AttachmentContainerKindTMName:
		return fmt.Sprintf("TMName=%s", r.TMName)
	default:
		return fmt.Sprintf("unknown container ref kind:%v", r.Kind())
	}
}

func (r AttachmentContainerRef) Kind() AttachmentContainerKind {
	if (r.Author != "" && r.Manufacturer != "" && r.TMID != "" && r.TMName != "") || (r.Author == "" && r.Manufacturer == "" && r.TMID == "" && r.TMName == "") {
		return AttachmentContainerKindInvalid
	}
	if r.Author != "" {
		return AttachmentContainerKindAuthor
	}
	if r.Manufacturer != "" {
		return AttachmentContainerKindManufacturer
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
	Protocols   []string          `json:"protocols,omitempty"`
	SearchMatch *SearchMatch      `json:"searchMatch,omitempty"`
	AttachmentContainer
}

type SearchMatch struct {
	Score     float32  `json:"score,omitempty"`
	Locations []string `json:"locations,omitempty"`
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

// FindByName searches by TM name and returns a pointer to the IndexEntry if found
func (idx *Index) FindByName(name string) *IndexEntry {
	if idx.dataByName == nil {
		idx.reindexData()
	}
	return idx.dataByName[name]
}

func (idx *Index) findByAuthor(author string) *IndexEntry {
	if idx.authorAttachments == nil {
		idx.reindexData()
	}
	return idx.authorAttachments[author]
}

func (idx *Index) findByManufacturer(manufacturer string) *IndexEntry {
	if idx.manufacturerAttachments == nil {
		idx.reindexData()
	}
	return idx.manufacturerAttachments[manufacturer]
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
	parts := strings.Split(tmid.Name, "/")
	if len(parts) >= 1 {
		authorName := parts[0]
		idxEntry := idx.findByAuthor(authorName)
		if idxEntry == nil {
			idxEntry = &IndexEntry{
				Name:   parts[0],
				Author: SchemaAuthor{Name: parts[0]},
			}
			idx.Data = append(idx.Data, idxEntry)
			idx.authorAttachments[idxEntry.Name] = idxEntry
		}
	}
	if len(parts) >= 2 {
		manufacturerName := parts[0] + "/" + parts[1]
		idxEntry := idx.findByManufacturer(manufacturerName)
		if idxEntry == nil {
			idxEntry = &IndexEntry{
				Name:         parts[0] + "/" + parts[1],
				Author:       SchemaAuthor{Name: parts[0]},
				Manufacturer: SchemaManufacturer{Name: parts[1]},
			}
			idx.Data = append(idx.Data, idxEntry)
			idx.manufacturerAttachments[idxEntry.Name] = idxEntry
		}
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
		Protocols:   ctm.protocols,
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
			if _, exists := idx.dataByName[name]; exists {
				delete(idx.dataByName, name)
			}
			if _, exists := idx.authorAttachments[name]; exists {
				delete(idx.authorAttachments, name)
			}
			if _, exists := idx.manufacturerAttachments[name]; exists {
				delete(idx.manufacturerAttachments, name)
			}
			return updated, name, nil
		}
	}
	return updated, "", nil
}

func (idx *Index) FindAttachmentContainer(ref AttachmentContainerRef) (*AttachmentContainer, *IndexEntry, error) {
	k := ref.Kind()
	var indexEntry *IndexEntry
	switch k {
	case AttachmentContainerKindInvalid:
		return nil, nil, ErrInvalidIdOrName
	case AttachmentContainerKindAuthor:
		indexEntry = idx.findByAuthor(ref.Author)
	case AttachmentContainerKindManufacturer:
		indexEntry = idx.findByManufacturer(ref.Manufacturer)
	case AttachmentContainerKindTMID:
		id, err := ParseTMID(ref.TMID)
		if err != nil {
			return nil, nil, err
		}
		indexEntry = idx.FindByName(id.Name)
	case AttachmentContainerKindTMName:
		fn, err := ParseFetchName(ref.TMName)
		if err != nil || fn.Semver != "" {
			return nil, nil, ErrInvalidIdOrName
		}
		indexEntry = idx.FindByName(ref.TMName)
	}

	if indexEntry == nil {
		if ref.Kind() == AttachmentContainerKindAuthor {
			return nil, nil, ErrAuthorNotFound
		} else if ref.Kind() == AttachmentContainerKindManufacturer {
			return nil, nil, ErrManufacturerNotFound
		} else if ref.Kind() == AttachmentContainerKindTMID {
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
	case AttachmentContainerKindAuthor:
		attDir = fmt.Sprintf("%s/%s", ref.Author, AttachmentsDir)
	case AttachmentContainerKindManufacturer:
		attDir = fmt.Sprintf("%s/%s", ref.Manufacturer, AttachmentsDir)
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
