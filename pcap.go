package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
)

const defaultPcapFile = "netscope.pcap"

type pcapState struct {
	cmd      *exec.Cmd
	filename string
	started  time.Time
}

func startPcapCapture(iface, portFilter string) (*pcapState, error) {
	if _, err := exec.LookPath("tshark"); err != nil {
		return nil, err
	}

	if iface == "" || iface == "any" {
		// pcap writes require a concrete interface on some tshark builds.
		// Fallback to the first non-loopback interface when possible.
		iface = pickInterface()
	}

	args := []string{"-i", iface, "-w", defaultPcapFile}
	if portFilter != "" {
		args = append(args, "-f", portFilter)
	}

	cmd := exec.Command("tshark", args...)
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &pcapState{
		cmd:      cmd,
		filename: defaultPcapFile,
		started:  time.Now(),
	}, nil
}

func stopPcapCapture(s *pcapState) error {
	if s == nil || s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
		_ = s.cmd.Process.Kill()
		return err
	}
	return s.cmd.Wait()
}

func pickInterface() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "any"
	}
	for _, iface := range ifaces {
		name := iface.Name
		if name != "" && name != "lo" {
			return name
		}
	}
	return "any"
}

func pcapStatus(s *pcapState) string {
	if s == nil || s.cmd == nil || s.cmd.Process == nil {
		return ""
	}
	return fmt.Sprintf("pcap %s (%s)", s.filename, time.Since(s.started).Round(time.Second))
}
