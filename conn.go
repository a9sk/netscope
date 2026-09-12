package main

import (
	"fmt"
	"strings"
	"sync"
	"syscall"

	gsnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

const maxConnections = 200

type Connection struct {
	LocalAddr  string
	RemoteAddr string
	Status     string
	PID        int32
	Name       string
	Protocol   string
}

func formatConnAddr(ip string, port uint32) string {
	if strings.Contains(ip, ":") {
		return fmt.Sprintf("[%s]:%d", ip, port)
	}
	return fmt.Sprintf("%s:%d", ip, port)
}

var procNameCache sync.Map

func processName(pid int32) string {
	if pid == 0 {
		return ""
	}
	if name, ok := procNameCache.Load(pid); ok {
		return name.(string)
	}
	p, err := process.NewProcess(pid)
	if err != nil {
		return ""
	}
	name, err := p.Name()
	if err != nil {
		return ""
	}
	procNameCache.Store(pid, name)
	return name
}

func fetchConnections() ([]Connection, int, int, int, error) {
	conns, err := gsnet.Connections("all")
	if err != nil {
		return nil, 0, 0, 0, err
	}

	listeningPorts := make(map[uint32]struct{})
	for _, c := range conns {
		if c.Status == "LISTEN" {
			listeningPorts[c.Laddr.Port] = struct{}{}
		}
	}

	incoming := 0
	outgoing := 0
	active := make([]Connection, 0, len(conns))
	for _, c := range conns {
		if c.Status == "LISTEN" {
			continue
		}

		proto := "TCP"
		if c.Type == syscall.SOCK_DGRAM {
			proto = "UDP"
		}

		local := formatConnAddr(c.Laddr.IP, c.Laddr.Port)
		remote := formatConnAddr(c.Raddr.IP, c.Raddr.Port)

		if c.Raddr.IP == "" || c.Raddr.IP == "0.0.0.0" || c.Raddr.IP == "::" {
			continue
		}

		active = append(active, Connection{
			LocalAddr:  local,
			RemoteAddr: remote,
			Status:     c.Status,
			PID:        c.Pid,
			Name:       processName(c.Pid),
			Protocol:   proto,
		})

		if c.Status == "ESTABLISHED" {
			if _, ok := listeningPorts[c.Laddr.Port]; ok {
				incoming++
			} else {
				outgoing++
			}
		}
	}

	if len(active) > maxConnections {
		active = active[:maxConnections]
	}
	return active, incoming, outgoing, len(listeningPorts), nil
}
