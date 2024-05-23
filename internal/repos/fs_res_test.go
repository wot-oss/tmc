package repos

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/testutils"
)

func TestFileRepo_RangeResources(t *testing.T) {
	// given: a file repository
	temp, _ := os.MkdirTemp("", "fr")
	defer os.RemoveAll(temp)
	r := &FileRepo{
		root: temp,
		spec: model.NewRepoSpec("fr"),
	}

	// and given: some resources in the file repository
	files := []string{
		"a.tm.json",
		"a/2.tm.json",
		"a/3.jpg",
		"b.json",
		"b/1.tm.json",
		"b/2.tm.json",
		"b/3.md",
		"b/bb/bbb/bbbb/1.tm.json",
		"z/zz",
	}
	for _, f := range files {
		if filepath.Ext(f) != "" {
			_ = testutils.CreateFile(temp, f, []byte(f))
		} else {
			_ = testutils.CreateDir(temp, f)
		}
	}

	type expResData struct {
		typ     model.ResourceType
		hasBody bool
		err     error
	}

	tests := []struct {
		resTypes     []model.ResourceType
		resNames     []string
		expResources map[string]expResData
	}{
		// range all resources by type filter (ThingModel)
		{[]model.ResourceType{model.ResTypeTM}, []string{},
			map[string]expResData{
				"a.tm.json":               {model.ResTypeTM, true, nil},
				"a/2.tm.json":             {model.ResTypeTM, true, nil},
				"b/1.tm.json":             {model.ResTypeTM, true, nil},
				"b/2.tm.json":             {model.ResTypeTM, true, nil},
				"b/bb/bbb/bbbb/1.tm.json": {model.ResTypeTM, true, nil},
			},
		},
		// range all resources without type filter but with by name filter
		{[]model.ResourceType{}, []string{"a.tm.json", "b/1.tm.json", "z/1.tm.json", "z/zz"},
			map[string]expResData{
				"a.tm.json":   {model.ResTypeTM, true, nil},
				"b/1.tm.json": {model.ResTypeTM, true, nil},
				"z/1.tm.json": {model.ResTypeTM, false, ErrResourceNotExists},
				"z/zz":        {model.ResTypeUnknown, false, ErrResourceInvalid},
			},
		},
		// range all resources with type filter (ThingModel) and with name filter
		{[]model.ResourceType{model.ResTypeTM}, []string{"a.tm.json", "b/1.tm.json", "z/1.tm.json", "z/zz"},
			map[string]expResData{
				"a.tm.json":   {model.ResTypeTM, true, nil},
				"b/1.tm.json": {model.ResTypeTM, true, nil},
				"z/1.tm.json": {model.ResTypeTM, false, ErrResourceNotExists},
			},
		},
		// range all resources with unknown type
		{[]model.ResourceType{model.ResTypeUnknown}, []string{},
			map[string]expResData{
				"a/3.jpg": {model.ResTypeUnknown, true, nil},
				"b.json":  {model.ResTypeUnknown, true, nil},
				"b/3.md":  {model.ResTypeUnknown, true, nil},
			},
		},
	}

	for i, test := range tests {
		filter := model.ResourceFilter{
			Names: test.resNames,
			Types: test.resTypes,
		}

		count := 0
		// when: range the resources with the given filter
		err := r.RangeResources(context.Background(), filter, func(res model.Resource, err error) bool {
			count++
			expResData := test.expResources[res.Name]
			// then: the found resource has the expected resource type
			assert.Equalf(t, expResData.typ, res.Typ, "in test %d for %s", i, res.Name)
			// and then: the found resource has the expected body
			if expResData.hasBody {
				assert.Equalf(t, []byte(res.Name), res.Raw, "in test %d for %s", i, res.Name)
			}
			// and then: if an error occurred, it is the expected one
			assert.ErrorIsf(t, err, expResData.err, "in test %d for %s", i, res.Name)
			return true
		})
		// and then: the number of found resources equals the expected number
		assert.Equalf(t, len(test.expResources), count, "in test %d", i)
		assert.NoError(t, err)
	}
}
