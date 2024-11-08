package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/utils"
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

	_, _ = fmt.Fprintf(table, "NAME\tTYPE\tENBL\tLOCATION\tDESCRIPTION\n")
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
		descr, _ := value[repos.KeyRepoDescription].(string)

		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", elideString(name, colWidth), typ, enblS, u, descr)
	}
	_ = table.Flush()
	return nil
}

func RepoAdd(name, typ, confStr, confFile, descr string) error {
	if !repos.ValidRepoNameRegex.MatchString(name) {
		Stderrf("invalid name: %v", name)
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

	rc, err := repos.NewRepoConfig(typ, confStr, bytes, descr)
	if err != nil {
		return err
	}

	err = repos.Add(name, rc)
	if err != nil {
		Stderrf("error saving repo config: %v", err)
	}
	return err
}

func RepoSetConfig(name, confStr, confFile string) error {
	return updateRepoConfig(name, func(conf map[string]any) (map[string]any, error) {
		var bytes []byte
		if confFile != "" {
			var err error
			_, bytes, err = utils.ReadRequiredFile(confFile)
			if err != nil {
				Stderrf("cannot read file: %v", confFile)
				return nil, err
			}
		}

		rs, err := repos.ReadConfig()
		if err != nil {
			return nil, err
		}
		oldConf := rs[name]
		newConf, err := repos.NewRepoConfig(utils.JsGetStringOrEmpty(oldConf, repos.KeyRepoType), confStr, bytes, utils.JsGetStringOrEmpty(oldConf, repos.KeyRepoDescription))
		return newConf, err
	})
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

func RepoSetDescription(ctx context.Context, name, description string) error {
	return updateRepoConfig(name, func(conf map[string]any) (map[string]any, error) {
		conf[repos.KeyRepoDescription] = description
		return conf, nil
	})
}

func updateRepoConfig(name string, updater func(conf map[string]any) (map[string]any, error)) error {
	conf, err := repos.ReadConfig()
	if err != nil {
		Stderrf("error reading repo config: %v", err)
		return err
	}

	rc, ok := conf[name]
	if !ok {
		Stderrf("repo %s not found", name)
		return repos.ErrRepoNotFound
	}

	rc, err = updater(rc)
	if err != nil {
		Stderrf("couldn't update repo config: %v", err)
		return err
	}
	err = repos.SetConfig(name, rc)
	if err != nil {
		Stderrf("error saving repo config: %v", err)
		return err
	}
	return nil

}

func RepoSetAuth(name, kind string, data []string) error {
	return updateRepoConfig(name, func(rc map[string]any) (map[string]any, error) {
		switch kind {
		case repos.AuthMethodNone:
			delete(rc, repos.KeyRepoAuth)
			break
		case repos.AuthMethodBearerToken:
			delete(rc, repos.KeyRepoAuth)
			confValues := parseNamedArgs(data)
			err := assertNamedArgs(confValues, []string{"token"})
			if err != nil {
				rc[repos.KeyRepoAuth] = map[string]any{
					repos.AuthMethodBearerToken: data,
				}
			} else {
				rc[repos.KeyRepoAuth] = map[string]any{
					repos.AuthMethodBearerToken: confValues["token"],
				}
			}
		case repos.AuthMethodBasic:
			delete(rc, repos.KeyRepoAuth)
			confValues := parseNamedArgs(data)
			err := assertNamedArgs(confValues, []string{"username", "password"})
			if err != nil {
				Stderrf("cannot set auth of type 'basic': %v", err)
				return nil, err
			}
			rc[repos.KeyRepoAuth] = map[string]any{
				repos.AuthMethodBasic: confValues,
			}
		//case repos.AuthMethodOauthClientCredentials:
		//	delete(rc, repos.KeyRepoAuth)
		//	confValues := parseNamedArgs(data)
		//	err := assertNamedArgs(confValues, []string{"client-id", "client-secret", "token-url"}, "scopes")
		//	if err != nil {
		//		Stderrf("cannot set auth of type 'oauth-client-credentials': %v", err)
		//		return nil, err
		//	}
		//	rc[repos.KeyRepoAuth] = map[string]any{
		//		repos.AuthMethodOauthClientCredentials: confValues,
		//	}
		default:
			Stderrf("unknown auth type: %s", kind)
			return nil, errors.New("unknown auth type")
		}
		return rc, nil
	})
}

func RepoSetHeaders(name string, data []string) error {
	return updateRepoConfig(name, func(rc map[string]any) (map[string]any, error) {
		delete(rc, repos.KeyRepoHeaders)
		m := make(map[string][]string)
		for _, item := range data {
			key, value, _ := strings.Cut(item, "=")
			if arr, ok := m[key]; ok {
				arr = append(arr, value)
				m[key] = arr
			} else {
				m[key] = []string{value}
			}
		}

		rc[repos.KeyRepoHeaders] = m
		return rc, nil
	})
}

func parseNamedArgs(namedArgs []string) map[string]string {
	m := make(map[string]string)
	for _, item := range namedArgs {
		key, value, _ := strings.Cut(item, "=")
		m[key] = value
	}
	return m
}

func assertNamedArgs(pairs map[string]string, mandatory []string, allowed ...string) error {
	mnd := make([]string, len(mandatory))
	copy(mnd, mandatory)
	for name, _ := range pairs {
		mi := slices.Index(mnd, name)
		ai := slices.Index(allowed, name)
		if mi != -1 || ai != -1 {
			if mi != -1 {
				mnd = slices.Delete(mnd, mi, mi+1)
			}
		} else {
			allKeys := []string{}
			allKeys = append(allKeys, mandatory...)
			allKeys = append(allKeys, allowed...)
			slices.Sort(allKeys)
			allKeys = slices.Compact(allKeys)
			return fmt.Errorf("key is not allowed: %s. allowed keys are: %v", name, strings.Join(allKeys, ", "))
		}
	}
	if len(mnd) > 0 {
		return fmt.Errorf("missing mandatory keys: %v", strings.Join(mnd, ", "))
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
