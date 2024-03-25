package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

func Fetch(ctx context.Context, repo model.RepoSpec, idOrName, outputPath string, restoreId bool) error {

	id, thing, err, errs := commands.FetchByTMIDOrName(ctx, repo, idOrName, restoreId)
	if err != nil {
		Stderrf("Could not fetch from repo: %v", err)
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
