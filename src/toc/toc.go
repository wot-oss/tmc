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

const TMExt = ".tm.jsonld"
const TOCFilename = "tm-catalog.toc.json"

type toc struct {
	Meta     tocMeta                            `json:"meta"`
	Contents map[string]model.CatalogThingModel `json:"contents"`
}

type tocMeta struct {
	Created time.Time `json:"created"`
}

func Create(path string) error {
	// Prepare data collection for logging stats
	var log = slog.Default()
	fileCount := 0
	start := time.Now()

	newTOC := toc{
		Meta:     tocMeta{Created: time.Now()},
		Contents: map[string]model.CatalogThingModel{},
	}

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
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
				// Use id as index, but don't repeat inside object
				id := thingMeta.ID
				thingMeta.ID = ""
				newTOC.Contents[id] = thingMeta
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

// getThingMetadata reads the file at path and returns its contents
// unmarshalled into the CatalogThingModel struct
func getThingMetadata(path string) (model.CatalogThingModel, error) {
	// Read TM file as bytes
	data, err := os.ReadFile(path)
	if err != nil {
		return model.CatalogThingModel{}, err
	}

	// Try to decode bytes into thingMeta struct
	var meta model.CatalogThingModel
	meta.Path = path
	err = json.Unmarshal(data, &meta)
	if err != nil {
		return model.CatalogThingModel{}, err
	}

	if meta.ID == "" {
		msg := "Thing Model does not have the required 'id' field"
		return model.CatalogThingModel{}, fmt.Errorf(msg)
	}

	return meta, nil
}

func saveToc(tocBytes []byte) error {
	// Check for an existing toc file
	file, err := os.Create(TOCFilename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(tocBytes)
	return nil
}
