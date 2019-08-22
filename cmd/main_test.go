package main

import "testing"

func TestShortHash(t *testing.T) {
	type args struct {
		hash string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "empty", args: args{hash: ""}, want: ""},
		{name: "< 7 chars", args: args{hash: "d5a"}, want: "d5a"},
		{name: "= 7 chars", args: args{hash: "d5adef4"}, want: "d5adef4"},
		{name: "> 7 chars", args: args{hash: "d5adef4bef"}, want: "d5adef4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shortHash(tt.args.hash); got != tt.want {
				t.Errorf("shortHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
