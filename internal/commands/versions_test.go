package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func TestVersionsCommand_ListVersions(t *testing.T) {
	rm := remotes.NewMockRemoteManager(t)
	r1 := remotes.NewMockRemote(t)
	r2 := remotes.NewMockRemote(t)
	rm.On("All").Return([]remotes.Remote{r1, r2}, nil)
	r1.On("Versions", "senseall").Return(
		model.FoundEntry{
			Name:         "omnicorp/senseall",
			Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
			Mpn:          "senseall",
			Author:       model.SchemaAuthor{Name: "omnicorp"},
			Versions: []model.FoundVersion{
				{
					TOCVersion: model.TOCVersion{
						TMID: "omnicorp/senseall/v0.36.0-20231231153548-243d1b462ccc.tm.json",
					},
					FoundIn: "r1",
				},
				{
					TOCVersion: model.TOCVersion{
						TMID: "omnicorp/senseall/v0.35.0-20231230153548-243d1b462bbb.tm.json",
					},
					FoundIn: "r1",
				},
			},
		}, nil)
	r2.On("Versions", "senseall").Return(model.FoundEntry{

		Name:         "omnicorp/senseall",
		Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
		Mpn:          "senseall",
		Author:       model.SchemaAuthor{Name: "omnicorp"},
		Versions: []model.FoundVersion{
			{
				TOCVersion: model.TOCVersion{
					TMID: "omnicorp/senseall/v0.34.0-20231130153548-243d1b462aaa.tm.json",
				},
				FoundIn: "r2",
			},
			{
				TOCVersion: model.TOCVersion{
					TMID: "omnicorp/senseall/v0.35.0-20231230173548-243d1b462bbb.tm.json",
				},
				FoundIn: "r2",
			},
		},
	}, nil)

	c := NewVersionsCommand(rm)
	res, err := c.ListVersions("", "senseall")

	assert.NoError(t, err)
	assert.Len(t, res.Versions, 3)
	assert.Equal(t, []model.FoundVersion{
		{
			TOCVersion: model.TOCVersion{TMID: "omnicorp/senseall/v0.34.0-20231130153548-243d1b462aaa.tm.json"},
			FoundIn:    "r2",
		},
		{
			TOCVersion: model.TOCVersion{TMID: "omnicorp/senseall/v0.35.0-20231230173548-243d1b462bbb.tm.json"},
			FoundIn:    "r2",
		},
		{
			TOCVersion: model.TOCVersion{TMID: "omnicorp/senseall/v0.36.0-20231231153548-243d1b462ccc.tm.json"},
			FoundIn:    "r1",
		},
	}, res.Versions)
}
