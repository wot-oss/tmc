package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
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

	_, _ = fmt.Fprintf(table, "NAME\tTYPE\tURL\n")
	for name, value := range config {
		typ := elideString(fmt.Sprintf("%v", value[remotes.KeyRemoteType]), colWidth)
		u := elideString(fmt.Sprintf("%v", value[remotes.KeyRemoteUrl]), colWidth)
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\n", elideString(name, colWidth), typ, u)
	}
	_ = table.Flush()
	return nil
}

func RemoteAdd(name, typ, confStr, confFile string) error {
	if name == "" {
		Stderrf("invalid name: %v", name)
		return ErrInvalidArgs
	}
	if !isValidType(typ) {
		Stderrf("invalid type: %v. Valid types are: %v", typ, remotes.SupportedTypes)
		return ErrInvalidArgs
	}

	if confStr != "" && confFile != "" {
		Stderrf("specify either <config> or <configFileName>, not both")
		return ErrInvalidArgs
	}
	if confStr == "" && confFile == "" {
		Stderrf("must specify either <config> or <configFileName>")
		return ErrInvalidArgs
	}

	var bytes []byte
	if confFile != "" {
		var err error
		_, bytes, err = internal.ReadRequiredFile(confFile)
		if err != nil {
			Stderrf("cannot read file: %v", confFile)
			return err
		}
	}

	return remotes.Add(name, typ, confStr, bytes)
}
func RemoteSetDefault(name string) error {
	err := remotes.SetDefault(name)
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
	}
	return nil
}

func isValidType(typ string) bool {
	for _, t := range remotes.SupportedTypes {
		if typ == t {
			return true
		}
	}
	return false
}
