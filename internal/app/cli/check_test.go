package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

func TestCheck_Resources(t *testing.T) {
	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), r, nil))

	t.Run("with only valid ThingModels", func(t *testing.T) {
		restore, getStdout := testutils.ReplaceStdout()
		defer restore()

		// given: some ThingModels found in a repository
		tms := []model.Resource{
			{ // correct TM
				Name:    "mycompany/bartech/bazlamp/v0.0.1-20240101120000-1fc13316b7d8.tm.json",
				RelPath: "mycompany/bartech/bazlamp/v0.0.1-20240101120000-1fc13316b7d8.tm.json", Typ: model.ResTypeTM, Raw: []byte(tm1),
			},
			{ // another correct TM
				Name:    "yourcompany/bartech/bazlamp/v0.0.1-20240101120000-1fc13316b7d8.tm.json",
				RelPath: "yourcompany/bartech/bazlamp/v0.0.1-20240101120000-1fc13316b7d8.tm.json", Typ: model.ResTypeTM, Raw: []byte(tm6),
			},
		}

		r.On("RangeResources", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			visit := args.Get(2).(func(resource model.Resource, err error) bool)
			for _, tm := range tms {
				visit(tm, nil)
			}
		}).Return(nil).Once()

		// when: checking the given ThingModels
		err := CheckResources(context.Background(), model.NewRepoSpec("r1"), nil)
		stdout := getStdout()

		// then: there is no total error
		assert.NoError(t, err)
		// and then: stdout does not contain any error
		assert.NotContains(t, stdout, CheckErr.String())
	})

	t.Run("with some invalid ThingModels", func(t *testing.T) {
		restore, getStdout := testutils.ReplaceStdout()
		defer restore()

		// given: some ThingModels found in a repository
		tms := []model.Resource{
			{ // correct TM
				Name:    "mycompany/bartech/bazlamp/v0.0.1-20240101120000-1fc13316b7d8.tm.json",
				RelPath: "mycompany/bartech/bazlamp/v0.0.1-20240101120000-1fc13316b7d8.tm.json", Typ: model.ResTypeTM, Raw: []byte(tm1),
			},
			{ // missing id
				Name:    "mycompany/bartech/bazlamp/v0.0.2-20240101130000-2fc13316b7d8.tm.json",
				RelPath: "mycompany/bartech/bazlamp/v0.0.2-20240101130000-2fc13316b7d8.tm.json", Typ: model.ResTypeTM, Raw: []byte(tm2),
			},
			{ // invalid id
				Name:    "mycompany/bartech/bazlamp/v0.0.3-20240101140000-3fc13316b7d8.tm.json",
				RelPath: "mycompany/bartech/bazlamp/v0.0.3-20240101140000-3fc13316b7d8.tm.json", Typ: model.ResTypeTM, Raw: []byte(tm3),
			},
			{ // missing MPN
				Name:    "mycompany/bartech/bazlamp/v0.0.4-20240101150000-4fc13316b7d8.tm.json",
				RelPath: "mycompany/bartech/bazlamp/v0.0.4-20240101150000-4fc13316b7d8.tm.json", Typ: model.ResTypeTM, Raw: []byte(tm4),
			},
			{ // invalid json
				Name:    "mycompany/bartech/bazlamp/v0.0.5-20240101160000-5fc13316b7d8.tm.json",
				RelPath: "mycompany/bartech/bazlamp/v0.0.5-20240101160000-5fc13316b7d8.tm.json", Typ: model.ResTypeTM, Raw: []byte(tm5),
			},
			{ // id does not match resource location
				Name:    "mycompany/bartech/bazlamp/v0.0.6-20240101160000-5fc13316b7d8.tm.json",
				RelPath: "mycompany/bartech/bazlamp/v0.0.6-20240101160000-5fc13316b7d8.tm.json", Typ: model.ResTypeTM, Raw: []byte(tm1),
			},
		}

		r.On("RangeResources", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			visit := args.Get(2).(func(resource model.Resource, err error) bool)
			for _, tm := range tms {
				visit(tm, nil)
			}
		}).Return(nil).Once()

		// when: checking the given ThingModels
		err := CheckResources(context.Background(), model.NewRepoSpec("r1"), nil)
		stdout := getStdout()
		// then: there is a total error
		assert.ErrorIs(t, err, errCheckFailed)
		// and then: stdout shows no error for valid ThingModel
		assert.NotContains(t, stdout, CheckResult{typ: CheckErr, refName: tms[0].Name, text: ""}.String())
		// and then: stdout shows errors for invalid ThingModels (no check for concrete error msg)
		assert.Contains(t, stdout, CheckResult{typ: CheckErr, refName: tms[1].Name, text: ""}.String())
		assert.Contains(t, stdout, CheckResult{typ: CheckErr, refName: tms[2].Name, text: ""}.String())
		assert.Contains(t, stdout, CheckResult{typ: CheckErr, refName: tms[3].Name, text: ""}.String())
		assert.Contains(t, stdout, CheckResult{typ: CheckErr, refName: tms[4].Name, text: ""}.String())
		assert.Contains(t, stdout, CheckResult{typ: CheckErr, refName: tms[5].Name, text: ""}.String())
	})

	t.Run("with visit error", func(t *testing.T) {
		restore, getStdout := testutils.ReplaceStdout()
		defer restore()

		// given: some ThingModels found in a repository
		tm := model.Resource{
			// correct TM
			Name:    "mycompany/bartech/bazlamp/v0.0.1-20240101120000-1fc13316b7d8.tm.json",
			RelPath: "mycompany/bartech/bazlamp/v0.0.1-20240101120000-1fc13316b7d8.tm.json", Typ: model.ResTypeTM, Raw: []byte(tm1),
		}

		someVisitError := errors.New("some visit error")
		r.On("RangeResources", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			visit := args.Get(2).(func(resource model.Resource, err error) bool)
			visit(tm, someVisitError)
		}).Return(nil).Once()

		// when: checking the given ThingModel
		err := CheckResources(context.Background(), model.NewRepoSpec("r1"), nil)
		stdout := getStdout()
		// then: there is a total error
		assert.ErrorIs(t, err, errCheckFailed)
		// and then: stdout shows error for invalid ThingModel (no check for concrete error msg)
		assert.Contains(t, stdout, CheckResult{typ: CheckErr, refName: tm.Name, text: ""}.String())
	})

	t.Run("with general RangeResources error", func(t *testing.T) {
		someError := errors.New("some error")
		r.On("RangeResources", mock.Anything, mock.Anything, mock.Anything).Return(someError).Once()
		err := CheckResources(context.Background(), model.NewRepoSpec("r1"), nil)
		assert.ErrorIs(t, err, someError)
	})

	t.Run("with repository not found", func(t *testing.T) {
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), nil, repos.ErrRepoNotFound))
		err := CheckResources(context.Background(), model.NewRepoSpec("r1"), nil)
		assert.ErrorIs(t, err, repos.ErrRepoNotFound)
	})
}

