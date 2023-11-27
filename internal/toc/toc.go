package toc

import (
	"encoding/json"
	"errors"
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
				thingModel, err := getThingMetadata(rootPath, absPath)
				if err != nil {
					msg := "Failed to extract metadata from file %s with error:"
					msg = fmt.Sprintf(msg, absPath)
					log.Error(msg)
					log.Error(err.Error())
					log.Error("The file will be excluded from the table of contents.")
					return nil
				}
				err = insert(&newTOC, thingModel)
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

	relPath, err := filepath.Rel(rootPath, absPath)
	if err != nil {
		msg := "unable to compute relative path to root %s"
		return model.CatalogThingModel{}, errors.New(fmt.Sprintf(msg, absPath))
	}
	ctm.Links = append(ctm.Links, model.Link{Rel: model.RelContent, Type: model.ThingModelMediaType, HRef: relPath})

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

func insert(toc *model.TOC, ctm model.CatalogThingModel) error {
	official := internal.Prep(ctm.Manufacturer.Name) == internal.Prep(ctm.Author.Name)
	tmid, err := model.ParseTMID(ctm.ID, official)
	if err != nil {
		return err
	}
	contentLink := ctm.Links.FindLink(model.RelContent)
	if contentLink == nil {
		return errors.New("no relative path provided in content link.")
	}
	name := filepath.Dir(contentLink.HRef)
	tocEntry, ok := toc.FindByName(name)
	// TODO: provide copy method for CatalogThingModel in TocThing
	if !ok {
		tocEntry.Name = name
		tocEntry.Manufacturer.Name = tmid.Manufacturer
		tocEntry.Mpn = tmid.Mpn
		tocEntry.Author.Name = tmid.Author
	}
	version := model.Version{Model: tmid.Version.Base.String()}
	id := ctm.ID
	ctm.ID = ""
	tv := model.TOCVersion{
		ExtendedFields: ctm.ExtendedFields,
		TimeStamp:      tmid.Version.Timestamp,
		Version:        version,
		ID:             id,
		Digest:         tmid.Version.Hash,
	}
	tocEntry.Versions = append(tocEntry.Versions, tv)
	toc.Data = append(toc.Data, tocEntry)
	return nil
}
