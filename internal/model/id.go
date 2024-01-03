package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

var (
	ErrInvalidVersion       = errors.New("invalid version string")
	ErrInvalidPseudoVersion = errors.New("no valid pseudo-version found")
	ErrInvalidId            = errors.New("id invalid")
	ErrVersionDiffers       = errors.New("id has a differing version from given ThingModel")
)

type TMID struct {
	Name         string
	OptionalPath string
	Author       string
	Manufacturer string
	Mpn          string
	Version      TMVersion
}

func NewTMID(author, manufacturer, mpn, optPath string, version TMVersion) TMID {
	id := TMID{
		OptionalPath: optPath,
		Author:       author,
		Manufacturer: manufacturer,
		Mpn:          mpn,
		Version:      version,
	}
	parts := []string{id.Author}
	if id.Manufacturer != id.Author {
		parts = append(parts, id.Manufacturer)
	}
	parts = append(parts, id.Mpn, id.OptionalPath)
	name := JoinSkippingEmpty(parts, "/")
	id.Name = name

	return id

}

type TMVersion struct {
	Base      *semver.Version
	Timestamp string
	Hash      string
}

var pseudoVersionRegex *regexp.Regexp

const pseudoVersionRegexString = "(([0-9A-Za-z\\-]+)\\-)?([0-9]{14})-([0-9a-z]{12})"
const TMFileExtension = ".tm.json"

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
	parts := []string{id.Author}
	if id.Manufacturer != id.Author {
		parts = append(parts, id.Manufacturer)
	}
	parts = append(parts, id.Mpn, id.OptionalPath)
	parts = append(parts, id.Version.String()+TMFileExtension)
	return JoinSkippingEmpty(parts, "/")
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

func MustParseTMID(s string, official bool) TMID {
	tmid, err := ParseTMID(s, official)
	if err != nil {
		panic(err)
	}
	return tmid
}
func ParseTMID(s string, official bool) (TMID, error) {
	if !strings.HasSuffix(s, TMFileExtension) {
		return TMID{}, ErrInvalidId
	}
	s = strings.TrimSuffix(s, TMFileExtension)
	parts := strings.Split(s, "/")
	minLength := 4
	if official {
		minLength = 3
	}
	if len(parts) < minLength {
		return TMID{}, ErrInvalidId
	}
	filename := parts[len(parts)-1]
	parts = parts[0 : len(parts)-1]
	auth := parts[0]
	var manuf, mpn string
	optPathStart := 3
	if official {
		optPathStart = 2
		manuf = auth
		mpn = parts[1]
	} else {
		manuf = parts[1]
		mpn = parts[2]
	}
	optPath := ""
	if len(parts) > optPathStart {
		optPath = strings.Join(parts[optPathStart:], "/")
	}

	ver, err := ParseTMVersion(filename)
	if err != nil {
		return TMID{}, ErrInvalidId
	}

	return TMID{
		Name:         strings.Join(parts, "/"),
		OptionalPath: optPath,
		Author:       auth,
		Manufacturer: manuf,
		Mpn:          mpn,
		Version:      ver,
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
	return id.Author == other.Author &&
		id.Manufacturer == other.Manufacturer &&
		id.Mpn == other.Mpn &&
		id.Version.BaseString() == other.Version.BaseString() &&
		id.Version.Hash == other.Version.Hash
}

func (id TMID) AssertValidFor(tm *ThingModel) error {
	if id.Mpn != tm.Mpn || id.Author != tm.Author.Name || id.Manufacturer != tm.Manufacturer.Name {
		return ErrInvalidId
	}
	if id.Version.Base.Original() != TMVersionFromOriginal(tm.Version.Model).Base.Original() {
		return ErrVersionDiffers
	}

	return nil
}
