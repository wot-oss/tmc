package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var createDockerImgCmd = &cobra.Command{
	Use:   "docker <image-name> <output-tar> [--name <label-name>] [--maintainer <label-maintainer>] [--version <label-version>]",
	Short: "Create a docker image for current TMC configuration",
	Long:  `Create a docker image for current TMC configuration. Packs all configured repositories to a single docker image.`,
	Args:  cobra.MinimumNArgs(1),
	Run:   createDockerImg,
}

func init() {
	RootCmd.AddCommand(createDockerImgCmd)
	AddRepoConstraintFlags(createDockerImgCmd)
	AddOutputFormatFlag(createDockerImgCmd)

	createDockerImgCmd.Flags().String("name", "", "Specify the image label NAME (optional, default: W3C Thing Model Catalog)")
	createDockerImgCmd.Flags().String("maintainer", "", "Specify the image label MAINTAINER (optional, default: https://github.com/wot-oss)")
	createDockerImgCmd.Flags().String("version", "", "Specify the image label VERSION (optional, default: latest)")
}

func createDockerImg(cmd *cobra.Command, args []string) {
	spec := RepoSpecFromFlags(cmd)
	format := cmd.Flag("format").Value.String()

	dockerImgCLIArgs := strings.Join(args, " ")
	imgTag := strings.Split(dockerImgCLIArgs, " ")[0]
	outputFile := strings.Split(dockerImgCLIArgs, " ")[1]
	maintainerName, _ := cmd.Flags().GetString("name")
	maintainerEmail, _ := cmd.Flags().GetString("maintainer")
	version, _ := cmd.Flags().GetString("version")
	err := cli.CreateDockerImage(context.Background(), spec, imgTag, outputFile, format, maintainerName, maintainerEmail, version)
	if err != nil {
		cli.Stderrf("creating docker image failed: %v", err)
		os.Exit(1)
	}
}
