package commands

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

type FetchName struct {
	Name           string
	SemVerOrDigest string
}

var ErrTmNotFound = errors.New("TM not found")

var fetchNameRegex = regexp.MustCompile(`^([\w\-0-9]+(/[\w\-0-9]+)+)(:(.+))?$`)

func ParseFetchName(fetchName string) (FetchName, error) {
	// Find submatches in the input string
	matches := fetchNameRegex.FindStringSubmatch(fetchName)

	// Check if there are enough submatches
	if len(matches) < 2 {
		msg := fmt.Sprintf("Invalid name format: %s - Must be NAME[:SEMVER|DIGEST]", fetchName)
		slog.Default().Error(msg)
		return FetchName{}, fmt.Errorf(msg)
	}

	fn := FetchName{}
	// Extract values from submatches
	fn.Name = matches[1]
	if len(matches) > 4 {
		fn.SemVerOrDigest = matches[4]
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

func (c *FetchCommand) FetchByTMIDOrName(spec remotes.RepoSpec, idOrName string) (string, []byte, error) {
	tmid, fn, err := ParseAsTMIDOrFetchName(idOrName)
	if err != nil {
		return "", nil, err
	}
	if tmid != nil {
		return c.FetchByTMID(spec, idOrName)
	}
	return c.FetchByName(spec, *fn)
}

func (c *FetchCommand) FetchByTMID(spec remotes.RepoSpec, tmid string) (string, []byte, error) {
	rs, err := remotes.GetSpecdOrAll(c.remoteMgr, spec)
	if err != nil {
		return "", nil, err
	}

	for _, r := range rs {
		id, thing, err := r.Fetch(tmid)
		if err == nil {
			return id, thing, nil
		}
	}

	msg := fmt.Sprintf("No thing model found for %v", tmid)
	slog.Default().Error(msg)
	return "", nil, ErrTmNotFound

}
func (c *FetchCommand) FetchByName(spec remotes.RepoSpec, fn FetchName) (string, []byte, error) {
	log := slog.Default()
	tocThing, err := NewVersionsCommand(c.remoteMgr).ListVersions(spec, fn.Name)
	if err != nil {
		return "", nil, err
	}

	var id string
	var foundIn remotes.RepoSpec
	// Just the name specified: fetch most recent
	if len(fn.SemVerOrDigest) == 0 {
		id, foundIn, err = findMostRecentVersion(tocThing.Versions)
		if err != nil {
			return "", nil, err
		}
	} else if _, err := semver.NewVersion(fn.SemVerOrDigest); err == nil {
		id, foundIn, err = findMostRecentMatchingVersion(tocThing.Versions, fn.SemVerOrDigest)
		if err != nil {
			return "", nil, err
		}
	} else {
		id, foundIn, err = findByDigest(tocThing.Versions, fn.SemVerOrDigest)
		if err != nil {
			return "", nil, err
		}
	}

	log.Debug(fmt.Sprintf("fetching %v from %s", id, foundIn))
	return c.FetchByTMID(foundIn, id)
}

func findMostRecentVersion(versions []model.FoundVersion) (id string, source remotes.RepoSpec, err error) {
	log := slog.Default()
	if len(versions) == 0 {
		msg := "No versions found"
		log.Error(msg)
		return "", remotes.EmptySpec, errors.New(msg)
	}

	latestVersion, _ := semver.NewVersion("v0.0.0")
	var latestTimeStamp time.Time

	for _, version := range versions {
		// TODO: use StrictNewVersion
		currentVersion, err := semver.NewVersion(version.Version.Model)
		if err != nil {
			log.Error(err.Error())
			return "", remotes.EmptySpec, err
		}
		if currentVersion.GreaterThan(latestVersion) {
			latestVersion = currentVersion
			latestTimeStamp, err = time.Parse(model.PseudoVersionTimestampFormat, version.TimeStamp)
			id = version.TMID
			source = remotes.NewSpecFromFoundSource(version.FoundIn)
			continue
		}
		if currentVersion.Equal(latestVersion) {
			currentTimeStamp, err := time.Parse(model.PseudoVersionTimestampFormat, version.TimeStamp)
			if err != nil {
				log.Error(err.Error())
				return "", remotes.EmptySpec, err
			}
			if currentTimeStamp.After(latestTimeStamp) {
				latestTimeStamp = currentTimeStamp
				id = version.TMID
				source = remotes.NewSpecFromFoundSource(version.FoundIn)
				continue
			}
		}
	}
	return id, source, nil
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
		msg := fmt.Sprintf("No version %s found", ver)
		log.Error(msg)
		return "", remotes.EmptySpec, errors.New(msg)
	}

	// sort the remaining by semver then timestamp in descending order
	slices.SortStableFunc(versions, func(a, b model.FoundVersion) int {
		av := semver.MustParse(a.Version.Model)
		bv := semver.MustParse(b.Version.Model)
		vc := bv.Compare(av)
		if vc != 0 {
			return vc
		}
		return strings.Compare(b.TimeStamp, a.TimeStamp) // our timestamps can be compared lexicographically
	})

	// and here's our winner
	v := versions[0]
	return v.TMID, remotes.NewSpecFromFoundSource(v.FoundIn), nil
}

func findByDigest(versions []model.FoundVersion, digest string) (id string, source remotes.RepoSpec, err error) {
	log := slog.Default()
	if len(versions) == 0 {
		msg := "No versions found"
		log.Error(msg)
		return "", remotes.EmptySpec, errors.New(msg)
	}

	digest = utils.ToTrimmedLower(digest)
	for _, version := range versions {
		// TODO: how to know if it is official?
		tmid, err := model.ParseTMID(version.TMID, false)
		if err != nil {
			log.Error(fmt.Sprintf("Unable to parse TMID from %s", version.TMID))
			return "", remotes.EmptySpec, err
		}
		if tmid.Version.Hash == digest {
			return version.TMID, remotes.NewSpecFromFoundSource(version.FoundIn), nil
		}
	}
	msg := fmt.Sprintf("No thing model found for digest %s", digest)
	log.Error(msg)
	return "", remotes.EmptySpec, errors.New(msg)
}
