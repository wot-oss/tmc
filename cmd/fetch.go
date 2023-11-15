package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch NAME[:SEMVER|DIGEST]",
	Short: "Fetches the TM by name",
	Long:  "Fetches TM by name, optionally accepting semantic version or digest",
	Args:  cobra.ExactArgs(1),
	Run:   executeFetch,
}

func init() {
	rootCmd.AddCommand(fetchCmd)
	fetchCmd.Flags().StringP("remote", "r", "", "use named remote instead of default")
}

func executeFetch(cmd *cobra.Command, args []string) {
	log := slog.Default()

	remoteName := cmd.Flag("remote").Value.String()
	remote, err := remotes.Get(remoteName)
	if err != nil {
		log.Error(fmt.Sprintf("could not initialize a remote instance for %s. check config", remoteName), "error", err)
		os.Exit(1)
	}

	fn := &commands.FetchName{}
	err = fn.Parse(args[0])
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	thing, err := commands.FetchThingByName(fn, remote)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println(string(thing))
}
