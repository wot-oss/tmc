package remotes

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

const defaultDirPermissions = 0775
const defaultFilePermissions = 0664

type FileRemote struct {
	root string
}

type ErrTMExists struct {
	ExistingId model.TMID
}

func (e *ErrTMExists) Error() string {
	return fmt.Sprintf("Thing Model already exists under id: %v", e.ExistingId)
}

func NewFileRemote(config map[string]any) (*FileRemote, error) {
	loc := config[KeyRemoteLoc]
	locString, ok := loc.(string)
	if !ok {
		return nil, fmt.Errorf("invalid file remote config. loc is either not found or not a string: %v", loc)
	}
	rootPath, err := utils.ExpandHome(locString)
	if err != nil {
		return nil, err
	}
	return &FileRemote{
		root: rootPath,
	}, nil
}

func (f *FileRemote) Push(id model.TMID, raw []byte) error {
	if len(raw) == 0 {
		return errors.New("nothing to write")
	}
	fullPath, dir := f.filenames(id)
	err := os.MkdirAll(dir, defaultDirPermissions)
	if err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	if found, existingId := f.getExistingID(id); found {
		slog.Default().Info(fmt.Sprintf("TM already exists under ID %v", existingId))
		return &ErrTMExists{ExistingId: existingId}
	}

	err = os.WriteFile(fullPath, raw, defaultFilePermissions)
	if err != nil {
		return fmt.Errorf("could not write TM to catalog: %v", err)
	}
	slog.Default().Info("saved Thing Model file", "filename", fullPath)

	return nil
}

func (f *FileRemote) filenames(id model.TMID) (string, string) {
	fullPath := filepath.Join(f.root, id.String())
	dir := filepath.Dir(fullPath)
	return fullPath, dir
}

func (f *FileRemote) getExistingID(id model.TMID) (bool, model.TMID) {
	fullName, dir := f.filenames(id)
	// try full name as given
	if _, err := os.Stat(fullName); err == nil {
		return true, id
	}
	// try without timestamp
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, model.TMID{}
	}
	existingTMVersions := findTMFileEntriesByBaseVersion(entries, id.Version)
	if len(existingTMVersions) > 0 && existingTMVersions[0].Hash == id.Version.Hash {
		ret := id
		ret.Version = existingTMVersions[0]
		return true, ret
	}

	return false, model.TMID{}
}

// findTMFileEntriesByBaseVersion finds directory entries that correspond to TM file names, converts those to TMVersions,
// filters out those that have a differing base version from the one given as argument, and sorts the remaining in
// descending order
func findTMFileEntriesByBaseVersion(entries []os.DirEntry, version model.TMVersion) []model.TMVersion {
	baseString := version.BaseString()
	var res []model.TMVersion
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		ver, err := model.ParseTMVersion(strings.TrimSuffix(e.Name(), model.TMFileExtension))
		if err != nil {
			continue
		}

		if baseString == ver.BaseString() {
			res = append(res, ver)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return strings.Compare(res[i].String(), res[j].String()) > 0
	})
	return res
}

func (f *FileRemote) Fetch(id model.TMID) ([]byte, error) {
	exists, actualId := f.getExistingID(id)
	if !exists {
		return nil, os.ErrNotExist
	}
	actualFilename, _ := f.filenames(actualId)
	return os.ReadFile(actualFilename)
}

func (f *FileRemote) CreateToC() error {
	return createTOC(f.root)
}

func (f *FileRemote) List(filter string) (model.TOC, error) {
	log := slog.Default()
	if len(filter) == 0 {
		log.Debug("Creating list")
	} else {
		log.Debug(fmt.Sprintf("Creating list with filter '%s'", filter))
	}

	data, err := os.ReadFile(filepath.Join(f.root, TOCFilename))
	if err != nil {
		return model.TOC{}, errors.New("No toc found. Run `create-toc` for this remote.")
	}

	var toc model.TOC
	err = json.Unmarshal(data, &toc)
	toc.Filter(filter)
	if err != nil {
		return model.TOC{}, err
	}
	return toc, nil
}

