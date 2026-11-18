package portscan

import (
	"net"
	"time"
)

type State string

const Open State = "open"
const Closed State = "closed"
const Timeout State = "timeout"
const Unreachable State = "unreachable"
const Canceled State = "canceled"
const Error State = "error"

type Result struct {
	Host     string
	IP       net.IP
	Port     uint16
	State    State
	Duration time.Duration
	Err      error
}
