package model

import (
	"errors"
	"fmt"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

// ThingModel is a model for unmarshalling a Thing Model to be
// imported. It contains only the fields required to be accepted into
// the catalog.
type ThingModel struct {
	ID           string             `json:"id,omitempty"`
	Description  string             `json:"description"`
	Manufacturer SchemaManufacturer `json:"schema:manufacturer" validate:"required"`
	Mpn          string             `json:"schema:mpn" validate:"required"`
	Author       SchemaAuthor       `json:"schema:author" validate:"required"`
	Version      Version            `json:"version"`
	Links        `json:"links"`
}

func (tm *ThingModel) IsOfficial() bool {
	return EqualsAsSchemaName(tm.Manufacturer.Name, tm.Author.Name)
}

func EqualsAsSchemaName(s1, s2 string) bool {
	return utils.ToTrimmedLower(s1) == utils.ToTrimmedLower(s2)
}

type SchemaAuthor struct {
	Name string `json:"schema:name" validate:"required"`
}
type SchemaManufacturer struct {
	Name string `json:"schema:name" validate:"required"`
}

type Version struct {
	Model string `json:"model"`
}

type RepoSpec struct {
	remoteName string
	dir        string
}

func NewSpec(remoteName, dir string) (RepoSpec, error) {
	if remoteName != "" && dir != "" {
		return RepoSpec{}, ErrInvalidSpec
	}
	return RepoSpec{
		remoteName: remoteName,
		dir:        dir,
	}, nil
}

func NewRemoteSpec(remoteName string) RepoSpec {
	return RepoSpec{
		remoteName: remoteName,
	}
}

func NewDirSpec(dir string) RepoSpec {
	return RepoSpec{
		dir: dir,
	}
}

func NewSpecFromFoundSource(s FoundSource) RepoSpec {
	return RepoSpec{
		remoteName: s.RemoteName,
		dir:        s.Directory,
	}
}

func (r RepoSpec) ToFoundSource() FoundSource {
	return FoundSource{
		Directory:  r.dir,
		RemoteName: r.remoteName,
	}
}

func (r RepoSpec) Dir() string {
	return r.dir
}
func (r RepoSpec) RemoteName() string {
	return r.remoteName
}
func (r RepoSpec) String() string {
	if r.dir == "" {
		if r.remoteName == "" {
			return fmt.Sprintf("undefined repository")
		}
		return fmt.Sprintf("remote <%s>", r.remoteName)
	}
	return fmt.Sprintf("directory %s", r.dir)
}

var EmptySpec, _ = NewSpec("", "")

var ErrInvalidSpec = errors.New("illegal remote spec: both dir and remoteName given")