func (f *FileRemote) Versions(name string) (model.TOCEntry, error) {
	log := slog.Default()
	if len(name) == 0 {
		log.Error("Please specify a name to show the TM.")
		return model.TOCEntry{}, errors.New("please specify a name to show the TM")
	}
	toc, err := f.List("")
	if err != nil {
		return model.TOCEntry{}, err
	}
	name = strings.TrimSpace(name)

	tocThing := toc.FindByName(name)
	if tocThing == nil {
		msg := fmt.Sprintf("No thing model found for name: %s", name)
		log.Error(msg)
		return model.TOCEntry{}, errors.New(msg)
	}

	return *tocThing, nil
}
func createFileRemoteConfig(dirName string, bytes []byte) (map[string]any, error) {
	if dirName != "" {
		absDir, err := makeAbs(dirName)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			KeyRemoteType: RemoteTypeFile,
			KeyRemoteLoc:  absDir,
		}, nil
	} else {
		rc, err := AsRemoteConfig(bytes)
		if err != nil {
			return nil, err
		}
		if rType, ok := rc[KeyRemoteType]; ok {
			if rType != RemoteTypeFile {
				return nil, fmt.Errorf("invalid json config. type must be \"file\" or absent")
			}
		}
		rc[KeyRemoteType] = RemoteTypeFile
		l, ok := rc[KeyRemoteLoc]
		if !ok {
			return nil, fmt.Errorf("invalid json config. must have key \"loc\"")
		}
		ls, ok := l.(string)
		if !ok {
			return nil, fmt.Errorf("invalid json config. url must be a string")
		}
		la, err := makeAbs(ls)
		if err != nil {
			return nil, err
		}
		rc[KeyRemoteLoc] = la
		return rc, nil
	}
}

func makeAbs(dir string) (string, error) {
	if filepath.IsAbs(dir) {
		return dir, nil
	} else {
		if !strings.HasPrefix(dir, "~") {
			var err error
			dir, err = filepath.Abs(dir)
			if err != nil {
				return "", err
			}
		}
		return dir, nil
	}
}

const TMExt = ".tm.json"
const TOCFilename = "tm-catalog.toc.json"

func createTOC(rootPath string) error {
	// Prepare data collection for logging stats
	var log = slog.Default()
	fileCount := 0
	start := time.Now()

	newTOC := model.TOC{
		Meta: model.TOCMeta{Created: time.Now()},
		Data: []*model.TOCEntry{},
	}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if strings.HasSuffix(info.Name(), TMExt) {
				thingMeta, err := getThingMetadata(path)
				if err != nil {
					msg := "Failed to extract metadata from file %s with error:"
					msg = fmt.Sprintf(msg, path)
					log.Error(msg)
					log.Error(err.Error())
					log.Error("The file will be excluded from the table of contents.")
					return nil
				}
				err = newTOC.Insert(&thingMeta)
				if err != nil {
					log.Error(fmt.Sprintf("Failed to insert %s into toc:", path))
					log.Error(err.Error())
					log.Error("The file will be excluded from the table of contents.")
					return nil
				}
				fileCount++
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	duration := time.Now().Sub(start)
	// Ignore error as we are sure our struct does not contain channel,
	// complex or function values that would throw an error.
	newTOCJson, _ := json.MarshalIndent(newTOC, "", "  ")
	err = saveTOC(rootPath, newTOCJson)
	msg := "Created table of content with %d entries in %s "
	msg = fmt.Sprintf(msg, fileCount, duration.String())
	log.Info(msg)
	return nil
}

func getThingMetadata(path string) (model.CatalogThingModel, error) {
	// TODO: should internal.ReadRequiredFiles be used here?
	data, err := os.ReadFile(path)
	if err != nil {
		return model.CatalogThingModel{}, err
	}

	var ctm model.CatalogThingModel
	err = json.Unmarshal(data, &ctm)
	if err != nil {
		return model.CatalogThingModel{}, err
	}

	return ctm, nil
}

func saveTOC(rootPath string, tocBytes []byte) error {
	file, err := os.Create(filepath.Join(rootPath, TOCFilename))
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(tocBytes)
	return nil
}
