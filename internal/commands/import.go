package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/wot-oss/tmc/internal/commands/validate"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/utils"
)

const (
	maxNameLength = 255
)

var ErrTMNameTooLong = fmt.Errorf("TM name too long (max %d allowed)", maxNameLength)

type Now func() time.Time
type ImportCommand struct {
	now Now
}

func NewImportCommand(now Now) *ImportCommand {
	return &ImportCommand{
		now: now,
	}
}

// ImportFile prepares file contents for importing (generates id if necessary, etc.) and imports to repo.
// Returns ImportResult which includes the ID that the TM has been stored under, and error.
// If the repo already contains the same TM, the error will be an instance of repos.ErrTMIDConflict
func (c *ImportCommand) ImportFile(ctx context.Context, raw []byte, repo repos.Repo, opts repos.ImportOptions) (repos.ImportResult, error) {
	log := slog.Default()
	tm, err := validate.ValidateThingModel(raw)
	if err != nil {
		log.Error("validation failed", "error", err)
		return repos.ImportResultFromError(err)
	}
	prepared, id, err := prepareToImport(c.now, tm, raw, opts.OptPath)
	if err != nil {
		return repos.ImportResultFromError(err)
	}

	res, err := repo.Import(ctx, id, prepared, opts)
	if err == nil {
		log.Info("import executed without error")
	}
	return res, err
}

func prepareToImport(now Now, tm *model.ThingModel, raw []byte, optPath string) ([]byte, model.TMID, error) {
	var intermediate = make([]byte, len(raw))
	copy(intermediate, raw)

	// see if there's an id in the file that needs to be preserved
	value, dataType, _, err := jsonparser.Get(intermediate, "id")
	if err != nil && dataType != jsonparser.NotExist {
		return nil, model.TMID{}, err
	}
	var idFromFile model.TMID
	switch dataType {
	case jsonparser.String:
		origId := string(value)
		// check if the id from file is ours or external
		idFromFile, err = model.ParseTMID(origId)
		if err != nil {
			if errors.Is(err, model.ErrInvalidId) {
				// move the existing id to original link if it's external
				intermediate = moveIdToOriginalLink(intermediate, origId)
			} else {
				// ParseTMID returned unexpected error. better stop here
				return nil, model.TMID{}, err
			}
		}
	}

	// generate a new id for the file
	generatedId, normalized := generateNewId(now, tm, intermediate, optPath)
	finalId := idFromFile
	// overwrite the id from file with the newly generated if idFromFile is invalid for given content
	if !generatedId.Equals(idFromFile) {
		finalId = generatedId
	}
	if len(finalId.Name) > maxNameLength {
		return nil, model.TMID{}, fmt.Errorf("%w: %s", ErrTMNameTooLong, finalId.Name)
	}
	idString, _ := json.Marshal(finalId.String())
	final, err := jsonparser.Set(normalized, idString, "id")
	if err != nil {
		return nil, model.TMID{}, err
	}
	return final, finalId, nil
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

// generateNewId normalizes file content for digest calculation and generates a new id for the file with current timestamp
// normalized file has the "id" set to empty string
// returns the generated id and normalized file content that the id was generated for
func generateNewId(now Now, tm *model.ThingModel, raw []byte, optPath string) (model.TMID, []byte) {
	hashStr, raw, _ := CalculateFileDigest(raw) // ignore the error, because the file has been validated already
	ver := model.TMVersionFromOriginal(tm.Version.Model)
	ver.Hash = hashStr
	ver.Timestamp = now().UTC().Format(model.PseudoVersionTimestampFormat)
	return model.NewTMID(tm.Author.Name, tm.Manufacturer.Name, tm.Mpn, sanitizePathForID(optPath), ver), raw
}

func sanitizePathForID(p string) string {
	if p == "" {
		return p
	}
	p = strings.Replace(p, "\\", "/", -1)
	p = path.Clean(p)
	p, _ = strings.CutPrefix(p, "/")
	p, _ = strings.CutSuffix(p, "/")

	parts := strings.Split(p, "/")
	for i, part := range parts {
		parts[i] = utils.SanitizeName(part)
	}
	p = strings.Join(parts, "/")
	return p
}
