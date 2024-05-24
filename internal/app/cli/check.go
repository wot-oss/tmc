package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/wot-oss/tmc/internal/commands/validate"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

const (
	CheckOK = CheckResultType(iota)
	CheckErr
)

var errCheckResourcesFailed = errors.New("check resources failed")

type CheckResultType int

func (t CheckResultType) String() string {
	switch t {
	case CheckOK:
		return "OK"
	case CheckErr:
		return "error"
	default:
		return "unknown"
	}
}

type CheckResult struct {
	typ     CheckResultType
	refName string
	text    string
}

func (r CheckResult) String() string {
	return fmt.Sprintf("%v\t %s: %s", r.typ, r.refName, r.text)
}

func CheckResources(ctx context.Context, spec model.RepoSpec, names []string) error {
	repo, err := repos.Get(spec)
	if err != nil {
		Stderrf("could not initialize a repo instance for %v: %v. check config", spec, err)
		return err
	}

	fmt.Printf("Checking resources of catalog: %s ...\n", spec)

	resFilter := model.ResourceFilter{
		Names: names,
		Types: []model.ResourceType{model.ResTypeTM},
	}

	var totalRes []CheckResult
	err = repo.RangeResources(ctx, resFilter, func(res model.Resource, visitErr error) bool {

		if visitErr != nil {
			totalRes = append(totalRes, CheckResult{typ: CheckErr, refName: res.Name, text: visitErr.Error()})
			return true
		}

		if res.Typ == model.ResTypeTM {
			res, cErr := checkThingModel(res)
			if cErr != nil {
				totalRes = append(totalRes, res)
				return true
			}
		}
		return true
	})

	if err != nil {
		Stderrf(err.Error())
	}

	for _, res := range totalRes {
		fmt.Println(res)

		if err == nil && res.typ == CheckErr {
			err = errCheckResourcesFailed
		}
	}
	return err
}

func checkThingModel(res model.Resource) (CheckResult, error) {
	_, err := validate.ValidateThingModel(res.Raw)
	if err != nil {
		return CheckResult{typ: CheckErr, refName: res.Name, text: err.Error()}, err
	}

	tm := &model.ThingModel{}
	err = json.Unmarshal(res.Raw, tm)
	if err != nil {
		return CheckResult{typ: CheckErr, refName: res.Name, text: err.Error()}, err
	}

	if tm.ID == "" {
		err = errors.New("TM id missing")
		return CheckResult{typ: CheckErr, refName: res.Name, text: err.Error()}, err
	}

	_, err = model.ParseTMID(tm.ID)
	if err != nil {
		return CheckResult{typ: CheckErr, refName: res.Name, text: err.Error()}, err
	}

	if res.RelPath != tm.ID {
		err = errors.New("TM id does not match resource location")
		return CheckResult{typ: CheckErr, refName: res.RelPath, text: err.Error()}, err
	}

	return CheckResult{typ: CheckOK, refName: res.Name, text: fmt.Sprintf("check successful")}, nil
}

func CheckIndex(ctx context.Context, spec model.RepoSpec) error {

	repo, err := repos.Get(spec)
	if err != nil {
		Stderrf("could not initialize a repo instance for %v: %v. check config", spec, err)
		return err
	}

	fmt.Printf("Checking index of catalog (%s) ...\n", spec)

	err = repo.AnalyzeIndex(ctx)
	if err != nil {
		Stderrf(err.Error())
	}
	return err
}
