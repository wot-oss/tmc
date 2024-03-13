package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

type FetchExecutor struct {
}

func NewFetchExecutor() *FetchExecutor {
	return &FetchExecutor{}
}

func (e *FetchExecutor) Fetch(remote model.RepoSpec, idOrName, outputPath string, restoreId bool) error {

	id, thing, err, errs := commands.NewFetchCommand().FetchByTMIDOrName(remote, idOrName, restoreId)
	if err != nil {
		Stderrf("Could not fetch from remote: %v", err)
		return err
	}
	defer printErrs("Errors occurred while fetching:", errs)

	thing = utils.ConvertToNativeLineEndings(thing)

	if outputPath == "" {
		fmt.Println(string(thing))
		return nil
	}

	f, err := os.Stat(outputPath)
	if err != nil && !os.IsNotExist(err) {
		Stderrf("Could not stat output folder: %v", err)
		return err
	}
	if f != nil && !f.IsDir() {
		Stderrf("output target folder --output is not a folder")
		return errors.New("output target folder --output is not a folder")
	}

	finalOutput := filepath.Join(outputPath, id)
	err = os.MkdirAll(filepath.Dir(finalOutput), 0770)
	if err != nil {
		Stderrf("could not write ThingModel to file %s: %v", finalOutput, err)
		return err
	}

	err = os.WriteFile(finalOutput, thing, 0660)
	if err != nil {
		Stderrf("could not write ThingModel to file %s: %v", finalOutput, err)
		return err
	}

	return nil
}
