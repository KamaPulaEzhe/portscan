package portscan

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

type target struct {
	host string
	ip   net.IP
}

func normalizeHosts(hosts []string) ([]string, error) {
	if len(hosts) == 0 {
		return nil, errors.New("empty hosts slice")
	}
	var result []string
	seen := make(map[string]bool)

	for _, h := range hosts {
		host := strings.TrimSpace(h)
		if host == "" {
			return nil, errors.New("empty host")
		}
		if v := seen[host]; v {
			continue
		}
		result = append(result, host)
		seen[host] = true
	}
	return result, nil
}

func resolveHost(ctx context.Context, host string) ([]net.IP, error) {
	ip := net.ParseIP(host)
	if ip != nil {
		return []net.IP{ip}, nil
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := []net.IP{}
	for _, addr := range addresses {
		ips = append(ips, addr.IP)
	}
	return ips, nil
}

func resolveTargets(ctx context.Context, hosts []string) ([]target, error) {
	hostsNorm, err := normalizeHosts(hosts)
	if err != nil {
		return nil, err
	}
	targets := []target{}
	for _, host := range hostsNorm {
		ips, err := resolveHost(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolve host %q: %w", host, err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("resolve host %q: no IP addresses found", host)
		}
		for _, ip := range ips {
			targetIP := target{
				host: host,
				ip:   ip,
			}
			targets = append(targets, targetIP)
		}
	}
	return targets, nil
}
