package portscan

import (
	"reflect"
	"testing"
)

func TestNormalizeHosts(t *testing.T) {
	cases := []struct {
		name     string
		in, want []string
		bad      bool
	}{
		{"empty list", nil, nil, true}, {"blank", []string{"   "}, nil, true},
		{"empty element", []string{"valid", ""}, nil, true},
		{"trim duplicates order", []string{" 127.0.0.1 ", "::1", "127.0.0.1"}, []string{"127.0.0.1", "::1"}, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeHosts(tt.in)
			if tt.bad {
				if err == nil || got != nil {
					t.Fatalf("got %q %v, want nil and error", got, err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %q %v, want %q", got, err, tt.want)
			}
		})
	}
}
