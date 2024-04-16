package model

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
)

func TestParseId(t *testing.T) {
	i1 := "author/manufacturer/mpn/v1.2.3-pre1-20231109150513-e86784632bf6.tm.json"
	id, err := ParseTMID(i1)

	assert.NoError(t, err)
	assert.Equal(t, "author/manufacturer/mpn", id.Name)
	assert.Equal(t, "v1.2.3-pre1", id.Version.Base.Original())

	i2 := "author/manufacturer/mpn/byfirmware/v1/v1.2.3-pre1-20231109150513-e86784632bf6.tm.json"
	id, err = ParseTMID(i2)

	assert.NoError(t, err)
	assert.Equal(t, "author/manufacturer/mpn/byfirmware/v1", id.Name)
	assert.Equal(t, "v1.2.3-pre1", id.Version.Base.Original())

	ids := []string{
		"author/manufacturer/mpn/v1.2.3-20231109150513-e86784632bf6.tm.js",
		"author/manufacturer/mpn/v1.2.3.tm.json",
	}
	for i, v := range ids {
		id, err = ParseTMID(v)
		assert.ErrorIs(t, err, ErrInvalidId, "incorrect error at %d", i)
	}
}

func TestParseTMVersion(t *testing.T) {
	v1 := "v1.2.3-pre1-20231109150513-e86784632bf6"
	tv1, err := ParseTMVersion(v1)
	assert.NoError(t, err)
	assert.Equal(t, "v1.2.3-pre1", tv1.Base.Original())
	assert.Equal(t, "20231109150513", tv1.Timestamp)
	assert.Equal(t, "e86784632bf6", tv1.Hash)

	v2 := "v1.2.3-20231109150513-e86784632bf6"
	tv2, err := ParseTMVersion(v2)
	assert.NoError(t, err)
	assert.Equal(t, "v1.2.3", tv2.Base.Original())
	assert.Equal(t, "20231109150513", tv2.Timestamp)
	assert.Equal(t, "e86784632bf6", tv2.Hash)

	v3 := "v1.2.3"
	_, err = ParseTMVersion(v3)
	assert.ErrorIs(t, err, ErrInvalidPseudoVersion)

	v4 := "1.2.3-20231109150513-e86784632bf6"
	_, err = ParseTMVersion(v4)
	assert.ErrorIs(t, err, ErrInvalidVersion)

}

func TestTMVersionFromOriginal(t *testing.T) {
	ts := []struct {
		v   string
		exp string
	}{
		{"", "v0.0.0"},
		{"1.2.3", "v1.2.3"},
		{"abc", "v0.0.0"},
		{"15", "v15.0.0"},
		{"v1.2.3", "v1.2.3"},
		{"1.2.3-alpha1", "v1.2.3-alpha1"},
	}

	for i, test := range ts {
		tv := TMVersionFromOriginal(test.v)
		assert.Equal(t, test.exp, tv.String(), "wrong tm version at %d", i)
	}
}

func TestTMID_String(t *testing.T) {
	id := NewTMID("author", "manufacturer", "mpn", "byfirmware/v1",
		TMVersion{
			Base:      semver.MustParse("v1.2.3"),
			Timestamp: "20241243052343",
			Hash:      "ab1234567890",
		})

	assert.Equal(t, "author/manufacturer/mpn/byfirmware/v1/v1.2.3-20241243052343-ab1234567890.tm.json", id.String())

	id2 := NewTMID("manufacturer", "manufacturer", "mpn", "byfirmware/v1",
		TMVersion{
			Base:      semver.MustParse("v1.2.3"),
			Timestamp: "20241243052343",
			Hash:      "ab1234567890",
		})

	assert.Equal(t, "manufacturer/manufacturer/mpn/byfirmware/v1/v1.2.3-20241243052343-ab1234567890.tm.json", id2.String())
}
