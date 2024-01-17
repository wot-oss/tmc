package commands

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
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

func (c *FetchCommand) FetchByTMIDOrName(spec remotes.RepoSpec, idOrName string) (string, []byte, error) {
	_, err := model.ParseTMID(idOrName, true)
	if err == nil {
		id, tm, err := c.FetchByTMID(spec, idOrName)
		if !errors.Is(err, ErrTmNotFound) {
			return id, tm, err
		}
	}

	fn, err := ParseFetchName(idOrName)
	if err != nil {
		slog.Default().Info("could not parse as either TMID or fetch name", "idOrName", idOrName)
		return "", nil, err
	}
	return c.FetchByName(spec, fn)
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
	} else if fetchVersion, err := semver.NewVersion(fn.SemVerOrDigest); err == nil {
		id, foundIn, err = findMostRecentTimeStamp(tocThing.Versions, fetchVersion)
		if err != nil {
			return "", nil, err
		}
	} else {
		id, foundIn, err = findDigest(tocThing.Versions, fn.SemVerOrDigest)
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

func findMostRecentTimeStamp(versions []model.FoundVersion, ver *semver.Version) (id string, source remotes.RepoSpec, err error) {
	log := slog.Default()
	if len(versions) == 0 {
		msg := "No versions found"
		log.Error(msg)
		return "", remotes.EmptySpec, errors.New(msg)
	}
	var latestTimeStamp time.Time

	for _, version := range versions {
		// TODO: use StrictNewVersion
		currentVersion, err := semver.NewVersion(version.Version.Model)
		if err != nil {
			log.Error(err.Error())
			return "", remotes.EmptySpec, err
		}

		if !currentVersion.Equal(ver) {
			continue
		}
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
	if len(id) == 0 {
		msg := fmt.Sprintf("No version %s found", ver.String())
		log.Error(msg)
		return "", remotes.EmptySpec, errors.New(msg)
	}
	return id, source, nil
}

func findDigest(versions []model.FoundVersion, digest string) (id string, source remotes.RepoSpec, err error) {
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
