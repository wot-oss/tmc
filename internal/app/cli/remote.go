package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/repos"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

var ErrInvalidArgs = errors.New("invalid arguments")

func RepoList() error {
	colWidth := columnWidth()
	config, err := repos.ReadConfig()
	if err != nil {
		Stderrf("Cannot read repo config: %v", err)
		return err
	}
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintf(table, "NAME\tTYPE\tENBL\tLOCATION\n")
	for name, value := range config {
		typ := fmt.Sprintf("%v", value[repos.KeyRepoType])
		e := utils.JsGetBool(value, repos.KeyRepoEnabled)
		enbl := e == nil || *e
		var enblS string
		if enbl {
			enblS = "Y"
		} else {
			enblS = "N"
		}
		u := fmt.Sprintf("%v", value[repos.KeyRepoLoc])
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", elideString(name, colWidth), typ, enblS, u)
	}
	_ = table.Flush()
	return nil
}

func RepoAdd(name, typ, confStr, confFile string) error {
	return repoSaveConfig(name, typ, confStr, confFile, repos.Add)
}
func RepoSetConfig(name, typ, confStr, confFile string) error {
	return repoSaveConfig(name, typ, confStr, confFile, repos.SetConfig)
}

func repoSaveConfig(name, typ, confStr, confFile string, saver func(name, typ, confStr string, confFile []byte) error) error {
	if !repos.ValidRepoNameRegex.MatchString(name) {
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
		Stderrf("invalid type: %v. Valid types are: %v", typ, repos.SupportedTypes)
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
		Stderrf("error saving repo config: %v", err)
	}
	return err
}
func inferType(typ string, bytes []byte) string {
	if typ != "" {
		return typ
	}
	if len(bytes) > 0 {
		config, err := repos.AsRepoConfig(bytes)
		if err == nil {
			t := config[repos.KeyRepoType]
			if t != nil {
				if ts, ok := t.(string); ok {
					return ts
				}
			}
		}
	}
	return ""
}

func RepoToggleEnabled(name string) error {
	err := repos.ToggleEnabled(name)
	if err != nil {
		Stderrf("%v", err)
	}
	return err
}

func RepoRemove(name string) error {
	err := repos.Remove(name)
	if err != nil {
		Stderrf("%v", err)
	}
	return err
}

func RepoShow(name string) error {
	config, err := repos.ReadConfig()
	if err != nil {
		Stderrf("Cannot read repo config: %v", err)
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
		fmt.Printf("no repo named %s\n", name)
		return repos.ErrRepoNotFound
	}
	return nil
}

func RepoRename(oldName, newName string) (err error) {
	err = repos.Rename(oldName, newName)
	if err != nil {
		if errors.Is(err, repos.ErrRepoNotFound) {
			Stderrf("repo %s not found", oldName)
			return
		}
		if errors.Is(err, repos.ErrInvalidRepoName) {
			Stderrf("invalid repo name: %s", newName)
			return
		}
		Stderrf("error renaming a repo: %v", err)
	}
	return
}

func RepoSetAuth(name, kind, data string) error {
	conf, err := repos.ReadConfig()
	if err != nil {
		Stderrf("error setting auth: %v", err)
		return err
	}

	rc, ok := conf[name]
	if !ok {
		Stderrf("repo %s not found", name)
		return repos.ErrRepoNotFound
	}
	switch kind {
	case "bearer":
		delete(rc, repos.KeyRepoAuth)
		rc[repos.KeyRepoAuth] = map[string]any{
			"bearer": data,
		}
	default:
		Stderrf("unknown auth type: %s", kind)
		return errors.New("unknown auth type")
	}
	rb, _ := json.Marshal(rc)

	err = repos.SetConfig(name, fmt.Sprint(rc[repos.KeyRepoType]), "", rb)
	if err != nil {
		Stderrf("error saving repo config: %v", err)
		return err
	}
	return nil
}

func isValidType(typ string) bool {
	for _, t := range repos.SupportedTypes {
		if typ == t {
			return true
		}
	}
	return false
}
