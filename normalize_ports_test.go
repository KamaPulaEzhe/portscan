package portscan

import (
	"reflect"
	"testing"
)

func TestNormalizePorts(t *testing.T) {
	tests := []struct {
		name string
		in   []int
		want []uint16
		bad  bool
	}{
		{"duplicates", []int{80, 443, 80, 81}, []uint16{80, 443, 81}, false},
		{"bounds", []int{1, 65535}, []uint16{1, 65535}, false},
		{"empty", nil, nil, true}, {"zero", []int{80, 0}, nil, true},
		{"negative", []int{-1}, nil, true}, {"too large", []int{65536}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizePorts(tt.in)
			if tt.bad {
				if err == nil || got != nil {
					t.Fatalf("got %v, %v; want nil, error", got, err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, %v; want %v, nil", got, err, tt.want)
			}
		})
	}
}
