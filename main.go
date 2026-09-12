package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	iface := flag.String("i", "any", "network interface to capture and monitor")
	portStr := flag.String("p", "", "port filter, e.g. 80 or \"tcp port 80\"")
	flag.Parse()

	portFilter := normalizePortFilter(*portStr)

	p := tea.NewProgram(initialModel(*iface, portFilter), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func normalizePortFilter(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Allow bare numbers to be treated as a port filter for both TCP and UDP.
	if _, err := fmt.Sscanf(s, "%d", new(int)); err == nil && !strings.ContainsAny(s, " ") {
		return "port " + s
	}
	return s
}
