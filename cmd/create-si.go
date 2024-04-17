/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"

	"github.com/blevesearch/bleve/v2"
	//_ "github.com/blevesearch/bleve/v2/config"
	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/model"
)

func checkFatal(err error, txt string) {
	if err != nil {
		slog.Default().Error(txt, "error", err)
		os.Exit(1)
	}
}

// createSiCmd represents the createSi command
var createSiCmd = &cobra.Command{
	Use:   "create-si",
	Short: "Creates or updates a search index",
	Long:  `Creates or updates a search index for all entries in the "Table of Contents"`,
	Run:   executeCreateSearchIndex,
}

func init() {
	RootCmd.AddCommand(createSiCmd)
	createSiCmd.Flags().StringP("repo", "r", "", "name of the remote to pull from")
	createSiCmd.Flags().StringP("directory", "d", "", "TM repository directory to pull from")
}

func executeCreateSearchIndex(cmd *cobra.Command, args []string) {
	asBatch := true
	count := 0
	maxDocuments := math.MaxInt
	repoName := cmd.Flag("repo").Value.String()
	dirName := cmd.Flag("directory").Value.String()

	spec, err := model.NewSpec(repoName, dirName)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --repo and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}
	searchResult, err, _ := commands.List(context.Background(), spec, nil)
	if err != nil {
		os.Exit(1)
	}
	// try to open index, if it not there create a fresh one
	index, err := bleve.Open("../catalog.bleve")

	if err != nil {
		// open a new index
		indexMapping := bleve.NewIndexMapping()
		index, err = bleve.New("../catalog.bleve", indexMapping)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	//closeIndex(index)

	contents := searchResult.Entries
	var batch *bleve.Batch
	if asBatch {
		batch = index.NewBatch()
	}
	//contents = contents[14:]
testLoop:
	for _, value := range contents {
		for _, version := range value.Versions {
			fqName := version.TMID
			// for now we use the relative filename as documentID
			blID := fqName
			id, thing, err, _ := commands.FetchByTMID(context.Background(), spec, fqName, false)
			_ = id
			if err != nil {
				slog.Default().Error("cant Fetch TM", "error", err.Error())
				continue
			}
			var data any
			unmErr := json.Unmarshal(thing, &data)
			checkFatal(unmErr, "Unmarshal")

			// ask if Document is already indexed
			doc, _ := index.Document(blID)
			if doc != nil {
				if batch != nil {
					batch.Delete(blID)
				} else {
					index.Delete(blID)
				}
				fmt.Printf("deleted exisiting document with id=%s first\n", blID)
				// deleteErr := index.Delete(blID)
				// fmt.Printf("deleted exisiting document with id=%s first%v\n", blID, deleteErr)
			} else {
				fmt.Printf("new document with id=%s\n", blID)
			}
			var idxErr error
			if batch != nil {
				idxErr = batch.Index(blID, data)
			} else {
				idxErr = index.Index(blID, data)
			}
			checkFatal(idxErr, "index Document")
			count++
			if count >= maxDocuments {
				break testLoop
			}
		}

	}
	if batch != nil {
		fmt.Printf("index batch for %d Documents\n", count)
		err = index.Batch(batch)
		checkFatal(err, "run batch")
	}
	err = index.Close()
	checkFatal(err, "close index")
}
