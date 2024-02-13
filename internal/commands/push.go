package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/kennygrant/sanitize"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands/validate"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

const maxPushRetries = 3

type Now func() time.Time
type PushCommand struct {
	now Now
}

func NewPushCommand(now Now) *PushCommand {
	return &PushCommand{
		now: now,
	}
}

// PushFile prepares file contents for pushing (generates id if necessary, etc.) and pushes to remote.
// Returns the ID that the TM has been stored under, and error.
// If the remote already contains the same TM, returns the id of the existing TM and an instance of remotes.ErrTMIDConflict
func (c *PushCommand) PushFile(raw []byte, remote remotes.Remote, optPath string) (string, error) {
	log := slog.Default()
	tm, err := validate.ValidateThingModel(raw)
	if err != nil {
		log.Error("validation failed", "error", err)
		return "", err
	}
	retriesLeft := maxPushRetries
RETRY:
	retriesLeft--
	versioned, id, err := prepareToImport(c.now, tm, raw, optPath)
	if err != nil {
		return "", err
	}

	err = remote.Push(id, versioned)
	if err != nil {
		var errConflict *remotes.ErrTMIDConflict
		if errors.As(err, &errConflict) {
			if errConflict.Type == remotes.IdConflictSameTimestamp {
				if retriesLeft >= 0 {
					time.Sleep(1 * time.Second) // sleep 1 sec to get a different timestamp in id
					goto RETRY
				}
				return errConflict.ExistingId, err
			}
			log.Info("Thing Model conflicts with existing", "id", id, "existing-id", errConflict.ExistingId, "conflictType", errConflict.Type)
			return errConflict.ExistingId, err
		}
		log.Error("error pushing to remote", "error", err)
		return id.String(), err
	}
	log.Info("pushed successfully")
	return id.String(), nil
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
		idFromFile, err = model.ParseTMID(origId, tm.Author.Name == tm.Manufacturer.Name)
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
		parts[i] = sanitize.BaseName(part)
	}
	p = strings.Join(parts, "/")
	return p
}
