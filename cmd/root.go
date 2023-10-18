package cmd

import (
	"github.com/spf13/viper"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tm-catalog-cli",
	Short: "A CLI client for TM catalogs",
	Long: `tm-catalog-cli is a CLI client for contributing to and searching
ThingModel catalogs.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	viper.SetDefault("remotes", map[string]any{
		"localFS": map[string]any{
			"type": "file",
			"url":  "file:~/tm-catalog",
		},
	})
	viper.AddConfigPath(".")

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tm-catalog-cli.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
