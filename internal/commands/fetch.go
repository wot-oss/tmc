package commands

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

type FetchName struct {
	Name   string
	Semver string
}

var ErrInvalidFetchName = errors.New("invalid fetch name")

var fetchNameRegex = regexp.MustCompile(`^([\w\-0-9]+(/[\w\-0-9]+)+)(:(.+))?$`)

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

type FetchCommand struct {
	remoteMgr remotes.RemoteManager
}

func NewFetchCommand(manager remotes.RemoteManager) *FetchCommand {
	return &FetchCommand{
		remoteMgr: manager,
	}
}

// ParseAsTMIDOrFetchName parses idOrName as model.TMID. If that fails, parses it as FetchName.
// Returns error is idOrName is not valid as either. Only one of returned pointers may be not nil
func ParseAsTMIDOrFetchName(idOrName string) (*model.TMID, *FetchName, error) {
	tmid, err := model.ParseTMID(idOrName, true)
	if err == nil {
		return &tmid, nil, nil
	}
	fn, err := ParseFetchName(idOrName)
	if err == nil {
		return nil, &fn, nil
	}

	slog.Default().Info("could not parse as either TMID or fetch name", "idOrName", idOrName)
	return nil, nil, err
}

func (c *FetchCommand) FetchByTMIDOrName(spec remotes.RepoSpec, idOrName string) (string, []byte, error, []*remotes.RepoAccessError) {
	tmid, fn, err := ParseAsTMIDOrFetchName(idOrName)
	if err != nil {
		return "", nil, err, nil
	}
	if tmid != nil {
		return c.FetchByTMID(spec, idOrName)
	}
	return c.FetchByName(spec, *fn)
}

func (c *FetchCommand) FetchByTMID(spec remotes.RepoSpec, tmid string) (string, []byte, error, []*remotes.RepoAccessError) {
	rs, err := remotes.GetSpecdOrAll(c.remoteMgr, spec)
	if err != nil {
		return "", nil, err, nil
	}

	return rs.Fetch(tmid)
}
func (c *FetchCommand) FetchByName(spec remotes.RepoSpec, fn FetchName) (string, []byte, error, []*remotes.RepoAccessError) {
	log := slog.Default()
	tocVersions, err, errs := NewVersionsCommand(c.remoteMgr).ListVersions(spec, fn.Name)
	if err != nil {
		return "", nil, err, errs
	}
	versions := make([]model.FoundVersion, len(tocVersions))
	copy(versions, tocVersions)

	var id string
	var foundIn remotes.RepoSpec
	// Just the name specified: fetch most recent
	if len(fn.Semver) == 0 {
		id, foundIn, err = findMostRecentVersion(versions)
		if err != nil {
			return "", nil, err, errs
		}
	} else {
		if _, err := semver.NewVersion(fn.Semver); err == nil {
			id, foundIn, err = findMostRecentMatchingVersion(versions, fn.Semver)
			if err != nil {
				return "", nil, err, errs
			}
		} else {
			return "", nil, err, errs
		}
	}

	log.Debug(fmt.Sprintf("fetching %v from %s", id, foundIn))
	tmid, bytes, err, _ := c.FetchByTMID(foundIn, id)
	return tmid, bytes, err, errs
}

func findMostRecentVersion(versions []model.FoundVersion) (string, remotes.RepoSpec, error) {
	log := slog.Default()
	if len(versions) == 0 {
		err := fmt.Errorf("%w: no versions found", remotes.ErrTmNotFound)
		log.Error(err.Error())
		return "", remotes.EmptySpec, err
	}

	sortFoundVersionsDesc(versions)

	v := versions[0]
	return v.TMID, remotes.NewSpecFromFoundSource(v.FoundIn), nil
}

func findMostRecentMatchingVersion(versions []model.FoundVersion, ver string) (id string, source remotes.RepoSpec, err error) {
	log := slog.Default()
	ver, _ = strings.CutPrefix(ver, "v")

	// figure out how to match versions with ver
	var matcher func(*semver.Version) bool
	dots := strings.Count(ver, ".")
	if dots == 2 { // ver contains major.minor.patch
		sv := semver.MustParse(ver)
		matcher = sv.Equal
	} else { // at least one semver part is missing in ver
		c, err := semver.NewConstraint(fmt.Sprintf("~%s", ver))
		if err != nil {
			log.Error("couldn't parse semver constraint", "error", err)
			return "", remotes.EmptySpec, err
		}
		matcher = c.Check
	}

	// delete versions not matching ver from the list
	versions = slices.DeleteFunc(versions, func(version model.FoundVersion) bool {
		semVersion, err := semver.NewVersion(version.Version.Model)
		if err != nil {
			log.Error(err.Error())
			return false
		}
		matches := matcher(semVersion)
		return !matches
	})

	// see if anything remained
	if len(versions) == 0 {
		err := fmt.Errorf("%w: no version %s found", remotes.ErrTmNotFound, ver)
		log.Error(err.Error())
		return "", remotes.EmptySpec, err
	}

	// sort the remaining by semver then timestamp in descending order
	sortFoundVersionsDesc(versions)

	// and here's our winner
	v := versions[0]
	return v.TMID, remotes.NewSpecFromFoundSource(v.FoundIn), nil
}

// sortFoundVersionsDesc sorts by semver then timestamp in descending order, ie. from newest to oldest
func sortFoundVersionsDesc(versions []model.FoundVersion) {
	slices.SortStableFunc(versions, func(a, b model.FoundVersion) int {
		av := semver.MustParse(a.Version.Model)
		bv := semver.MustParse(b.Version.Model)
		vc := bv.Compare(av)
		if vc != 0 {
			return vc
		}
		return strings.Compare(b.TimeStamp, a.TimeStamp) // our timestamps can be compared lexicographically
	})
}
