package toc

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

const TMExt = ".tm.json"
const TOCFilename = "tm-catalog.toc.json"

func Create(path string) error {
	// Prepare data collection for logging stats
	var log = slog.Default()
	fileCount := 0
	start := time.Now()

	newTOC := model.Toc{
		Meta:     model.TocMeta{Created: time.Now()},
		Contents: map[string]model.TocThing{},
	}

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if strings.HasSuffix(info.Name(), TMExt) {
				thingModel, err := getThingMetadata(path)
				if err != nil {
					msg := "Failed to extract metadata from file %s with error:"
					msg = fmt.Sprintf(msg, path)
					log.Error(msg)
					log.Error(err.Error())
					log.Error("The file will be excluded from the table of contents.")
					return nil
				}
				insert(newTOC, thingModel)
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
	err = saveToc(newTOCJson)
	msg := "Created table of content with %d entries in %s "
	msg = fmt.Sprintf(msg, fileCount, duration.String())
	log.Info(msg)
	return nil
}

func getThingMetadata(path string) (model.CatalogThingModel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.CatalogThingModel{}, err
	}

	var ctm model.CatalogThingModel
	ctm.Path = path
	err = json.Unmarshal(data, &ctm)
	if err != nil {
		return model.CatalogThingModel{}, err
	}

	if ctm.ID == "" {
		msg := "Thing Model does not have the required 'id' field"
		return model.CatalogThingModel{}, fmt.Errorf(msg)
	}

	return ctm, nil
}

func saveToc(tocBytes []byte) error {
	// Create or open toc file
	file, err := os.Create(TOCFilename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(tocBytes)
	return nil
}

func insert(table model.Toc, ctm model.CatalogThingModel) {
	// TODO: extract timestamp from ID and add to tocEntry
	name := filepath.Dir(ctm.Path)
	tocEntry, ok := table.Contents[name]
	if !ok {
		tocEntry.ThingModel = ctm.ThingModel
	}
	// TODO: remove stopgap
	now := time.Now()
	tv := model.TocVersion{ExtendedFields: ctm.ExtendedFields, TimeStamp: &now}
	tocEntry.Versions = append(tocEntry.Versions, tv)
	table.Contents[name] = tocEntry
}
