package main

import (
	"bufio"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const maxPackets = 500

type Packet struct {
	Time     time.Duration
	Protocol string
	Src      string
	Dst      string
	Length   int
}

type captureReadyMsg struct {
	ch  chan Packet
	err error
}

type captureDoneMsg struct{}

type packetMsg struct{ p Packet }

func startCaptureCmd(iface, portFilter string) tea.Cmd {
	return func() tea.Msg {
		ch, err := startTshark(iface, portFilter)
		return captureReadyMsg{ch: ch, err: err}
	}
}

func packetCmd(ch <-chan Packet) tea.Cmd {
	return func() tea.Msg {
		p, ok := <-ch
		if !ok {
			return captureDoneMsg{}
		}
		return packetMsg{p: p}
	}
}

func startTshark(iface, portFilter string) (chan Packet, error) {
	if _, err := exec.LookPath("tshark"); err != nil {
		return nil, err
	}

	if iface == "" {
		iface = "any"
	}

	args := []string{
		"-i", iface,
		"-T", "fields",
		"-e", "frame.time_relative",
		"-e", "ip.src",
		"-e", "ip.dst",
		"-e", "tcp.srcport",
		"-e", "tcp.dstport",
		"-e", "udp.srcport",
		"-e", "udp.dstport",
		"-e", "frame.len",
		"-e", "_ws.col.Protocol",
		"-l",
	}
	if portFilter != "" {
		args = append(args, "-f", portFilter)
	}

	cmd := exec.Command("tshark", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	ch := make(chan Packet, 200)
	go func() {
		defer close(ch)
		go io.Copy(io.Discard, stderr)

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if p, ok := parseTsharkLine(line); ok {
				select {
				case ch <- p:
				default:
					<-ch
					ch <- p
				}
			}
		}
		cmd.Wait()
	}()

	return ch, nil
}

func parseTsharkLine(line string) (Packet, bool) {
	fields := strings.Split(line, "\t")
	if len(fields) < 9 {
		return Packet{}, false
	}

	ts, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return Packet{}, false
	}

	length, _ := strconv.Atoi(fields[7])
	proto := strings.ToUpper(fields[8])

	src := formatAddr(fields[1], fields[3], fields[5])
	dst := formatAddr(fields[2], fields[4], fields[6])

	return Packet{
		Time:     time.Duration(ts * float64(time.Second)),
		Protocol: proto,
		Src:      src,
		Dst:      dst,
		Length:   length,
	}, true
}

func formatAddr(ip, tcpPort, udpPort string) string {
	if ip == "" {
		return "?"
	}
	port := tcpPort
	if port == "" {
		port = udpPort
	}
	if port == "" {
		return ip
	}
	if strings.Contains(ip, ":") {
		return "[" + ip + "]:" + port
	}
	return ip + ":" + port
}
