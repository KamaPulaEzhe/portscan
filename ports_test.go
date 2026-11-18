package portscan

import (
	"reflect"
	"testing"
)

func TestRange(t *testing.T) {
	cases := []struct {
		name       string
		start, end int
		want       []int
		wantErr    bool
	}{
		{"one", 1, 1, []int{1}, false},
		{"maximum", 65535, 65535, []int{65535}, false},
		{"inclusive", 80, 82, []int{80, 81, 82}, false},
		{"zero start", 0, 1, nil, true},
		{"zero end", 1, 0, nil, true},
		{"negative", -1, 10, nil, true},
		{"above maximum", 1, 65536, nil, true},
		{"start above maximum", 65536, 65536, nil, true},
		{"reversed", 1000, 1, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Range(tc.start, tc.end)
			if tc.wantErr {
				if err == nil || got != nil {
					t.Fatalf("got %v, %v; want nil, error", got, err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, %v; want %v, nil", got, err, tc.want)
			}
		})
	}
}
