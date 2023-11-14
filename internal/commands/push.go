package commands

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kennygrant/sanitize"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var now = time.Now

const pseudoVersionTimestampFormat = "20060102150405"

func PushToRemote(filename string, remoteName string, optPath string, optTree bool) error {
	optPath = sanitizePath(optPath)

	log := slog.Default()
	remote, err := remotes.Get(remoteName)
	if err != nil {
		log.Error(fmt.Sprintf("could not ìnitialize a remote instance for %s. check config", remoteName), "error", err)
		return err
	}

	abs, err := filepath.Abs(filename)
	if err != nil {
		log.Error("error expanding file name", "filename", filename, "error", err)
		return err
	}

	stat, err := os.Stat(abs)
	if err != nil {
		log.Error("cannot read file or directory", "filename", filename, "error", err)
		return err
	}
	if stat.IsDir() {
		return pushDirectory(abs, remote, optPath, optTree)
	} else {
		return pushFile(filename, remote, optPath)
	}
}

func sanitizePath(path string) string {
	if path == "" {
		return path
	}
	p := sanitize.Path(path)
	p, _ = strings.CutPrefix(p, "/")
	p, _ = strings.CutSuffix(p, "/")
	return p
}

func pushDirectory(absDirname string, remote remotes.Remote, optPath string, optTree bool) error {
	err := filepath.WalkDir(absDirname, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		if err != nil {
			return err
		}

		if optTree {
			optPath = filepath.Dir(strings.TrimPrefix(path, absDirname))
		}

		err = pushFile(path, remote, optPath)

		return err
	})

	return err

}

func pushFile(filename string, remote remotes.Remote, optPath string) error {
	log := slog.Default()
	_, raw, err := internal.ReadRequiredFile(filename)
	if err != nil {
		log.Error("couldn't read file", "filename", filename, "error", err)
		return err
	}
	_, err = PushFile(raw, remote, optPath)
	if err != nil {
		return err
	}
	return nil
}
func PushFile(raw []byte, remote remotes.Remote, optPath string) (model.TMID, error) {
	log := slog.Default()
	tm, err := ValidateThingModel(raw)
	if err != nil {
		log.Error("validation failed", "error", err)
		return model.TMID{}, err
	}

	versioned, id, err := prepareToImport(tm, raw, optPath)
	if err != nil {
		return model.TMID{}, err
	}

	err = remote.Push(id, versioned)
	if err != nil {
		var errExists *remotes.ErrTMExists
		if errors.As(err, &errExists) {
			log.Info("Thing Model already exists", "existing-id", errExists.ExistingId)
			return errExists.ExistingId, nil
		}
		log.Error("error pushing to remote", "error", err)
		return id, err
	}
	log.Info("pushed successfully")
	return id, nil
}

func prepareToImport(tm *model.ThingModel, raw []byte, optPath string) ([]byte, model.TMID, error) {
	manuf := tm.Manufacturer.Name
	auth := tm.Author.Name
	if tm == nil || len(auth) == 0 || len(manuf) == 0 || len(tm.Mpn) == 0 {
		return nil, model.TMID{}, errors.New("ThingModel cannot be nil or have empty mandatory fields")
	}
	value, dataType, _, err := jsonparser.Get(raw, "id")
	if err != nil && dataType != jsonparser.NotExist {
		return nil, model.TMID{}, err
	}
	var prepared = make([]byte, len(raw))
	copy(prepared, raw)
	var idFromFile model.TMID
	switch dataType {
	case jsonparser.String:
		origId := string(value)
		idFromFile, err = model.ParseTMID(origId, tm.Author.Name == tm.Manufacturer.Name)
		if err != nil {
			if errors.Is(err, model.ErrInvalidId) || idFromFile.AssertValidFor(tm) != nil {
				prepared = moveIdToOriginalLink(prepared, origId)
			} else {
				return nil, model.TMID{}, err
			}
		} else {

		}
	}

	generatedId := generateNewId(tm, prepared, optPath)
	finalId := idFromFile
	if !generatedId.Equals(idFromFile) {
		finalId = generatedId
		idString, _ := json.Marshal(generatedId.String())
		prepared, err = jsonparser.Set(prepared, idString, "id")
		if err != nil {
			return nil, model.TMID{}, err
		}
	}

	return prepared, finalId, nil
}

func moveIdToOriginalLink(raw []byte, id string) []byte {
	linksValue, dataType, _, err := jsonparser.Get(raw, "links")
	if err != nil && dataType != jsonparser.NotExist {
		return raw
	}

	link := map[string]any{"href": id, "rel": "original"}
	var linksArray []map[string]any

	switch dataType {
	case jsonparser.NotExist:
		// put "links" : [{"href": "{{id}}", "rel": "original"}]
		linksArray = []map[string]any{link}
	case jsonparser.Array:
		err := json.Unmarshal(linksValue, &linksArray)
		if err != nil {
			slog.Default().Error("error unmarshalling links", "error", err)
			return raw
		}
		for _, eLink := range linksArray {
			if rel, ok := eLink["rel"]; ok && rel == "original" {
				// link to original found => abort
				return raw
			}
		}
		linksArray = append(linksArray, link)

	default:
		// unexpected type of "links"
		slog.Default().Warn(fmt.Sprintf("unexpected type of links %v", dataType))
		return raw
	}

	linksBytes, err := json.Marshal(linksArray)
	if err != nil {
		slog.Default().Error("unexpected marshal error", "error", err)
		return raw
	}
	raw, err = jsonparser.Set(raw, linksBytes, "links")

	return raw
}

func generateNewId(tm *model.ThingModel, raw []byte, optPath string) model.TMID {
	fileForHashing := jsonparser.Delete(raw, "id")
	hasher := sha1.New()
	hasher.Write(fileForHashing)
	hash := hasher.Sum(nil)
	hashStr := fmt.Sprintf("%x", hash[:6])
	ver := model.TMVersionFromOriginal(tm.Version.Model)
	ver.Hash = hashStr
	ver.Timestamp = now().UTC().Format(pseudoVersionTimestampFormat)
	return model.TMID{
		OptionalPath: optPath,
		Author:       tm.Author.Name,
		Manufacturer: tm.Manufacturer.Name,
		Mpn:          tm.Mpn,
		Version:      ver,
	}
}
