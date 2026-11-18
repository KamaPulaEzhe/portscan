package portscan

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

func localListener(t *testing.T) (net.Listener, int) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on loopback: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l, l.Addr().(*net.TCPAddr).Port
}

func readResults(t *testing.T, ch <-chan Result) []Result {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	var results []Result
	for {
		select {
		case r, ok := <-ch:
			if !ok {
				return results
			}
			results = append(results, r)
		case <-timer.C:
			t.Fatal("results channel did not close")
		}
	}
}

func TestScanOpenAndDeduplication(t *testing.T) {
	_, port := localListener(t)
	scanner, err := New(WithConcurrency(2), WithConnectTimeout(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ch, err := scanner.Scan(ctx, []string{"127.0.0.1", " 127.0.0.1 "}, []int{port, port})
	if err != nil {
		t.Fatal(err)
	}
	got := readResults(t, ch)
	if len(got) != 1 {
		t.Fatalf("got %d results, want 1: %+v", len(got), got)
	}
	r := got[0]
	if r.Host != "127.0.0.1" || !r.IP.Equal(net.ParseIP("127.0.0.1")) || int(r.Port) != port || r.State != Open || r.Err != nil || r.Duration < 0 {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestScanClosed(t *testing.T) {
	l, port := localListener(t)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	scanner, err := New()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ch, err := scanner.Scan(ctx, []string{"127.0.0.1"}, []int{port})
	if err != nil {
		t.Fatal(err)
	}
	got := readResults(t, ch)
	if len(got) != 1 || got[0].State != Closed || got[0].Err == nil {
		t.Fatalf("got %+v, want one closed result with error", got)
	}
}

func TestScanValidation(t *testing.T) {
	scanner, err := New()
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, expire := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer expire()
	tests := []struct {
		name  string
		ctx   context.Context
		hosts []string
		ports []int
		want  error
	}{
		{"nil context", nil, []string{"127.0.0.1"}, []int{1}, nil},
		{"canceled", canceled, []string{"127.0.0.1"}, []int{1}, context.Canceled},
		{"expired", expired, []string{"127.0.0.1"}, []int{1}, context.DeadlineExceeded},
		{"empty hosts", context.Background(), nil, []int{1}, nil},
		{"blank host", context.Background(), []string{"  "}, []int{1}, nil},
		{"empty ports", context.Background(), []string{"127.0.0.1"}, nil, nil},
		{"invalid port", context.Background(), []string{"127.0.0.1"}, []int{65536}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch, err := scanner.Scan(tt.ctx, tt.hosts, tt.ports)
			if err == nil || ch != nil {
				t.Fatalf("Scan() = %v, %v; want nil channel and error", ch, err)
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestScanCancelWithoutReader(t *testing.T) {
	l, port := localListener(t)
	accepted := make(chan error, 1)
	go func() {
		c, err := l.Accept()
		if err == nil {
			_ = c.Close()
		}
		accepted <- err
	}()
	scanner, err := New(WithConcurrency(1))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := scanner.Scan(ctx, []string{"127.0.0.1"}, []int{port})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-accepted:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not connect")
	}
	// No result has been received before cancellation.
	cancel()
	_ = readResults(t, ch)
}

func TestConcurrentScans(t *testing.T) {
	_, port := localListener(t)
	scanner, err := New(WithConcurrency(2))
	if err != nil {
		t.Fatal(err)
	}
	const count = 8
	var wg sync.WaitGroup
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			ch, err := scanner.Scan(ctx, []string{"127.0.0.1"}, []int{port})
			if err != nil {
				t.Errorf("Scan: %v", err)
				return
			}
			var results []Result
			for r := range ch {
				results = append(results, r)
			}
			if len(results) != 1 || results[0].State != Open {
				t.Errorf("got %+v, want one open result", results)
			}
		}()
	}
	wg.Wait()
}
