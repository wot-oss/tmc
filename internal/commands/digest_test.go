package commands

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateFileDigest(t *testing.T) {
	type args struct {
		raw []byte
	}
	tests := []struct {
		name      string
		args      args
		wantHash  string
		wantBytes []byte
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name:      "linux no id",
			args:      args{raw: []byte("{\n\"title\":\"test\"\n}")},
			wantHash:  "7ae21a619c71",
			wantBytes: []byte("{\n\"title\":\"test\"\n,\"id\":\"\"}"),
			wantErr:   assert.NoError,
		},
		{
			name:      "linux empty id",
			args:      args{raw: []byte("{\n\"title\":\"test\"\n,\"id\":\"\"}")},
			wantHash:  "7ae21a619c71",
			wantBytes: []byte("{\n\"title\":\"test\"\n,\"id\":\"\"}"),
			wantErr:   assert.NoError,
		},
		{
			name:      "linux our inserted id",
			args:      args{raw: []byte("{\n\"title\":\"test\"\n,\"id\":\"author/omnicorp/senseall/opt/dir/v3.2.1-20231110123243-863e9f0f950a.tm.json\"}")},
			wantHash:  "7ae21a619c71",
			wantBytes: []byte("{\n\"title\":\"test\"\n,\"id\":\"\"}"),
			wantErr:   assert.NoError,
		},
		{
			name:      "linux pre-existing empty id",
			args:      args{raw: []byte("{\n\"id\":\"\",\n\"title\":\"test\"\n}")},
			wantHash:  "60d900490eb6",
			wantBytes: []byte("{\n\"id\":\"\",\n\"title\":\"test\"\n}"),
			wantErr:   assert.NoError,
		},
		{
			name:      "linux pre-existing id",
			args:      args{raw: []byte("{\n\"id\":\"author/omnicorp/senseall/opt/dir/v3.2.1-20231110123243-863e9f0f950a.tm.json\",\n\"title\":\"test\"\n}")},
			wantHash:  "60d900490eb6",
			wantBytes: []byte("{\n\"id\":\"\",\n\"title\":\"test\"\n}"),
			wantErr:   assert.NoError,
		},
		{
			name:      "windows no id",
			args:      args{raw: []byte("{\r\n\"title\":\"test\"\r\n}")},
			wantHash:  "7ae21a619c71",
			wantBytes: []byte("{\n\"title\":\"test\"\n,\"id\":\"\"}"),
			wantErr:   assert.NoError,
		},
		{
			name:      "windows empty id",
			args:      args{raw: []byte("{\r\n\"title\":\"test\"\r\n,\"id\":\"\"}")},
			wantHash:  "7ae21a619c71",
			wantBytes: []byte("{\n\"title\":\"test\"\n,\"id\":\"\"}"),
			wantErr:   assert.NoError,
		},
		{
			name:      "windows pre-existing empty id",
			args:      args{raw: []byte("{\r\n\"id\":\"\",\r\n\"title\":\"test\"\r\n}")},
			wantHash:  "60d900490eb6",
			wantBytes: []byte("{\n\"id\":\"\",\n\"title\":\"test\"\n}"),
			wantErr:   assert.NoError,
		},
		{
			name:      "windows pre-existing id",
			args:      args{raw: []byte("{\r\n\"id\":\"author/omnicorp/senseall/v3.2.1-20231110123243-863e9f0f950a.tm.json\",\r\n\"title\":\"test\"\r\n}")},
			wantHash:  "60d900490eb6",
			wantBytes: []byte("{\n\"id\":\"\",\n\"title\":\"test\"\n}"),
			wantErr:   assert.NoError,
		},
		{
			name:      "broken json",
			args:      args{raw: []byte("\r\n\"id\":\"asdf\",\r\n\"title\":\"test\"\r\n}")},
			wantHash:  "",
			wantBytes: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return !assert.Error(t, err, i...)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHash, gotBytes, err := CalculateFileDigest(tt.args.raw)
			if !tt.wantErr(t, err, fmt.Sprintf("CalculateFileDigest(%v)", tt.args.raw)) {
				return
			}
			assert.Equalf(t, tt.wantHash, gotHash, "CalculateFileDigest(%v)", tt.args.raw)
			assert.Equalf(t, tt.wantBytes, gotBytes, "CalculateFileDigest(%v)", tt.args.raw)
		})
	}
}
