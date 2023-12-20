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

var fetchNameRegex = regexp.MustCompile(`^([^:]+)(:(.+))?$`)

func (fn *FetchName) Parse(fetchName string) error {
	// Find submatches in the input string
	matches := fetchNameRegex.FindStringSubmatch(fetchName)

	// Check if there are enough submatches
	if len(matches) < 2 {
		msg := fmt.Sprintf("Invalid name format: %s - Must be NAME[:SEMVER|DIGEST]", fetchName)
		slog.Default().Error(msg)
		return fmt.Errorf(msg)
	}

	// Extract values from submatches
	fn.Name = matches[1]
	if len(matches) > 3 {
		fn.SemVerOrDigest = matches[3]
	}
	return nil
}

func FetchThingByName(fn *FetchName, remoteName string) ([]byte, error) {
	log := slog.Default()
	tocThing, err := ListVersions(remoteName, fn.Name)
	if err != nil {
		return nil, err
	}

	var id string
	var foundIn string
	var thing []byte
	// Just the name specified: fetch most recent
	if len(fn.SemVerOrDigest) == 0 {
		id, foundIn, err = findMostRecentVersion(tocThing.Versions)
		if err != nil {
			return nil, err
		}
	} else if fetchVersion, err := semver.NewVersion(fn.SemVerOrDigest); err == nil {
		id, foundIn, err = findMostRecentTimeStamp(tocThing.Versions, fetchVersion)
		if err != nil {
			return nil, err
		}
	} else {
		id, foundIn, err = findDigest(tocThing.Versions, fn.SemVerOrDigest)
		if err != nil {
			return nil, err
		}
	}

	// TODO: cannot use IsOfficial of ThingModel here
	official := utils.ToTrimmedLower(tocThing.Author.Name) == utils.ToTrimmedLower(tocThing.Manufacturer.Name)
	tmid, err := model.ParseTMID(id, official)

	log.Debug(fmt.Sprintf("fetching %v from %s", tmid, foundIn))

	remote, err := remotes.Get(foundIn)
	if err != nil {
		return nil, err
	}
	thing, err = remote.Fetch(tmid)
	if err != nil {
		msg := fmt.Sprintf("No thing model found for %s", fn)
		slog.Default().Error(msg)
		return nil, errors.New(msg)
	}
	return thing, nil
}

func findMostRecentVersion(versions []model.FoundVersion) (id, source string, err error) {
	log := slog.Default()
	if len(versions) == 0 {
		msg := "No versions found"
		log.Error(msg)
		return "", "", errors.New(msg)
	}

	latestVersion, _ := semver.NewVersion("v0.0.0")
	var latestTimeStamp time.Time

	for _, version := range versions {
		// TODO: use StrictNewVersion
		currentVersion, err := semver.NewVersion(version.Version.Model)
		if err != nil {
			log.Error(err.Error())
			return "", "", err
		}
		if currentVersion.GreaterThan(latestVersion) {
			latestVersion = currentVersion
			latestTimeStamp, err = time.Parse(pseudoVersionTimestampFormat, version.TimeStamp)
			id = version.TMID
			source = version.FoundIn
			continue
		}
		if currentVersion.Equal(latestVersion) {
			currentTimeStamp, err := time.Parse(pseudoVersionTimestampFormat, version.TimeStamp)
			if err != nil {
				log.Error(err.Error())
				return "", "", err
			}
			if currentTimeStamp.After(latestTimeStamp) {
				latestTimeStamp = currentTimeStamp
				id = version.TMID
				source = version.FoundIn
				continue
			}
		}
	}
	return id, source, nil
}

func findMostRecentTimeStamp(versions []model.FoundVersion, ver *semver.Version) (id, source string, err error) {
	log := slog.Default()
	if len(versions) == 0 {
		msg := "No versions found"
		log.Error(msg)
		return "", "", errors.New(msg)
	}
	var latestTimeStamp time.Time

	for _, version := range versions {
		// TODO: use StrictNewVersion
		currentVersion, err := semver.NewVersion(version.Version.Model)
		if err != nil {
			log.Error(err.Error())
			return "", "", err
		}

		if !currentVersion.Equal(ver) {
			continue
		}
		currentTimeStamp, err := time.Parse(pseudoVersionTimestampFormat, version.TimeStamp)
		if err != nil {
			log.Error(err.Error())
			return "", "", err
		}
		if currentTimeStamp.After(latestTimeStamp) {
			latestTimeStamp = currentTimeStamp
			id = version.TMID
			source = version.FoundIn
			continue
		}
	}
	if len(id) == 0 {
		msg := fmt.Sprintf("No version %s found", ver.String())
		log.Error(msg)
		return "", "", errors.New(msg)
	}
	return id, source, nil
}

func findDigest(versions []model.FoundVersion, digest string) (id, source string, err error) {
	log := slog.Default()
	if len(versions) == 0 {
		msg := "No versions found"
		log.Error(msg)
		return "", "", errors.New(msg)
	}

	digest = utils.ToTrimmedLower(digest)
	for _, version := range versions {
		// TODO: how to know if it is official?
		tmid, err := model.ParseTMID(version.TMID, false)
		if err != nil {
			log.Error(fmt.Sprintf("Unable to parse TMID from %s", version.TMID))
			return "", "", err
		}
		if tmid.Version.Hash == digest {
			return version.TMID, version.FoundIn, nil
		}
	}
	msg := fmt.Sprintf("No thing model found for digest %s", digest)
	log.Error(msg)
	return "", "", errors.New(msg)
}
