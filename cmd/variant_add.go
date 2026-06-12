package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/commands"
)

var variantAddCmd = &cobra.Command{
	Use:   "add --tm-id <tm-id> [--variant-id <variant-id> | --mpn <mpn> [--description <description>] [--author <author>]]",
	Short: "Add a variant to an index entry",
	Long:  `Either link an existing variant TMID with --variant-id, or create a new variant from the parent Thing Model with --mpn (optional --description and --author) and link it.`,
	Args:  cobra.NoArgs,
	Run:   addVariant,
}

func addVariant(command *cobra.Command, _ []string) {
	spec := RepoSpecFromFlags(command)
	tmID := command.Flag("tm-id").Value.String()

	variantID, _ := command.Flags().GetString("variant-id")
	mpn, _ := command.Flags().GetString("mpn")
	description, _ := command.Flags().GetString("description")
	author, _ := command.Flags().GetString("author")

	opts := commands.AddVariantOptions{
		VariantID:   variantID,
		Mpn:         mpn,
		Description: description,
		Author:      author,
	}

	if opts.VariantID == "" && opts.Mpn == "" {
		cli.Stderrf("either --variant-id or --mpn must be provided")
		os.Exit(1)
	}
	if opts.VariantID != "" && opts.Mpn != "" {
		cli.Stderrf("--variant-id and --mpn are mutually exclusive")
		os.Exit(1)
	}
	if opts.VariantID != "" && (opts.Description != "" || opts.Author != "") {
		cli.Stderrf("--description and --author can only be used with --mpn")
		os.Exit(1)
	}

	err := cli.VariantAdd(context.Background(), spec, tmID, opts)
	if err != nil {
		cli.Stderrf("variant add failed: %v", err)
		os.Exit(1)
	}
}

func init() {
	AddRepoDisambiguatorFlags(variantAddCmd)
	variantAddCmd.Flags().String("tm-id", "", "TMID of the parent Thing Model")
	variantAddCmd.Flags().String("variant-id", "", "TMID of an existing variant Thing Model to link")
	variantAddCmd.Flags().String("mpn", "", "MPN for a new variant created from the parent Thing Model")
	variantAddCmd.Flags().String("description", "", "Optional description override for a newly created variant")
	variantAddCmd.Flags().String("author", "", "Optional author override for a newly created variant")
	variantAddCmd.MarkFlagRequired("tm-id")
	variantCmd.AddCommand(variantAddCmd)
}
