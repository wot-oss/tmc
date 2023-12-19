package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

var ErrInvalidArgs = errors.New("invalid arguments")

func RemoteList() error {
	colWidth := columnWidth()
	config, err := remotes.ReadConfig()
	if err != nil {
		Stderrf("Cannot read remotes config: %v", err)
		return err
	}
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintf(table, "NAME\tTYPE\tENBL\tLOCATION\n")
	for name, value := range config {
		typ := fmt.Sprintf("%v", value[remotes.KeyRemoteType])
		e := utils.JsGetBool(value, remotes.KeyRemoteEnabled)
		enbl := e == nil || *e
		var enblS string
		if enbl {
			enblS = "Y"
		} else {
			enblS = "N"
		}
		u := fmt.Sprintf("%v", value[remotes.KeyRemoteLoc])
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", elideString(name, colWidth), typ, enblS, u)
	}
	_ = table.Flush()
	return nil
}

func RemoteAdd(name, typ, confStr, confFile string) error {
	return remoteSaveConfig(name, typ, confStr, confFile, remotes.Add)
}
func RemoteSetConfig(name, typ, confStr, confFile string) error {
	return remoteSaveConfig(name, typ, confStr, confFile, remotes.SetConfig)
}

func remoteSaveConfig(name, typ, confStr, confFile string, saver func(name, typ, confStr string, confFile []byte) error) error {
	if !remotes.ValidRemoteNameRegex.MatchString(name) {
		Stderrf("invalid name: %v", name)
		return ErrInvalidArgs
	}
	var bytes []byte
	if confFile != "" {
		var err error
		_, bytes, err = utils.ReadRequiredFile(confFile)
		if err != nil {
			Stderrf("cannot read file: %v", confFile)
			return err
		}
	}

	typ = inferType(typ, bytes)

	if !isValidType(typ) {
		Stderrf("invalid type: %v. Valid types are: %v", typ, remotes.SupportedTypes)
		return ErrInvalidArgs
	}

	if confStr != "" && confFile != "" {
		Stderrf("specify either <config> or --file=<configFileName>, not both")
		return ErrInvalidArgs
	}
	if confStr == "" && confFile == "" {
		Stderrf("must specify either <config> or --file=<configFileName>")
		return ErrInvalidArgs
	}
	err := saver(name, typ, confStr, bytes)
	if err != nil {
		Stderrf("error saving remote config: %v", err)
	}
	return err
}
func inferType(typ string, bytes []byte) string {
	if typ != "" {
		return typ
	}
	if len(bytes) > 0 {
		config, err := remotes.AsRemoteConfig(bytes)
		if err == nil {
			t := config[remotes.KeyRemoteType]
			if t != nil {
				if ts, ok := t.(string); ok {
					return ts
				}
			}
		}
	}
	return ""
}

func RemoteToggleEnabled(name string) error {
	err := remotes.ToggleEnabled(name)
	if err != nil {
		Stderrf("%v", err)
	}
	return err
}

func RemoteRemove(name string) error {
	err := remotes.Remove(name)
	if err != nil {
		Stderrf("%v", err)
	}
	return err
}

func RemoteShow(name string) error {
	config, err := remotes.ReadConfig()
	if err != nil {
		Stderrf("Cannot read remotes config: %v", err)
		return err
	}
	if rc, ok := config[name]; ok {
		bytes, err := json.MarshalIndent(rc, "", "  ")
		if err != nil {
			Stderrf("couldn't print config: %v", err)
			return err
		}
		fmt.Println(string(bytes))
	} else {
		fmt.Printf("no remote named %s\n", name)
		return remotes.ErrRemoteNotFound
	}
	return nil
}

func RemoteRename(oldName, newName string) (err error) {
	err = remotes.Rename(oldName, newName)
	if err != nil {
		if errors.Is(err, remotes.ErrRemoteNotFound) {
			Stderrf("remote %s not found", oldName)
			return
		}
		if errors.Is(err, remotes.ErrInvalidRemoteName) {
			Stderrf("invalid remote name: %s", newName)
			return
		}
		Stderrf("error renaming a remote: %v", err)
	}
	return
}

func isValidType(typ string) bool {
	for _, t := range remotes.SupportedTypes {
		if typ == t {
			return true
		}
	}
	return false
}
