package toc

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

const TMExt = ".tm.json"
const TOCFilename = "tm-catalog.toc.json"

func Create(rootPath string) error {
	// Prepare data collection for logging stats
	var log = slog.Default()
	fileCount := 0
	start := time.Now()

	newTOC := model.TOC{
		Meta: model.TOCMeta{Created: time.Now()},
		Data: []model.TOCEntry{},
	}

	err := filepath.Walk(rootPath, func(absPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if strings.HasSuffix(info.Name(), TMExt) {
				absPath = filepath.ToSlash(absPath)
				rootPath = filepath.ToSlash(rootPath)
				thingMeta, err := getThingMetadata(rootPath, absPath)
				if err != nil {
					msg := "Failed to extract metadata from file %s with error:"
					msg = fmt.Sprintf(msg, absPath)
					log.Error(msg)
					log.Error(err.Error())
					log.Error("The file will be excluded from the table of contents.")
					return nil
				}
				// rootPath/relPath provided by walker, can ignore error
				relPath, _ := filepath.Rel(rootPath, absPath)
				err = insert(relPath, &newTOC, thingMeta)
				if err != nil {
					log.Error(fmt.Sprintf("Failed to insert %s into toc:", absPath))
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
	err = saveToc(rootPath, newTOCJson)
	msg := "Created table of content with %d entries in %s "
	msg = fmt.Sprintf(msg, fileCount, duration.String())
	log.Info(msg)
	return nil
}

func getThingMetadata(rootPath, absPath string) (model.CatalogThingModel, error) {
	// TODO: should internal.ReadRequiredFiles be used here?
	data, err := os.ReadFile(absPath)
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

func saveToc(rootPath string, tocBytes []byte) error {
	file, err := os.Create(filepath.Join(rootPath, TOCFilename))
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(tocBytes)
	return nil
}

func insert(relPath string, toc *model.TOC, ctm model.CatalogThingModel) error {
	official := internal.Prep(ctm.Manufacturer.Name) == internal.Prep(ctm.Author.Name)
	tmid, err := model.ParseTMID(ctm.ID, official)
	if err != nil {
		return err
	}
	name := filepath.Dir(relPath)
	tocEntry, ok := toc.FindByName(name)
	// TODO: provide copy method for CatalogThingModel in TocThing
	if !ok {
		tocEntry.Name = name
		tocEntry.Manufacturer.Name = tmid.Manufacturer
		tocEntry.Mpn = tmid.Mpn
		tocEntry.Author.Name = tmid.Author
	}
	version := model.Version{Model: tmid.Version.Base.String()}
	externalID := ""
	original := ctm.Links.FindLink("original")
	if original != nil {
		externalID = original.HRef
	}

	tv := model.TOCVersion{
		Description: ctm.Description,
		TimeStamp:   tmid.Version.Timestamp,
		Version:     version,
		TMID:        ctm.ID,
		ExternalID:  externalID,
		Digest:      tmid.Version.Hash,
		Links:       map[string]string{"content": relPath},
	}
	tocEntry.Versions = append(tocEntry.Versions, tv)
	toc.Data = append(toc.Data, tocEntry)
	return nil
}