func TestCheck_Index(t *testing.T) {
	r := mocks.NewRepo(t)
	rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), r, nil))

	t.Run("with repository not found", func(t *testing.T) {
		rMocks.MockReposGet(t, rMocks.CreateMockGetFunction(t, model.NewRepoSpec("r1"), nil, repos.ErrRepoNotFound))
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"))
		assert.ErrorIs(t, err, repos.ErrRepoNotFound)
	})

	t.Run("without CheckIntegrity error", func(t *testing.T) {
		r.On("CheckIntegrity", mock.Anything).Return(nil).Once()
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"))
		assert.NoError(t, err)
	})

	t.Run("with CheckIntegrity error", func(t *testing.T) {
		r.On("CheckIntegrity", mock.Anything).Return(repos.ErrIndexMismatch).Once()
		err := CheckIntegrity(context.Background(), model.NewRepoSpec("r1"))
		assert.ErrorIs(t, err, repos.ErrIndexMismatch)
	})
}

// correct TM
var tm1 = `{
  "@context": [ "https://www.w3.org/2022/wot/td/v1.1", { "schema":"https://schema.org/" }],
  "@type": "tm:ThingModel",
  "id": "mycompany/bartech/bazlamp/v0.0.1-20240101120000-1fc13316b7d8.tm.json",
  "title": "Lamp Thing Model",
  "schema:author": { "schema:name": "MyCompany" },
  "schema:manufacturer": { "schema:name": "BarTech" },
  "schema:mpn": "BazLamp}",  
  "version": { "model": "v.0.0.1" }
}`

// id missing
var tm2 = `{
  "@context": [ "https://www.w3.org/2022/wot/td/v1.1", { "schema":"https://schema.org/" }],
  "@type": "tm:ThingModel", 
  "title": "Lamp Thing Model",
  "schema:author": { "schema:name": "MyCompany" },
  "schema:manufacturer": { "schema:name": "BarTech" },
  "schema:mpn": "BazLamp}",  
  "version": { "model": "v.0.0.2" }
}`

// invalid id
var tm3 = `{
  "@context": [ "https://www.w3.org/2022/wot/td/v1.1", { "schema":"https://schema.org/" }],
  "@type": "tm:ThingModel", 
  "id": "my-custom-id",
  "title": "Lamp Thing Model",
  "schema:author": { "schema:name": "MyCompany" },
  "schema:manufacturer": { "schema:name": "BarTech" },
  "schema:mpn": "BazLamp}",  
  "version": { "model": "v.0.0.3" }
}`

// missing MPN
var tm4 = `{
  "@context": [ "https://www.w3.org/2022/wot/td/v1.1", { "schema":"https://schema.org/" }],
  "@type": "tm:ThingModel",
  "id": "mycompany/bartech/bazlamp/v0.0.4-20240206142430-2cc13316b7d8.tm.json",
  "title": "Lamp Thing Model",
  "schema:author": { "schema:name": "MyCompany" },
  "schema:manufacturer": { "schema:name": "BarTech" }, 
  "version": { "model": "v.0.0.4" }
}`

// invalid json
var tm5 = `
  "@context": [ "https://www.w3.org/2022/wot/td/v1.1", { "schema":"https://schema.org/" }],
  "@type": "tm:ThingModel",
}`

// another correct TM
var tm6 = `{
  "@context": [ "https://www.w3.org/2022/wot/td/v1.1", { "schema":"https://schema.org/" }],
  "@type": "tm:ThingModel",
  "id": "yourcompany/bartech/bazlamp/v0.0.1-20240101120000-1fc13316b7d8.tm.json",
  "title": "Lamp Thing Model",
  "schema:author": { "schema:name": "YourCompany" },
  "schema:manufacturer": { "schema:name": "BarTech" },
  "schema:mpn": "BazLamp}",  
  "version": { "model": "v.0.0.1" }
}`
