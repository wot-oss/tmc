package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
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
	protocols    []string
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

func ParseThingModel(data []byte) (*ThingModel, error) {
	var tm ThingModel
	err := json.Unmarshal(data, &tm)
	if err != nil {
		return nil, err
	}

	protos, _ := collectProtocols(data)
	tm.protocols = protos
	return &tm, nil
}

// collectProtocols parses byte array containing a TM and returns all URL protocol schemes contained in the TM
func collectProtocols(data []byte) ([]string, error) {
	var tm map[string]any
	err := json.Unmarshal(data, &tm)
	if err != nil {
		return nil, err
	}
	properties, _ := utils.JsGetMap(tm, "properties")
	actions, _ := utils.JsGetMap(tm, "actions")
	events, _ := utils.JsGetMap(tm, "events")

	var protos []string
	base, _ := utils.JsGetString(tm, "base")
	baseProto := extractProtocol(base)
	if baseProto != "" {
		protos = append(protos, baseProto)
	}
	protos = append(protos, extractFormsProtocols(tm)...)
	for _, m := range []map[string]any{properties, actions, events} {
		protos = append(protos, extractAffordancesFormsProtocols(m)...)
	}
	slices.Sort(protos)
	protos = slices.Compact(protos)
	return protos, nil
}

func extractAffordancesFormsProtocols(affs map[string]any) []string {
	var protos []string
	for k, _ := range affs {
		aff, _ := utils.JsGetMap(affs, k)
		protos = append(protos, extractFormsProtocols(aff)...)
	}
	return protos
}

func extractFormsProtocols(m map[string]any) []string {
	var protos []string
	forms := utils.JsGetArray(m, "forms")
	for _, f := range forms {
		form, _ := f.(map[string]interface{})
		href, _ := utils.JsGetString(form, "href")
		proto := extractProtocol(href)
		if proto != "" {
			protos = append(protos, proto)
		}
	}
	return protos
}

var placeholdersRegexp = regexp.MustCompile("{{.+}}")

func extractProtocol(uri string) string {
	if uri == "" {
		return ""
	}

	// replace any placeholders in the URI with a string that will most probably make the resulting URI a valid one for parsing
	uri = placeholdersRegexp.ReplaceAllString(uri, "example.com")

	u, err := url.Parse(uri)
	if err != nil { //skip unparseable hrefs
		return ""
	}
	return strings.ToLower(u.Scheme)
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

func (t CheckResultType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

type CheckResult struct {
	Typ          CheckResultType `json:"type"`
	ResourceName string          `json:"resource"`
	Message      string          `json:"message,omitempty"`
}

func (r CheckResult) String() string {
	return fmt.Sprintf("%v \t%s: %s", r.Typ, r.ResourceName, r.Message)
}

// ResourceFilter is a function which determines whether a named resource should be processed or not
type ResourceFilter func(string) bool
