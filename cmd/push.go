package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/validation"
	"log/slog"
	"os"
	"path/filepath"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push [file.tm.json]",
	Short: "Push TM to remote",
	Long:  `Push TM to remote`,
	Args:  cobra.ExactArgs(1),
	Run:   executePush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringP("remote", "r", "", "use named remote instead of default")
	pushCmd.Flags().StringP("manufacturer", "m", "", "override manufacturer defined in ThingModel")
	pushCmd.Flags().StringP("mpn", "p", "", "override manufacturer part number defined in ThingModel")
}

func executePush(cmd *cobra.Command, args []string) {
	var log = slog.Default()

	log.Debug("executing push", "args", args)
	remoteName := cmd.Flag("remote").Value.String()
	remote, err := remotes.Get(remoteName)
	if err != nil {
		log.Error(fmt.Sprintf("could not ìnitialize a remote instance for %s. check config", remoteName), "error", err)
		os.Exit(1)
	}

	filename := args[0]
	abs, err := filepath.Abs(filename)
	if err != nil {
		log.Error("error expanding import file name", "filename", filename, "error", err)
		os.Exit(1)
	}
	log.Debug("importing file", "filename", abs)

	bytes, err := os.ReadFile(abs)
	if err != nil {
		log.Error("error reading import file", "filename", abs, "error", err)
		os.Exit(1)
	}
	if len(bytes) > 2 && bytes[0] == 0xef && bytes[1] == 0xbb && bytes[2] == 0xbf {
		bytes = bytes[3:]
	}
	log.Debug(fmt.Sprintf("read %d bytes from %s beginning with %s", len(bytes), abs, string(bytes[:100])))
	var content map[string]any
	err = json.Unmarshal(bytes, &content)
	if err != nil {
		var jserr *json.SyntaxError
		if errors.As(err, &jserr) {
			log.Error("error parsing import JSON file", "filename", abs, "error", jserr, "offset", jserr.Offset)
		} else {
			log.Error("error parsing import JSON file", "filename", abs, "error", err)
		}
		os.Exit(1)
	}

	validator, err := validation.NewTMValidator()
	if err != nil {
		log.Error("could not create TM validator", "error", err)
		os.Exit(1)
	}
	err = validator.ValidateTM(bytes)
	if err != nil {
		log.Error("validation failed", "error", err)

	}

	manufacturer, mpn, err := getManufacturerData(cmd, content)
	if err != nil {
		log.Error("manufacturer data could not be determined", "filename", abs, "error", err)
		os.Exit(1)
	}

	err = remote.Push(manufacturer, mpn, filepath.Base(abs), content)
	if err != nil {
		log.Error("error pushing to remote", "filename", abs, "error", err)
		os.Exit(1)
	}
	log.Info("pushed successfully")
}

func getManufacturerData(cmd *cobra.Command, content map[string]any) (string, string, error) {
	var mpn string
	mpnf := cmd.Flag("mpn").Value.String()
	if mpnf != "" {
		mpn = mpnf
	} else {
		if mpni, ok := content["schema:mpn"]; !ok {
			return "", "", fmt.Errorf("mpn is not defined neither in the TM, nor via --npm flag")
		} else {
			mpn, ok = mpni.(string)
			if !ok {
				return "", "", fmt.Errorf("unexpected type of schema:mpn value")
			}
		}
	}

	var manufacturer string
	manuff := cmd.Flag("manufacturer").Value.String()
	if manuff != "" {
		manufacturer = manuff
	} else {
		if manufi, ok := content["schema:manufacturer"]; !ok {
			return "", "", fmt.Errorf("manufacturer is not defined neither in the TM, nor via --manufacturer flag")
		} else {
			manufs, ok := manufi.(string)
			if ok {
				manufacturer = manufs
			} else if manufm, ok := manufi.(map[string]any); ok {
				if name, ok := manufm["name"]; !ok {
					return "", "", fmt.Errorf("unexpected type of schema:manufacturer value")
				} else {
					if names, ok := name.(string); !ok {
						return "", "", fmt.Errorf("unexpected type of schema:manufacturer value")
					} else {
						manufacturer = names
					}
				}
			} else {
				return "", "", fmt.Errorf("unexpected type of schema:manufacturer value")
			}
		}
	}
	return manufacturer, mpn, nil
}
