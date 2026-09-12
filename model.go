package main

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	gsnet "github.com/shirou/gopsutil/v4/net"
)

const (
	updateInterval = time.Second
	historySize    = 60
)

const (
	sortPacketTime = iota
	sortPacketProtocol
	sortPacketSrc
	sortPacketDst
	sortPacketLength
)

const (
	sortConnStatus = iota
	sortConnProtocol
	sortConnLocal
	sortConnRemote
	sortConnName
)

type netStats struct {
	incomingConns  int
	outgoingConns  int
	listening      int
	packetsIn      uint64
	packetsOut     uint64
	bytesIn        uint64
	bytesOut       uint64
	bytesInPerSec  float64
	bytesOutPerSec float64
	timestamp      time.Time
}

type tickMsg time.Time

type model struct {
	width    int
	height   int
	listRows int

	mode    string
	err     error
	pktChan chan Packet

	packets     []Packet
	connections []Connection

	iface      string
	portFilter string

	paused     bool
	filtering  bool
	filterInput string
	sortMode   int
	detailOpen bool

	pcap *pcapState

	stats      netStats
	prevStats  netStats
	historyIn  []float64
	historyOut []float64

	scrollOffset int
	selected     int
}

func initialModel(iface, portFilter string) model {
	return model{
		mode:       "connections",
		iface:      iface,
		portFilter: portFilter,
		historyIn:  make([]float64, 0, historySize),
		historyOut: make([]float64, 0, historySize),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		startCaptureCmd(m.iface, m.portFilter),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(updateInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) addPacket(p Packet) {
	m.packets = append(m.packets, p)
	if len(m.packets) > maxPackets {
		m.packets = m.packets[1:]
	}
	if !m.paused {
		m.selected = len(m.packets) - 1
		if m.selected < 0 {
			m.selected = 0
		}
	}
}

func (m *model) filteredPackets() []Packet {
	if m.filterInput == "" {
		return m.packets
	}
	needle := strings.ToLower(m.filterInput)
	out := make([]Packet, 0, len(m.packets))
	for _, p := range m.packets {
		if strings.Contains(strings.ToLower(p.Protocol), needle) ||
			strings.Contains(strings.ToLower(p.Src), needle) ||
			strings.Contains(strings.ToLower(p.Dst), needle) ||
			strings.Contains(strings.ToLower(strconv.Itoa(p.Length)), needle) {
			out = append(out, p)
		}
	}
	return out
}

func (m *model) filteredConnections() []Connection {
	if m.filterInput == "" {
		return m.connections
	}
	needle := strings.ToLower(m.filterInput)
	out := make([]Connection, 0, len(m.connections))
	for _, c := range m.connections {
		if strings.Contains(strings.ToLower(c.Status), needle) ||
			strings.Contains(strings.ToLower(c.Protocol), needle) ||
			strings.Contains(strings.ToLower(c.LocalAddr), needle) ||
			strings.Contains(strings.ToLower(c.RemoteAddr), needle) ||
			strings.Contains(strings.ToLower(c.Name), needle) ||
			strings.Contains(strings.ToLower(strconv.Itoa(int(c.PID))), needle) {
			out = append(out, c)
		}
	}
	return out
}

func (m *model) currentItems() []any {
	var items []any
	if m.mode == "tshark" {
		packets := m.sortedPackets(m.filteredPackets())
		items = make([]any, len(packets))
		for i := range packets {
			items[i] = packets[i]
		}
	} else {
		conns := m.sortedConnections(m.filteredConnections())
		items = make([]any, len(conns))
		for i := range conns {
			items[i] = conns[i]
		}
	}
	return items
}

func (m *model) sortedPackets(packets []Packet) []Packet {
	out := make([]Packet, len(packets))
	copy(out, packets)
	switch m.sortMode % 5 {
	case sortPacketTime:
		// Already ordered by arrival.
	case sortPacketProtocol:
		sort.SliceStable(out, func(i, j int) bool { return out[i].Protocol < out[j].Protocol })
	case sortPacketSrc:
		sort.SliceStable(out, func(i, j int) bool { return out[i].Src < out[j].Src })
	case sortPacketDst:
		sort.SliceStable(out, func(i, j int) bool { return out[i].Dst < out[j].Dst })
	case sortPacketLength:
		sort.SliceStable(out, func(i, j int) bool { return out[i].Length > out[j].Length })
	}
	return out
}

func (m *model) sortedConnections(conns []Connection) []Connection {
	out := make([]Connection, len(conns))
	copy(out, conns)
	switch m.sortMode % 5 {
	case sortConnStatus:
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Status != out[j].Status {
				return out[i].Status < out[j].Status
			}
			return out[i].RemoteAddr < out[j].RemoteAddr
		})
	case sortConnProtocol:
		sort.SliceStable(out, func(i, j int) bool { return out[i].Protocol < out[j].Protocol })
	case sortConnLocal:
		sort.SliceStable(out, func(i, j int) bool { return out[i].LocalAddr < out[j].LocalAddr })
	case sortConnRemote:
		sort.SliceStable(out, func(i, j int) bool { return out[i].RemoteAddr < out[j].RemoteAddr })
	case sortConnName:
		sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	}
	return out
}

