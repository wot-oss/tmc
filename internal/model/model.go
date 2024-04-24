package model

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/Masterminds/semver/v3"
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

type SchemaAuthor struct {
	Name string `json:"schema:name" validate:"required"`
}
type SchemaManufacturer struct {
	Name string `json:"schema:name" validate:"required"`
}

type Version struct {
	Model string `json:"model"`
}

type Attachment struct {
	Name    string
	Content []byte
}

type FetchName struct {
	Name   string
	Semver string
}

var ErrInvalidFetchName = errors.New("invalid fetch name")

var fetchNameRegex = regexp.MustCompile(`^([a-z\-0-9]+(/[\w\-0-9]+){2,})(:(.+))?$`)

func ParseFetchName(fetchName string) (FetchName, error) {
	// Find submatches in the input string
	matches := fetchNameRegex.FindStringSubmatch(fetchName)

	// Check if there are enough submatches
	if len(matches) < 2 {
		err := fmt.Errorf("%w: %s - must be NAME[:SEMVER]", ErrInvalidFetchName, fetchName)
		slog.Default().Error(err.Error())
		return FetchName{}, err
	}

	fn := FetchName{}
	// Extract values from submatches
	fn.Name = matches[1]
	if len(matches) > 4 && matches[4] != "" {
		fn.Semver = matches[4]
		_, err := semver.NewVersion(fn.Semver)
		if err != nil {
			return FetchName{}, fmt.Errorf("%w: %s - invalid semantic version", ErrInvalidFetchName, fetchName)
		}
	}
	return fn, nil
}

// ParseAsTMIDOrFetchName parses idOrName as model.TMID. If that fails, parses it as FetchName.
// Returns error is idOrName is not valid as either. Only one of returned pointers may be not nil
func ParseAsTMIDOrFetchName(idOrName string) (*TMID, *FetchName, error) {
	tmid, err1 := ParseTMID(idOrName)
	if err1 == nil {
		return &tmid, nil, nil
	}
	fn, err2 := ParseFetchName(idOrName)
	if err2 == nil {
		return nil, &fn, nil
	}

	slog.Default().Info("could not parse as either TMID or fetch name", "idOrName", idOrName)
	return nil, nil, fmt.Errorf("%w: %w, %w", ErrInvalidIdOrName, err1, err2)
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
