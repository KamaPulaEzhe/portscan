package portscan

import (
	"errors"
	"fmt"
)

func Range(start, end int) ([]int, error) {
	var ports []int
	if start <= 0 || start > 65535 {
		return nil, fmt.Errorf("range start %d must be between 1 and 65535", start)
	}
	if end <= 0 || end > 65535 {
		return nil, fmt.Errorf("range end %d must be between 1 and 65535", end)
	}
	if start > end {
		return nil, fmt.Errorf("range start %d exceeds end %d", start, end)
	}
	for p := start; p <= end; p++ {
		ports = append(ports, p)
	}
	return ports, nil
}

func normalizePorts(ports []int) ([]uint16, error) {
	if len(ports) == 0 {
		return nil, errors.New("empty ports slice")
	}
	var result []uint16
	seen := make(map[int]bool)

	for _, p := range ports {
		if p <= 0 || p > 65535 {
			return nil, fmt.Errorf("port %d must be between 1 and 65535", p)
		}

		if v := seen[p]; v {
			continue
		}
		result = append(result, uint16(p))
		seen[p] = true
	}
	return result, nil
}
