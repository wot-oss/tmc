package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var createDockerImgCmd = &cobra.Command{
	Use:   "docker <image-tag> <output-tar> [--name <label-name>] [--maintainer <label-maintainer>] [--version <label-version>]",
	Short: "Create a docker image for current TMC configuration",
	Long: `Create a docker image for current TMC configuration. Packs all configured repositories to a single docker image.
<image-tag> tag for the created docker image, converted to lower case. <output-tar> is the path to the output tar file, e.g. './tm-catalog.tar'.`,
	Args: cobra.ExactArgs(2),
	Run:  createDockerImg,
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

	imgTag := args[0]
	outputFile := args[1]

	maintainerName, _ := cmd.Flags().GetString("name")
	maintainerEmail, _ := cmd.Flags().GetString("maintainer")
	version, _ := cmd.Flags().GetString("version")

	err := cli.CreateDockerImage(context.Background(), &spec, strings.ToLower(imgTag), outputFile, format, maintainerName, maintainerEmail, version)
	if err != nil {
		cli.Stderrf("creating docker image failed: %v", err)
		os.Exit(1)
	}
}
