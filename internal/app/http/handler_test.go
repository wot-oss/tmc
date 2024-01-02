package http

import "testing"

func Test_getRelativeDepth(t *testing.T) {
	type args struct {
		path        string
		siblingPath string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"", args{"/inventory", "/inventory"}, 0},
		{"", args{"/long/path/to/inventory", "/inventory"}, 0},
		{"", args{"/somewhere/inventory/long/way/down", "/inventory"}, 3},
		{"", args{"/inventory/something", "/inventory"}, 1},
		{"", args{"/unrelated/path", "/inventory"}, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRelativeDepth(tt.args.path, tt.args.siblingPath); got != tt.want {
				t.Errorf("getRelativeDepth() = %v, want %v", got, tt.want)
			}
		})
	}
}
