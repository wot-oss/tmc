package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/commands"
)

var variantAddCmd = &cobra.Command{
	Use:   "add --tm-id <tm-id> [--variant-id <variant-id> | --mpn <mpn> [--description <description>] [--title <title>]]",
	Short: "Add a variant to an index entry",
	Long:  `Either link an existing variant TMID with --variant-id, or create a new variant from the parent Thing Model with --mpn (optional --description and --title) and link it.`,
	Args:  cobra.NoArgs,
	Run:   addVariant,
}

func addVariant(command *cobra.Command, _ []string) {
	spec := RepoSpecFromFlags(command)
	tmID := command.Flag("tm-id").Value.String()

	variantID, _ := command.Flags().GetString("variant-id")
	mpn, _ := command.Flags().GetString("mpn")
	description, _ := command.Flags().GetString("description")
	title, _ := command.Flags().GetString("title")

	opts := commands.AddVariantOptions{
		VariantID:   variantID,
		Mpn:         mpn,
		Description: description,
		Title:       title,
	}

	if opts.VariantID == "" && opts.Mpn == "" {
		cli.Stderrf("either --variant-id or --mpn must be provided")
		os.Exit(1)
	}
	if opts.VariantID != "" && opts.Mpn != "" {
		cli.Stderrf("--variant-id and --mpn are mutually exclusive")
		os.Exit(1)
	}
	if opts.VariantID != "" && (opts.Description != "" || opts.Title != "") {
		cli.Stderrf("--description and --title can only be used with --mpn")
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
	variantAddCmd.Flags().String("title", "", "Optional title override for a newly created variant")
	variantAddCmd.MarkFlagRequired("tm-id")
	variantCmd.AddCommand(variantAddCmd)

	AddRepoDisambiguatorFlags(variantAddBatchCmd)
	variantAddBatchCmd.Flags().String("json-file", "", "Path to JSON file with variants")
	variantCmd.AddCommand(variantAddBatchCmd)
}

var variantAddBatchCmd = &cobra.Command{
	Use:   "add-batch --json-file <file>",
	Short: "Add multiple variants to Thing Models from JSON",
	Long:  `Add multiple variants from a JSON file. Each variant requires tm-id and mpn, with optional description and title.`,
	Args:  cobra.NoArgs,
	Run:   addVariantBatch,
}

func addVariantBatch(command *cobra.Command, _ []string) {
	spec := RepoSpecFromFlags(command)

	jsonFile, _ := command.Flags().GetString("json-file")

	var jsonData []byte
	var err error

	if jsonFile != "" {
		jsonData, err = os.ReadFile(jsonFile)
		if err != nil {
			cli.Stderrf("failed to read JSON file: %v", err)
			os.Exit(1)
		}
	} else {
		cli.Stderrf("JSON file must be provided with --json-file")
		os.Exit(1)
	}

	var requests []commands.AddVariantBatchRequest
	if err := json.Unmarshal(jsonData, &requests); err != nil {
		cli.Stderrf("failed to parse JSON: %v", err)
		os.Exit(1)
	}

	if len(requests) == 0 {
		cli.Stderrf("no variants provided in JSON")
		os.Exit(1)
	}

	ctx := context.Background()
	successCount := 0
	failureCount := 0

	for _, req := range requests {
		opts := commands.AddVariantOptions{
			Mpn:         req.Mpn,
			Description: req.Description,
			Title:       req.Title,
		}
		err := cli.VariantAdd(ctx, spec, req.TmID, opts)
		if err != nil {
			failureCount++
		} else {
			successCount++
			fmt.Printf("Successfully added variant for %s\n", req.TmID)
		}
	}

	fmt.Printf("\nBatch variant addition completed: %d succeeded, %d failed\n", successCount, failureCount)

	if failureCount > 0 {
		os.Exit(1)
	}
}
