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
	versioned, id, err := prepareToImport(c.now, tm, raw, optPath, retriesLeft < 0)
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
			log.Info("Thing Model already exists", "existing-id", errConflict.ExistingId)
			return errConflict.ExistingId, err
		}
		log.Error("error pushing to remote", "error", err)
		return id.String(), err
	}
	log.Info("pushed successfully")
	return id.String(), nil
}

func prepareToImport(now Now, tm *model.ThingModel, raw []byte, optPath string, forceNewId bool) ([]byte, model.TMID, error) {
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
		}
	}

	generatedId, normalized := generateNewId(now, tm, prepared, optPath)
	finalId := idFromFile
	if forceNewId || !generatedId.Equals(idFromFile) {
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
