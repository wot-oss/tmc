package cmd

import "github.com/spf13/cobra"

var variantCmd = &cobra.Command{
	Use:   "variant",
	Short: "Manage Thing Model variants",
	Long:  `The subcommands of the variant command allow managing hasVariant links on index entries.`,
}

func init() {
	RootCmd.AddCommand(variantCmd)
}
