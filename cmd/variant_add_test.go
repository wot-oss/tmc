package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariantAddBatchRequiresOneParent(t *testing.T) {
	flag := variantAddBatchCmd.Flags().Lookup("tm-id")
	require.NotNil(t, flag)
	oldValue, oldChanged := flag.Value.String(), flag.Changed
	t.Cleanup(func() {
		require.NoError(t, flag.Value.Set(oldValue))
		flag.Changed = oldChanged
	})

	flag.Changed = false
	assert.ErrorContains(t, variantAddBatchCmd.ValidateRequiredFlags(), `"tm-id"`)
	require.NoError(t, variantAddBatchCmd.Flags().Set("tm-id", "parent-tm-id"))
	assert.NoError(t, variantAddBatchCmd.ValidateRequiredFlags())
}

func TestVariantAddAllowsOmittingFamilyName(t *testing.T) {
	for _, name := range []string{"tm-id", "family-name"} {
		flag := variantAddCmd.Flags().Lookup(name)
		require.NotNil(t, flag)
		oldValue, oldChanged := flag.Value.String(), flag.Changed
		t.Cleanup(func() {
			require.NoError(t, flag.Value.Set(oldValue))
			flag.Changed = oldChanged
		})
		flag.Changed = false
	}
	require.NoError(t, variantAddCmd.Flags().Set("tm-id", "parent-tm-id"))
	assert.NoError(t, variantAddCmd.ValidateRequiredFlags())
	require.NoError(t, variantAddCmd.Flags().Set("family-name", "Eagle"))
	assert.NoError(t, variantAddCmd.ValidateRequiredFlags())
}