func (m *model) adjustScroll() {
	items := m.currentItems()
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= len(items) {
		m.selected = len(items) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}

	if m.selected < m.scrollOffset {
		m.scrollOffset = m.selected
	}
	if m.listRows > 0 && m.selected >= m.scrollOffset+m.listRows {
		m.scrollOffset = m.selected - m.listRows + 1
	}
}

func (m *model) updateLayout() {
	chromeHeight := 10 // header(3) + detailBar(1) + footer(6)
	if m.detailOpen {
		chromeHeight = 12 // header(3) + detailBar(1) + detailPanel(8)
	}
	if m.height < chromeHeight {
		m.listRows = 0
		return
	}
	m.listRows = m.height - chromeHeight
}

func (m *model) updateStats() {
	now := time.Now()

	active, incoming, outgoing, listening, err := fetchConnections()
	if err == nil {
		m.stats.incomingConns = incoming
		m.stats.outgoingConns = outgoing
		m.stats.listening = listening
		m.connections = active

		if m.mode != "tshark" && m.selected >= len(m.connections) {
			m.selected = len(m.connections) - 1
			if m.selected < 0 {
				m.selected = 0
			}
		}
	}

	counters, err := gsnet.IOCounters(true)
	if err == nil {
		var totalIn, totalOut, totalPacketsIn, totalPacketsOut uint64
		for _, c := range counters {
			if m.iface != "" && m.iface != "any" && c.Name != m.iface {
				continue
			}
			if strings.HasPrefix(c.Name, "lo") {
				continue
			}
			totalIn += c.BytesRecv
			totalOut += c.BytesSent
			totalPacketsIn += c.PacketsRecv
			totalPacketsOut += c.PacketsSent
		}

		m.stats.bytesIn = totalIn
		m.stats.bytesOut = totalOut
		m.stats.packetsIn = totalPacketsIn
		m.stats.packetsOut = totalPacketsOut
		m.stats.timestamp = now

		if !m.prevStats.timestamp.IsZero() {
			delta := now.Sub(m.prevStats.timestamp).Seconds()
			if delta > 0 {
				m.stats.bytesInPerSec = float64(m.stats.bytesIn-m.prevStats.bytesIn) / delta
				m.stats.bytesOutPerSec = float64(m.stats.bytesOut-m.prevStats.bytesOut) / delta
			}
		}

		m.prevStats = m.stats
		m.pushHistory(m.stats.bytesInPerSec, m.stats.bytesOutPerSec)
	}
}

