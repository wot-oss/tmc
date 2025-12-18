package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var createDockerImgCmd = &cobra.Command{
	Use:   "docker <image-name> <output-tar>",
	Short: "Create a docker image for current TMC configuration",
	Long:  `Create a docker image for current TMC configuration. Packs all configured repositories to a single docker image.`,
	Args:  cobra.MinimumNArgs(1),
	Run:   createDockerImg,
}

func init() {
	RootCmd.AddCommand(createDockerImgCmd)
	AddRepoConstraintFlags(createDockerImgCmd)
	AddOutputFormatFlag(createDockerImgCmd)
}

func createDockerImg(cmd *cobra.Command, args []string) {
	spec := RepoSpecFromFlags(cmd)
	format := cmd.Flag("format").Value.String()

	dockerImgCLIArgs := strings.Join(args, " ")
	imgName := strings.Split(dockerImgCLIArgs, " ")[0]
	outputFile := strings.Split(dockerImgCLIArgs, " ")[1]
	err := cli.CreateDockerImage(context.Background(), spec, imgName, outputFile, format)
	if err != nil {
		cli.Stderrf("creating docker image failed: %v", err)
		os.Exit(1)
	}
}
