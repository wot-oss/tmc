package model

import (
	"errors"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"regexp"
	"strings"
)

var (
	ErrInvalidVersion       = errors.New("invalid version string")
	ErrInvalidPseudoVersion = errors.New("no valid pseudo-version found")
	ErrInvalidId            = errors.New("id invalid [for given ThingModel]")
	ErrVersionDiffers       = errors.New("id has a differing version from given ThingModel")
)

type TMID struct {
	OptionalPath string
	Author       string
	Manufacturer string
	Mpn          string
	Version      TMVersion
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

func ParseTMID(s string, tm *ThingModel) (TMID, error) {
	manuf := tm.Manufacturer.Name
	auth := tm.Author.Name
	if tm == nil || len(auth) == 0 || len(manuf) == 0 || len(tm.Mpn) == 0 {
		return TMID{}, errors.New("ThingModel cannot be nil or have empty mandatory fields")
	}
	if !strings.HasSuffix(s, TMFileExtension) {
		return TMID{}, ErrInvalidId
	}
	s = strings.TrimSuffix(s, TMFileExtension)
	official := auth == manuf
	parts := strings.Split(s, "/")
	if len(parts) < 3 {
		return TMID{}, ErrInvalidId
	}
	filename := parts[len(parts)-1]
	parts = parts[0 : len(parts)-1]
	optPathStart := 3
	if official {
		optPathStart = 2
		if parts[0] != manuf {
			return TMID{}, ErrInvalidId
		}
	} else {
		if parts[0] != auth || parts[1] != manuf {
			return TMID{}, ErrInvalidId
		}
	}
	if parts[optPathStart-1] != tm.Mpn {
		return TMID{}, ErrInvalidId
	}
	optPath := ""
	if len(parts) > optPathStart {
		optPath = strings.Join(parts[optPathStart:], "/")
	}

	ver, err := ParseTMVersion(filename)
	if err != nil {
		return TMID{}, ErrInvalidId
	}

	if ver.Base.Original() != TMVersionFromOriginal(tm.Version.Model).Base.Original() {
		return TMID{}, ErrVersionDiffers
	}

	return TMID{
		OptionalPath: optPath,
		Author:       auth,
		Manufacturer: manuf,
		Mpn:          tm.Mpn,
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
