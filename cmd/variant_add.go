package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/commands"
)

var variantAddCmd = &cobra.Command{
	Use:   "add --tm-id <tm-id> [--family-name <name>] [--variant-id <variant-id> --with-attachments | --mpn <mpn> [--description <description>] [--title <title>] [--with-attachments]]",
	Short: "Add a variant to an index entry",
	Long:  `Either link an existing variant TMID with --variant-id, or create a new variant from the parent Thing Model with --mpn (optional --description and --title) and link it. Optionally, include attachments from the parent TM with --with-attachments.`,
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
	familyName, _ := command.Flags().GetString("family-name")

	opts := commands.AddVariantOptions{
		FamilyName:      familyName,
		VariantID:       variantID,
		Mpn:             mpn,
		Description:     description,
		Title:           title,
		WithAttachments: command.Flags().Changed("with-attachments"),
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
	variantAddCmd.Flags().String("family-name", "", "Name for a new family; if supplied for an existing family, it must match")
	variantAddCmd.Flags().String("variant-id", "", "TMID of an existing variant Thing Model to link")
	variantAddCmd.Flags().String("mpn", "", "MPN for a new variant created from the parent Thing Model")
	variantAddCmd.Flags().String("description", "", "Optional description override for a newly created variant")
	variantAddCmd.Flags().String("title", "", "Optional title override for a newly created variant")
	variantAddCmd.Flags().Bool("with-attachments", false, "Include attachments from the parent Thing Model")
	variantAddCmd.MarkFlagRequired("tm-id")
	variantCmd.AddCommand(variantAddCmd)

	AddRepoDisambiguatorFlags(variantAddBatchCmd)
	variantAddBatchCmd.Flags().String("tm-id", "", "TMID of the parent Thing Model for all variants")
	variantAddBatchCmd.Flags().String("json-file", "", "Path to JSON file with variants")
	variantAddBatchCmd.MarkFlagRequired("tm-id")
	variantCmd.AddCommand(variantAddBatchCmd)
}

var variantAddBatchCmd = &cobra.Command{
	Use:   "add-batch --tm-id <tm-id> --json-file <file>",
	Short: "Add multiple variants to one Thing Model from JSON",
	Long:  `Add multiple variants to the parent specified by --tm-id. The JSON file contains a variants array and, for a new family, familyName. Each variant provides either variant-id or mpn, with optional description, title, and with-attachments for creation.`,
	Args:  cobra.NoArgs,
	Run:   addVariantBatch,
}

func addVariantBatch(command *cobra.Command, _ []string) {
	spec := RepoSpecFromFlags(command)
	tmID, _ := command.Flags().GetString("tm-id")

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

	var request commands.AddVariantsBatchRequest
	if !json.Valid(jsonData) {
		cli.Stderrf("failed to parse JSON: invalid JSON document")
		os.Exit(1)
	}
	decoder := json.NewDecoder(bytes.NewReader(jsonData))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		cli.Stderrf("failed to parse JSON: %v", err)
		os.Exit(1)
	}

	if len(request.Variants) == 0 {
		cli.Stderrf("no variants provided in JSON")
		os.Exit(1)
	}

	ctx := context.Background()
	successCount := 0
	failureCount := 0

	results, err := commands.AddVariantsBatch(ctx, spec, tmID, request)
	if err != nil {
		cli.Stderrf("variant batch add failed: %v", err)
		os.Exit(1)
	}
	for _, result := range results {
		if result.Error != "" {
			failureCount++
			cli.Stderrf("Could not add variant to %s: %s", tmID, result.Error)
		} else {
			successCount++
			fmt.Printf("Successfully added variant %s for %s\n", result.VariantID, tmID)
		}
	}

	fmt.Printf("\nBatch variant addition completed: %d succeeded, %d failed\n", successCount, failureCount)

	if failureCount > 0 {
		os.Exit(1)
	}
}
