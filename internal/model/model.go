package model

import (
	"errors"
	"fmt"

	"github.com/wot-oss/tmc/internal/utils"
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
	repoName string
	dir      string
}

func NewSpec(repoName, dir string) (RepoSpec, error) {
	if repoName != "" && dir != "" {
		return RepoSpec{}, ErrInvalidSpec
	}
	return RepoSpec{
		repoName: repoName,
		dir:      dir,
	}, nil
}

func NewRepoSpec(repoName string) RepoSpec {
	return RepoSpec{
		repoName: repoName,
	}
}

func NewDirSpec(dir string) RepoSpec {
	return RepoSpec{
		dir: dir,
	}
}

func NewSpecFromFoundSource(s FoundSource) RepoSpec {
	return RepoSpec{
		repoName: s.RepoName,
		dir:      s.Directory,
	}
}

func (r RepoSpec) ToFoundSource() FoundSource {
	return FoundSource{
		Directory: r.dir,
		RepoName:  r.repoName,
	}
}

func (r RepoSpec) Dir() string {
	return r.dir
}
func (r RepoSpec) RepoName() string {
	return r.repoName
}
func (r RepoSpec) String() string {
	if r.dir == "" {
		if r.repoName == "" {
			return fmt.Sprintf("unspecified repo")
		}
		return fmt.Sprintf("named repo <%s>", r.repoName)
	}
	return fmt.Sprintf("local repo %s", r.dir)
}

var EmptySpec, _ = NewSpec("", "")

var ErrInvalidSpec = errors.New("illegal repo spec: both local directory and repo name given")