func (m *model) pushHistory(in, out float64) {
	m.historyIn = append(m.historyIn, in)
	if len(m.historyIn) > historySize {
		m.historyIn = m.historyIn[1:]
	}
	m.historyOut = append(m.historyOut, out)
	if len(m.historyOut) > historySize {
		m.historyOut = m.historyOut[1:]
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && m.filtering {
		return m.handleFilterKey(key)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			if m.pcap != nil {
				_ = stopPcapCapture(m.pcap)
			}
			return m, tea.Quit
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
			m.adjustScroll()
		case "down", "j":
			items := m.currentItems()
			if m.selected < len(items)-1 {
				m.selected++
			}
			m.adjustScroll()
		case "pgup":
			m.selected -= m.listRows
			if m.selected < 0 {
				m.selected = 0
			}
			m.adjustScroll()
		case "pgdown":
			items := m.currentItems()
			m.selected += m.listRows
			if m.selected > len(items)-1 {
				m.selected = len(items) - 1
			}
			m.adjustScroll()
		case "home", "g":
			m.selected = 0
			m.adjustScroll()
		case "end", "G":
			items := m.currentItems()
			m.selected = len(items) - 1
			if m.selected < 0 {
				m.selected = 0
			}
			m.adjustScroll()
		case "c":
			if m.pktChan != nil {
				if m.mode == "tshark" {
					m.mode = "connections"
				} else {
					m.mode = "tshark"
				}
				m.adjustScroll()
			}
		case "/":
			m.filtering = true
		case "p":
			m.paused = !m.paused
		case "s":
			m.sortMode++
		case "d":
			m.detailOpen = !m.detailOpen
			m.updateLayout()
		case "e":
			if m.pcap == nil {
				state, err := startPcapCapture(m.iface, m.portFilter)
				if err != nil {
					m.err = err
				} else {
					m.pcap = state
					m.err = nil
				}
			} else {
				if err := stopPcapCapture(m.pcap); err != nil {
					m.err = err
				}
				m.pcap = nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()

	case tickMsg:
		m.updateStats()
		return m, tickCmd()

	case captureReadyMsg:
		if msg.err != nil {
			m.err = msg.err
			m.mode = "connections"
		} else {
			m.pktChan = msg.ch
			m.mode = "tshark"
			return m, packetCmd(m.pktChan)
		}

	case packetMsg:
		m.addPacket(msg.p)
		if m.mode == "tshark" {
			return m, packetCmd(m.pktChan)
		}

	case captureDoneMsg:
		m.mode = "connections"
		m.err = errors.New("packet capture stopped")
	}

	return m, nil
}

func (m model) handleFilterKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "enter":
		m.filtering = false
	case "backspace":
		if len(m.filterInput) > 0 {
			r := []rune(m.filterInput)
			m.filterInput = string(r[:len(r)-1])
			m.selected = 0
			m.scrollOffset = 0
			m.adjustScroll()
		}
	case "up", "down", "left", "right", "tab", "shift+tab":
		// Ignore navigation while typing.
	default:
		if key.Type == tea.KeyRunes {
			m.filterInput += string(key.Runes)
			m.selected = 0
			m.scrollOffset = 0
			m.adjustScroll()
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	if m.height < 6 {
		return "Terminal too small"
	}

	m.updateLayout()

	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(m.renderList())
	b.WriteString("\n")
	b.WriteString(m.renderDetailBar())
	b.WriteString("\n")
	if m.detailOpen {
		b.WriteString(m.renderDetailPanel())
	} else {
		b.WriteString(m.renderFooter())
	}

	return b.String()
}

func (m *model) renderHeader() string {
	modeText := strings.ToUpper(m.mode)
	if m.err != nil {
		modeText += " (tshark unavailable)"
	}
	if m.paused {
		modeText += " [PAUSED]"
	}
	if status := pcapStatus(m.pcap); status != "" {
		modeText += " " + status
	}

	left := titleStyle.Render(" netscope ")
	right := modeStyle.Render(" " + modeText + " ")
	top := lipgloss.JoinHorizontal(lipgloss.Left, left, " ", right)
	line1 := fillLine(top, m.width, headerBg)

	status := fmt.Sprintf(" packets %s  conns %d  listen %d  ▼ %s/s  ▲ %s/s",
		formatCount(uint64(len(m.packets))),
		m.stats.incomingConns+m.stats.outgoingConns,
		m.stats.listening,
		formatBytes(m.stats.bytesInPerSec),
		formatBytes(m.stats.bytesOutPerSec))
	line2 := fillLine(status, m.width, headerBg)
	sep := fillLine(strings.Repeat("─", m.width), m.width, headerBg)

	return line1 + "\n" + line2 + "\n" + sep
}

func (m *model) renderList() string {
	if m.listRows <= 0 {
		return ""
	}

	items := m.currentItems()
	m.adjustScroll()

	var b strings.Builder
	for i := 0; i < m.listRows; i++ {
		idx := m.scrollOffset + i
		if idx >= len(items) {
			b.WriteString(fillLine("", m.width, panelBg))
		} else {
			b.WriteString(m.renderItem(items[idx], idx == m.selected, m.width))
		}
		if i < m.listRows-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m *model) renderItem(item any, selected bool, width int) string {
	var s, proto string

	switch it := item.(type) {
	case Packet:
		proto = it.Protocol
		s = fmt.Sprintf("%6.3fs  %-5s  %-23s → %-23s  %4d B",
			float64(it.Time)/float64(time.Second),
			proto,
			it.Src,
			it.Dst,
			it.Length)
	case Connection:
		proto = it.Protocol
		name := it.Name
		if name == "" {
			name = "-"
		}
		status := it.Status
		if len(status) > 11 {
			status = status[:11]
		}
		s = fmt.Sprintf("%-11s %-5s  %-23s → %-23s  [%s]",
			status,
			proto,
			it.LocalAddr,
			it.RemoteAddr,
			name)
	}

	s = truncate(s, width)
	style := protocolStyle(proto)
	if selected {
		style = style.Copy().Reverse(true).Bold(true)
	}
	return style.Width(width).Render(s)
}

func (m *model) renderDetailBar() string {
	items := m.currentItems()
	if len(items) == 0 {
		return detailStyle.Width(m.width).Render(" no data ")
	}
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= len(items) {
		m.selected = len(items) - 1
	}

	var detail string
	switch it := items[m.selected].(type) {
	case Packet:
		detail = fmt.Sprintf(" %s  %s → %s  %d bytes ",
			it.Protocol, it.Src, it.Dst, it.Length)
	case Connection:
		name := it.Name
		if name == "" {
			name = "-"
		}
		detail = fmt.Sprintf(" %s  %s → %s  pid %d  %s ",
			it.Status, it.LocalAddr, it.RemoteAddr, it.PID, name)
	}

	detail = truncate(detail, m.width)
	return detailStyle.Width(m.width).Render(detail)
}

func (m *model) renderDetailPanel() string {
	items := m.currentItems()
	if len(items) == 0 {
		return detailStyle.Width(m.width).Height(8).Render(" no selection ")
	}

	var lines []string
	switch it := items[m.selected].(type) {
	case Packet:
		lines = []string{
			fmt.Sprintf("Protocol: %s", it.Protocol),
			fmt.Sprintf("Time:     %.6fs", float64(it.Time)/float64(time.Second)),
			fmt.Sprintf("Source:   %s", it.Src),
			fmt.Sprintf("Dest:     %s", it.Dst),
			fmt.Sprintf("Length:   %d bytes", it.Length),
		}
	case Connection:
		name := it.Name
		if name == "" {
			name = "-"
		}
		lines = []string{
			fmt.Sprintf("Status:   %s", it.Status),
			fmt.Sprintf("Protocol: %s", it.Protocol),
			fmt.Sprintf("Local:    %s", it.LocalAddr),
			fmt.Sprintf("Remote:   %s", it.RemoteAddr),
			fmt.Sprintf("PID:      %d", it.PID),
			fmt.Sprintf("Process:  %s", name),
		}
	}

	content := strings.Join(lines, "\n")
	return detailStyle.Width(m.width).Height(8).Render(content)
}

func (m *model) renderFooter() string {
	stats := fmt.Sprintf(" ▼ %s/s  ▲ %s/s  rx %s pkts  tx %s pkts",
		inStyle.Render(formatBytes(m.stats.bytesInPerSec)),
		outStyle.Render(formatBytes(m.stats.bytesOutPerSec)),
		formatCount(m.stats.packetsIn),
		formatCount(m.stats.packetsOut))
	line1 := fillLine(stats, m.width, footerBg)

	line2 := fillLine(labelStyle.Render(" download "), m.width, footerBg)
	sparkIn := sparklineInStyle.Width(m.width).Render(sparkline(m.historyIn, m.width))

	line4 := fillLine(labelStyle.Render(" upload "), m.width, footerBg)
	sparkOut := sparklineOutStyle.Width(m.width).Render(sparkline(m.historyOut, m.width))

	helpText := " q quit  ↑↓ scroll  c toggle  / filter  p pause  s sort  d detail  e pcap "
	if m.filtering {
		helpText = " filter: " + m.filterInput + "_ "
	}
	help := helpStyle.Render(helpText)
	line6 := fillLine(help, m.width, footerBg)

	return strings.Join([]string{line1, line2, sparkIn, line4, sparkOut, line6}, "\n")
}
