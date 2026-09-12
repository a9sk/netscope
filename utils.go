package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerBg = lipgloss.Color("#1e1e2e")
	panelBg  = lipgloss.Color("#181825")
	footerBg = lipgloss.Color("#1e1e2e")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#89b4fa")).
			Background(headerBg)

	modeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f38ba8")).
			Background(headerBg)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#cdd6f4")).
			Background(panelBg)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086")).
			Background(footerBg)

	inStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#a6e3a1")).
		Background(footerBg)

	outStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#89b4fa")).
			Background(footerBg)

	detailStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cdd6f4")).
			Background(lipgloss.Color("#313244"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086")).
			Background(footerBg)

	sparklineInStyle = lipgloss.NewStyle().
				Background(footerBg).
				Foreground(lipgloss.Color("#a6e3a1"))

	sparklineOutStyle = lipgloss.NewStyle().
				Background(footerBg).
				Foreground(lipgloss.Color("#89b4fa"))
)

func protocolStyle(proto string) lipgloss.Style {
	base := lipgloss.NewStyle().
		Background(panelBg).
		Foreground(lipgloss.Color("#cdd6f4"))

	switch proto {
	case "TCP":
		return base.Foreground(lipgloss.Color("#89b4fa"))
	case "UDP":
		return base.Foreground(lipgloss.Color("#a6e3a1"))
	case "DNS":
		return base.Foreground(lipgloss.Color("#f9e2af"))
	case "HTTP", "HTTP2", "QUIC":
		return base.Foreground(lipgloss.Color("#cba6f7"))
	case "TLS", "SSL", "HTTPS":
		return base.Foreground(lipgloss.Color("#fab387"))
	case "ICMP", "ICMPV6":
		return base.Foreground(lipgloss.Color("#f38ba8"))
	case "ARP":
		return base.Foreground(lipgloss.Color("#94e2d5"))
	default:
		return base
	}
}

func fillLine(text string, width int, bg lipgloss.Color) string {
	return lipgloss.NewStyle().Background(bg).Width(width).Render(text)
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) > w {
		return string(r[:w])
	}
	return s
}

func formatCount(n uint64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
}

func formatBytes(bps float64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case bps >= GB:
		return fmt.Sprintf("%.2f GB", bps/GB)
	case bps >= MB:
		return fmt.Sprintf("%.2f MB", bps/MB)
	case bps >= KB:
		return fmt.Sprintf("%.2f KB", bps/KB)
	default:
		return fmt.Sprintf("%.0f B", bps)
	}
}

func sparkline(values []float64, width int) string {
	if len(values) == 0 || width <= 0 {
		return ""
	}

	bars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var max float64
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	if max == 0 {
		max = 1
	}

	start := 0
	if len(values) > width {
		start = len(values) - width
	}

	var b strings.Builder
	for i := start; i < len(values); i++ {
		idx := int((values[i] / max) * float64(len(bars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(bars) {
			idx = len(bars) - 1
		}
		b.WriteRune(bars[idx])
	}

	return b.String()
}
