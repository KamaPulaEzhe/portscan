package portscan

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

type Scanner struct {
	concurrency    int
	connectTimeout time.Duration
	dialer         dialer
}

type dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type Option func(*Scanner)

type job struct {
	target target
	port   uint16
}

func WithConcurrency(n int) Option {
	return func(s *Scanner) {
		s.concurrency = n
	}
}

func WithConnectTimeout(d time.Duration) Option {
	return func(s *Scanner) {
		s.connectTimeout = d
	}
}

func New(opts ...Option) (*Scanner, error) {
	s := &Scanner{
		concurrency:    10,
		connectTimeout: time.Second,
	}

	for i, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("option %d is nil", i)
		}
		opt(s)
	}
	if s.concurrency <= 0 {
		return nil, fmt.Errorf("concurrency must be positive: %d", s.concurrency)
	}
	if s.connectTimeout <= 0 {
		return nil, fmt.Errorf("connect timeout must be positive: %s", s.connectTimeout)
	}
	s.dialer = &net.Dialer{Timeout: s.connectTimeout}
	return s, nil
}

// Scan starts concurrent TCP checks and returns their results on a channel.
// It validates ports and resolves hosts before returning. On cancellation, it
// stops scheduling work and allows workers to exit without waiting for a
// reader. Results may be incomplete; not every canceled check produces a
// Canceled result. The caller should cancel ctx if it stops reading early.
// Once the channel closes, ctx.Err can be checked for cancellation or deadline.
func (s *Scanner) Scan(ctx context.Context, hosts []string, ports []int) (<-chan Result, error) {
	var wg sync.WaitGroup

	if ctx == nil {
		return nil, errors.New("ctx is nil")
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	portsNorm, err := normalizePorts(ports)
	if err != nil {
		return nil, err
	}
	targets, err := resolveTargets(ctx, hosts)
	if err != nil {
		return nil, err
	}
	results := make(chan Result)
	jobs := make(chan job)
	wg.Add(s.concurrency)

	for range s.concurrency {
		go worker(ctx, &wg, s.dialer, jobs, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		defer close(jobs)
		for _, target := range targets {
			for _, port := range portsNorm {
				task := job{
					target: target,
					port:   port,
				}
				select {
				case <-ctx.Done():
					return
				case jobs <- task:

				}
			}
		}
	}()

	return results, nil
}

func worker(ctx context.Context, wg *sync.WaitGroup, d dialer, jobs <-chan job, results chan<- Result) {
	defer wg.Done()

	for {
		select {
		case task, ok := <-jobs:
			if !ok {
				return
			}

			addr := net.JoinHostPort(task.target.ip.String(), strconv.Itoa(int(task.port)))

			t := time.Now()
			conn, err := d.DialContext(ctx, "tcp", addr)
			duration := time.Since(t)

			if err == nil {
				conn.Close()
			}

			result := Result{
				Host:     task.target.host,
				IP:       task.target.ip,
				Port:     task.port,
				State:    classify(ctx, err),
				Duration: duration,
				Err:      err,
			}

			select {
			case results <- result:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
