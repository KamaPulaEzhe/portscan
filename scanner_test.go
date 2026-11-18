package portscan

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		opts        []Option
		concurrency int
		timeout     time.Duration
		wantErr     bool
	}{
		{name: "defaults", concurrency: 10, timeout: time.Second},
		{name: "custom options", opts: []Option{WithConcurrency(25), WithConnectTimeout(250 * time.Millisecond)}, concurrency: 25, timeout: 250 * time.Millisecond},
		{name: "zero concurrency", opts: []Option{WithConcurrency(0)}, wantErr: true},
		{name: "negative concurrency", opts: []Option{WithConcurrency(-1)}, wantErr: true},
		{name: "zero timeout", opts: []Option{WithConnectTimeout(0)}, wantErr: true},
		{name: "negative timeout", opts: []Option{WithConnectTimeout(-time.Second)}, wantErr: true},
		{name: "nil option", opts: []Option{nil}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("New panicked: %v", r)
				}
			}()
			s, err := New(tt.opts...)
			if tt.wantErr {
				if err == nil || s != nil {
					t.Fatalf("New() = (%v, %v), want (nil, error)", s, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("New() returned unexpected error: %v", err)
			}
			if s == nil {
				t.Fatal("New() returned nil scanner without an error")
			}
			if s.concurrency != tt.concurrency || s.connectTimeout != tt.timeout {
				t.Errorf("got concurrency %d and timeout %v; want %d and %v", s.concurrency, s.connectTimeout, tt.concurrency, tt.timeout)
			}
		})
	}
}
