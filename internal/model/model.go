package model

import (
	"errors"
	"fmt"
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

	return nil, nil, fmt.Errorf("could not parse %s as either TMID or fetch name: %w: %w, %w", idOrName, ErrInvalidIdOrName, err1, err2)
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

func (s RepoSpec) ToFoundSource() FoundSource {
	return FoundSource{
		Directory: s.dir,
		RepoName:  s.repoName,
	}
}

func (s RepoSpec) Dir() string {
	return s.dir
}
func (s RepoSpec) RepoName() string {
	return s.repoName
}
func (s RepoSpec) String() string {
	if s.dir == "" {
		if s.repoName == "" {
			return fmt.Sprintf("unspecified repo")
		}
		return fmt.Sprintf("named repo <%s>", s.repoName)
	}
	return fmt.Sprintf("local repo %s", s.dir)
}

func (s RepoSpec) IsEmpty() bool {
	return s.dir == "" && s.repoName == ""
}

var EmptySpec, _ = NewSpec("", "")

var ErrInvalidSpec = errors.New("illegal repo spec: both local directory and repo name given")

type RepoDescription struct {
	Name        string
	Type        string
	Description string
}

type CheckResultType int

const (
	CheckOK = CheckResultType(iota)
	CheckErr
)

func (t CheckResultType) String() string {
	switch t {
	case CheckOK:
		return "OK"
	case CheckErr:
		return "error"
	default:
		return "unknown"
	}
}

type CheckResult struct {
	Typ          CheckResultType
	ResourceName string
	Message      string
}

func (r CheckResult) String() string {
	return fmt.Sprintf("%v \t%s: %s", r.Typ, r.ResourceName, r.Message)
}

// ResourceFilter is a function which determines whether a named resource should be processed or not
type ResourceFilter func(string) bool
