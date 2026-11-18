package portscan

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"testing"
)

func TestClassify(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want State
	}{
		{"open", context.Background(), nil, Open},
		{"open after cancel", canceled, nil, Open},
		{"refused wrapped", context.Background(), fmt.Errorf("dial: %w", syscall.ECONNREFUSED), Closed},
		{"host unreachable", context.Background(), syscall.EHOSTUNREACH, Unreachable},
		{"network unreachable", context.Background(), syscall.ENETUNREACH, Unreachable},
		{"timeout wrapped", context.Background(), fmt.Errorf("dial: %w", context.DeadlineExceeded), Timeout},
		{"canceled priority", canceled, context.DeadlineExceeded, Canceled},
		{"other", context.Background(), errors.New("other"), Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classify(tt.ctx, tt.err); got != tt.want {
				t.Fatalf("classify() = %q, want %q", got, tt.want)
			}
		})
	}
}
