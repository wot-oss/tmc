package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/buger/jsonparser"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/utils"
)

type FetchName struct {
	Name   string
	Semver string
}

var ErrInvalidFetchName = errors.New("invalid fetch name")

var fetchNameRegex = regexp.MustCompile(`^([\w\-0-9]+(/[\w\-0-9]+){2,})(:(.+))?$`)

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
func ParseAsTMIDOrFetchName(idOrName string) (*model.TMID, *FetchName, error) {
	tmid, err := model.ParseTMID(idOrName)
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

func FetchByTMIDOrName(ctx context.Context, spec model.RepoSpec, idOrName string, restoreId bool) (string, []byte, error, []*repos.RepoAccessError) {
	tmid, fn, err := ParseAsTMIDOrFetchName(idOrName)
	if err != nil {
		return "", nil, err, nil
	}
	if tmid != nil {
		return FetchByTMID(ctx, spec, idOrName, restoreId)
	}
	return FetchByName(ctx, spec, *fn, restoreId)
}

func FetchByTMID(ctx context.Context, spec model.RepoSpec, tmid string, restoreId bool) (string, []byte, error, []*repos.RepoAccessError) {
	rs, err := repos.GetSpecdOrAll(spec)
	if err != nil {
		return "", nil, err, nil
	}

	fetch, bytes, err, accessErrors := rs.Fetch(ctx, tmid)
	if err == nil && restoreId {
		bytes = restoreExternalId(bytes)
	}
	return fetch, bytes, err, accessErrors
}

func restoreExternalId(raw []byte) []byte {
	linksValue, dataType, _, err := jsonparser.Get(raw, "links")
	if err != nil && dataType != jsonparser.NotExist {
		return raw
	}

	if dataType != jsonparser.Array {
		return raw
	}

	var originalId string
	var linksArray []map[string]any

	err = json.Unmarshal(linksValue, &linksArray)
	if err != nil {
		slog.Default().Error("error unmarshalling links", "error", err)
		return raw
	}
	var newLinks []map[string]any
	for _, eLink := range linksArray {
		rel, relOk := eLink["rel"]
		href := utils.JsGetString(eLink, "href")
		if relOk && rel == "original" && href != nil {
			originalId = *href
		} else {
			newLinks = append(newLinks, eLink)
		}
	}
	if len(linksArray) != len(newLinks) { // original id found
		var withLinks []byte
		if len(newLinks) > 0 {
			linksBytes, err := json.Marshal(newLinks)
			if err != nil {
				slog.Default().Error("unexpected marshal error", "error", err)
				return raw
			}
			withLinks, err = jsonparser.Set(raw, linksBytes, "links")
			if err != nil {
				slog.Default().Error("unexpected json set value error", "error", err)
				return raw
			}
		} else {
			withLinks = jsonparser.Delete(raw, "links")
		}
		idBytes, _ := json.Marshal(originalId)

		withId, err := jsonparser.Set(withLinks, idBytes, "id")
		if err != nil {
			slog.Default().Error("unexpected json set value error", "error", err)
			return raw
		}
		return withId
	}

	return raw

}

func FetchByName(ctx context.Context, spec model.RepoSpec, fn FetchName, restoreId bool) (string, []byte, error, []*repos.RepoAccessError) {
	log := slog.Default()
	res, err, errs := NewVersionsCommand().ListVersions(ctx, spec, fn.Name)
	if err != nil {
		return "", nil, err, errs
	}
	versions := make([]model.FoundVersion, len(res))
	copy(versions, res)

	var id string
	var foundIn model.RepoSpec
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
	tmid, bytes, err, _ := FetchByTMID(ctx, foundIn, id, restoreId)
	return tmid, bytes, err, errs
}

func findMostRecentVersion(versions []model.FoundVersion) (string, model.RepoSpec, error) {
	log := slog.Default()
	if len(versions) == 0 {
		err := fmt.Errorf("%w: no versions found", repos.ErrTmNotFound)
		log.Error(err.Error())
		return "", model.EmptySpec, err
	}

	sortFoundVersionsDesc(versions)

	v := versions[0]
	return v.TMID, model.NewSpecFromFoundSource(v.FoundIn), nil
}

func findMostRecentMatchingVersion(versions []model.FoundVersion, ver string) (id string, source model.RepoSpec, err error) {
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
			return "", model.EmptySpec, err
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
		err := fmt.Errorf("%w: no version %s found", repos.ErrTmNotFound, ver)
		log.Error(err.Error())
		return "", model.EmptySpec, err
	}

	// sort the remaining by semver then timestamp in descending order
	sortFoundVersionsDesc(versions)

	// and here's our winner
	v := versions[0]
	return v.TMID, model.NewSpecFromFoundSource(v.FoundIn), nil
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
