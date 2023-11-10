package commands

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"log/slog"
	"time"
)

const pseudoVersionTimestampFormat = "20060102150405"

func PushToRemote(remoteName string, filename string) error {
	log := slog.Default()
	remote, err := remotes.Get(remoteName)
	if err != nil {
		log.Error(fmt.Sprintf("could not ìnitialize a remote instance for %s. check config", remoteName), "error", err)
		return err
	}

	abs, raw, err := internal.ReadRequiredFile(filename)
	if err != nil {
		log.Error("couldn't read file", "error", err)
		return err
	}

	tm, err := ValidateThingModel(raw)
	if err != nil {
		log.Error("validation failed", "error", err)
		return err
	}

	versioned, id, err := prepareToImport(tm, raw)
	if err != nil {
		return err
	}

	err = remote.Push(tm, id, versioned)
	if err != nil {
		log.Error("error pushing to remote", "filename", abs, "error", err)
		return err
	}
	log.Info("pushed successfully")
	return nil
}

func prepareToImport(tm *model.ThingModel, raw []byte) ([]byte, model.TMID, error) {
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
		idFromFile, err = model.ParseTMID(origId, tm)
		if err != nil {
			if errors.Is(err, model.ErrInvalidId) {
				// fixme: move id to "original" in prepared
			} else if errors.Is(err, model.ErrVersionDiffers) {
				// version changed - continue to generating new id
			} else {
				// unexpected error
				return nil, model.TMID{}, err
			}
		}
	}

	generatedId := generateNewId(tm, prepared)
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

func generateNewId(tm *model.ThingModel, raw []byte) model.TMID {
	fileForHashing := jsonparser.Delete(raw, "id")
	hasher := sha1.New()
	hasher.Write(fileForHashing)
	hash := hasher.Sum(nil)
	hashStr := fmt.Sprintf("%x", hash[:6])
	ver := model.TMVersionFromOriginal(tm.Version.Model)
	ver.Hash = hashStr
	ver.Timestamp = time.Now().UTC().Format(pseudoVersionTimestampFormat)
	return model.TMID{
		OptionalPath: "", // fixme: pass it down from the command line args
		Author:       tm.Author.Name,
		Manufacturer: tm.Manufacturer.Name,
		Mpn:          tm.Mpn,
		Version:      ver,
	}
}
