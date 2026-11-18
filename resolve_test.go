package portscan

import (
	"context"
	"net"
	"testing"
)

func TestResolveHostIP(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "::1"} {
		t.Run(host, func(t *testing.T) {
			got, err := resolveHost(context.Background(), host)
			if err != nil || len(got) != 1 || !got[0].Equal(net.ParseIP(host)) {
				t.Fatalf("resolveHost(%q) = %v, %v", host, got, err)
			}
		})
	}
}

func TestResolveTargetsPreservesHost(t *testing.T) {
	host := "0:0:0:0:0:0:0:1"
	got, err := resolveTargets(context.Background(), []string{host, host})
	if err != nil || len(got) != 1 {
		t.Fatalf("resolveTargets() = %v, %v", got, err)
	}
	if got[0].host != host || !got[0].ip.Equal(net.ParseIP(host)) {
		t.Fatalf("unexpected target: %+v", got[0])
	}
}
