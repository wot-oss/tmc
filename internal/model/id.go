package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/wot-oss/tmc/internal/utils"
)

var (
	ErrInvalidVersion       = errors.New("invalid version string")
	ErrInvalidPseudoVersion = errors.New("no valid pseudo-version found")
	ErrInvalidId            = errors.New("id invalid")
	ErrInvalidIdOrName      = errors.New("id or name invalid")
	ErrVersionDiffers       = errors.New("id has a differing version from given ThingModel")
)

type TMID struct {
	Name    string
	Version TMVersion
}

func NewTMID(author, manufacturer, mpn, optPath string, version TMVersion) TMID {
	optPathParts := strings.Split(optPath, "/")
	for i, p := range optPathParts {
		optPathParts[i] = utils.SanitizeName(p)
	}
	parts := []string{utils.SanitizeName(author), utils.SanitizeName(manufacturer), utils.SanitizeName(mpn)}
	parts = append(parts, optPathParts...)
	name := JoinSkippingEmpty(parts, "/")
	id := TMID{
		Name:    name,
		Version: version,
	}
	return id
}

type TMVersion struct {
	Base      *semver.Version
	Timestamp string
	Hash      string
}

var pseudoVersionRegex *regexp.Regexp

const (
	TMFileExtension              = ".tm.json"
	PseudoVersionTimestampFormat = "20060102150405"
	pseudoVersionRegexString     = "(([0-9A-Za-z\\-]+)\\-)?([0-9]{14})-([0-9a-z]{12})"
)

func (v TMVersion) String() string {
	res := v.BaseString()
	if len(v.Timestamp) > 0 {
		res += "-" + v.Timestamp
	}
	if len(v.Hash) > 0 {
		res += "-" + v.Hash
	}
	return res
}

func (v TMVersion) BaseString() string {
	res := ""
	if v.Base != nil {
		res += v.Base.Original()
	}
	return res
}

func (id TMID) String() string {
	return fmt.Sprintf("%s/%s%s", id.Name, id.Version, TMFileExtension)
}

func JoinSkippingEmpty(elems []string, sep string) string {
	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0]
	}
	var b strings.Builder
	b.WriteString(elems[0])
	for _, s := range elems[1:] {
		if s != "" {
			b.WriteString(sep)
			b.WriteString(s)
		}
	}
	return b.String()
}

func init() {
	pseudoVersionRegex = regexp.MustCompile("^" + pseudoVersionRegexString + "$")
}

func MustParseTMID(s string) TMID {
	tmid, err := ParseTMID(s)
	if err != nil {
		panic(fmt.Errorf("%w: %s", err, s))
	}
	return tmid
}
func ParseTMID(s string) (TMID, error) {
	if s != strings.ToLower(s) {
		return TMID{}, ErrInvalidId
	}
	if !strings.HasSuffix(s, TMFileExtension) {
		return TMID{}, ErrInvalidId
	}
	s = strings.TrimSuffix(s, TMFileExtension)
	parts := strings.Split(s, "/")
	const minLength = 4
	if len(parts) < minLength {
		return TMID{}, ErrInvalidId
	}
	filename := parts[len(parts)-1]
	name := strings.TrimSuffix(s, "/"+filename)
	ver, err := ParseTMVersion(filename)
	if err != nil {
		return TMID{}, ErrInvalidId
	}

	return TMID{
		Name:    name,
		Version: ver,
	}, nil

}

func ParseTMVersion(s string) (TMVersion, error) {
	if !strings.HasPrefix(s, "v") {
		return TMVersion{}, ErrInvalidVersion
	}
	original, err := semver.NewVersion(s)
	if err != nil {
		return TMVersion{}, err
	}
	submatch := pseudoVersionRegex.FindSubmatch([]byte(original.Prerelease()))
	if len(submatch) != 5 {
		return TMVersion{}, ErrInvalidPseudoVersion
	}

	baseV := fmt.Sprintf("v%d.%d.%d", original.Major(), original.Minor(), original.Patch())
	pre := string(submatch[2])
	if len(pre) != 0 {
		baseV = baseV + "-" + pre
	}
	base, _ := semver.NewVersion(baseV)
	timestamp := string(submatch[3])
	hash := string(submatch[4])
	return TMVersion{
		Base:      base,
		Timestamp: timestamp,
		Hash:      hash,
	}, nil
}

func TMVersionFromOriginal(ver string) TMVersion {
	originalAsSemver, err := semver.NewVersion(ver)
	var baseStr string
	if ver == "" || err != nil {
		baseStr = "v0.0.0"
	} else {
		baseStr = originalAsSemver.String()
		if !strings.HasPrefix(baseStr, "v") {
			baseStr = "v" + baseStr
		}
	}
	newVer, _ := semver.NewVersion(baseStr)
	return TMVersion{
		Base:      newVer,
		Timestamp: "",
		Hash:      "",
	}
}

func (id TMID) Equals(other TMID) bool {
	return id.Name == other.Name &&
		id.Version.BaseString() == other.Version.BaseString() &&
		id.Version.Hash == other.Version.Hash
}
