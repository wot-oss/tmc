package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

func Fetch(remoteName, idOrName, outputFile string, withPath bool) error {
	if withPath && len(outputFile) == 0 {
		Stderrf("--with-path requires non-empty --output")
		return errors.New("--output not provided")
	}

	id, thing, err := commands.NewFetchCommand(remotes.DefaultManager()).FetchByTMIDOrName(remoteName, idOrName)
	if err != nil {
		Stderrf("Could not fetch from remote: %v", err)
		return err
	}
	thing = utils.ConvertToNativeLineEndings(thing)

	if outputFile == "" {
		fmt.Println(string(thing))
		return nil
	}

	var actualOutput string
	stat, err := os.Stat(outputFile)
	if err != nil && !os.IsNotExist(err) {
		Stderrf("Could not stat output file: %v", err)
		return err
	}
	if withPath && (os.IsNotExist(err) || !stat.IsDir()) {
		Stderrf("--with-path requires --output to be a folder")
		return errors.New("--output is not a folder")

	}
	if os.IsNotExist(err) {
		actualOutput = outputFile
	} else {
		if stat.IsDir() {
			if withPath {
				actualOutput = filepath.Join(outputFile, id)
			} else {
				actualOutput = filepath.Join(outputFile, filepath.Base(id))
			}
		} else {
			actualOutput = outputFile
		}
	}
	err = os.MkdirAll(filepath.Dir(actualOutput), 0770)
	if err != nil {
		Stderrf("Could not write output to file %s: %v", actualOutput, err)
		return err
	}
	err = os.WriteFile(actualOutput, thing, 0660)
	if err != nil {
		Stderrf("Could not write output to file %s: %v", actualOutput, err)
		return err
	}

	return nil
}
