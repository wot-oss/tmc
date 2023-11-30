package cli

import (
	"fmt"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

func Fetch(fetchName, remoteName string) error {
	fn := &commands.FetchName{}
	err := fn.Parse(fetchName)
	if err != nil {
		Stderrf("Could not parse name %s: %v", fetchName, err)
		return err
	}

	thing, err := commands.FetchThingByName(fn, remoteName)
	if err != nil {
		Stderrf("Could not fetch from remote: %v", err)
		return err
	}
	thing = utils.ConvertToNativeLineEndings(thing)
	fmt.Println(string(thing))
	return nil
}
