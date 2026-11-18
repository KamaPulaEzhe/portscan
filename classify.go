package portscan

import (
	"context"
	"errors"
	"net"
	"syscall"
)

func classify(ctx context.Context, err error) State {
	if err == nil {
		return Open
	}
	if ctx.Err() != nil {
		return Canceled
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return Closed
	}
	if errors.Is(err, syscall.EHOSTUNREACH) || errors.Is(err, syscall.ENETUNREACH) {
		return Unreachable
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return Timeout
	}
	return Error
}
