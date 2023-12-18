package cli

import (
	"fmt"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

func Fetch(remoteName, idOrName string) error {
	thing, err := commands.NewFetchCommand(remotes.DefaultManager()).FetchByTMIDOrName(remoteName, idOrName)
	if err != nil {
		Stderrf("Could not fetch from remote: %v", err)
		return err
	}
	thing = utils.ConvertToNativeLineEndings(thing)
	fmt.Println(string(thing))
	return nil
}
