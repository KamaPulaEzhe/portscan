package portscan

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

type dialerFunc func(context.Context, string, string) (net.Conn, error)

func (f dialerFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return f(ctx, network, address)
}

func TestScanDialErrors(t *testing.T) {
	tests := []struct {
		name    string
		dialErr error
		want    State
	}{
		{"timeout", context.DeadlineExceeded, Timeout},
		{"host unreachable", syscall.EHOSTUNREACH, Unreachable},
		{"network unreachable", syscall.ENETUNREACH, Unreachable},
		{"refused", syscall.ECONNREFUSED, Closed},
		{"other", errors.New("unexpected"), Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(WithConcurrency(1))
			if err != nil {
				t.Fatal(err)
			}
			expected := fmt.Errorf("dial: %w", tt.dialErr)
			s.dialer = dialerFunc(func(_ context.Context, network, address string) (net.Conn, error) {
				if network != "tcp" || address != "127.0.0.1:1234" {
					t.Errorf("dialed %q %q", network, address)
				}
				return nil, expected
			})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			ch, err := s.Scan(ctx, []string{"127.0.0.1"}, []int{1234})
			if err != nil {
				t.Fatal(err)
			}
			got := readResults(t, ch)
			if len(got) != 1 {
				t.Fatalf("got %d results, want 1", len(got))
			}
			if got[0].State != tt.want || !errors.Is(got[0].Err, tt.dialErr) {
				t.Fatalf("got state %q, err %v; want %q wrapping %v", got[0].State, got[0].Err, tt.want, tt.dialErr)
			}
		})
	}
}

func TestScanCancelInFlight(t *testing.T) {
	started := make(chan struct{})
	exited := make(chan struct{})
	s, err := New(WithConcurrency(1))
	if err != nil {
		t.Fatal(err)
	}
	s.dialer = dialerFunc(func(ctx context.Context, _, _ string) (net.Conn, error) {
		close(started)
		<-ctx.Done()
		close(exited)
		return nil, ctx.Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := s.Scan(ctx, []string{"127.0.0.1"}, []int{1234})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("dial did not start")
	}
	cancel()
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("dial did not observe cancellation")
	}
	results := readResults(t, ch)
	if len(results) > 1 {
		t.Fatalf("got %d results, want at most 1", len(results))
	}
	if len(results) == 1 && (results[0].State != Canceled || !errors.Is(results[0].Err, context.Canceled)) {
		t.Fatalf("unexpected result: %+v", results[0])
	}
}

func TestScanConcurrencyLimit(t *testing.T) {
	const limit = 2
	started := make(chan struct{}, limit)
	release := make(chan struct{})
	var active, maxActive atomic.Int32
	s, err := New(WithConcurrency(limit))
	if err != nil {
		t.Fatal(err)
	}
	s.dialer = dialerFunc(func(ctx context.Context, _, _ string) (net.Conn, error) {
		n := active.Add(1)
		defer active.Add(-1)
		for {
			old := maxActive.Load()
			if n <= old || maxActive.CompareAndSwap(old, n) {
				break
			}
		}
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-release:
			return nil, syscall.ECONNREFUSED
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ch, err := s.Scan(ctx, []string{"127.0.0.1"}, []int{1, 2, 3, 4, 5})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < limit; i++ {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal("workers did not start")
		}
	}
	close(release)
	got := readResults(t, ch)
	if len(got) != 5 {
		t.Fatalf("got %d results, want 5", len(got))
	}
	if max := maxActive.Load(); max != limit {
		t.Fatalf("maximum active dials = %d, want %d", max, limit)
	}
	for _, r := range got {
		if r.State != Closed {
			t.Errorf("got state %q, want closed", r.State)
		}
	}
}
