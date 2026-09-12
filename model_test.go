package main

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestInitialModel(t *testing.T) {
	m := initialModel("eth0", "port 80")
	assert.Equal(t, "connections", m.mode)
	assert.Equal(t, "eth0", m.iface)
	assert.Equal(t, "port 80", m.portFilter)
	assert.Equal(t, 0, m.selected)
}

func TestAddPacket(t *testing.T) {
	m := initialModel("any", "")
	m.addPacket(Packet{Protocol: "TCP", Src: "a", Dst: "b", Length: 10})
	assert.Len(t, m.packets, 1)
	assert.Equal(t, 0, m.selected)

	for i := 0; i < maxPackets+10; i++ {
		m.addPacket(Packet{Protocol: "UDP", Length: i})
	}
	assert.Len(t, m.packets, maxPackets)
}

func TestPauseKeepsCapturing(t *testing.T) {
	m := initialModel("any", "")
	m.pktChan = make(chan Packet)
	m.mode = "tshark"

	// Pause the display.
	m.paused = true
	for i := 0; i < 5; i++ {
		m.addPacket(Packet{Protocol: "TCP", Length: i})
	}
	// While paused, selected should not auto-advance to the latest packet.
	assert.Equal(t, 0, m.selected)
	assert.Len(t, m.packets, 5)

	// Unpause and add another packet; now selection advances.
	m.paused = false
	m.addPacket(Packet{Protocol: "TCP", Length: 99})
	assert.Equal(t, 5, m.selected)
}

func TestFilteringPackets(t *testing.T) {
	m := initialModel("any", "")
	m.packets = []Packet{
		{Protocol: "TCP", Src: "10.0.0.1:80", Dst: "10.0.0.2:443", Length: 100},
		{Protocol: "UDP", Src: "10.0.0.1:53", Dst: "8.8.8.8:53", Length: 50},
		{Protocol: "DNS", Src: "10.0.0.1", Dst: "8.8.8.8", Length: 40},
	}
	m.mode = "tshark"

	m.filterInput = "dns"
	items := m.currentItems()
	assert.Len(t, items, 1)
	assert.Equal(t, "DNS", items[0].(Packet).Protocol)

	m.filterInput = "53"
	items = m.currentItems()
	assert.Len(t, items, 1)
}

func TestFilteringConnections(t *testing.T) {
	m := initialModel("any", "")
	m.connections = []Connection{
		{Status: "ESTABLISHED", Protocol: "TCP", LocalAddr: "10.0.0.1:80", RemoteAddr: "1.1.1.1:443", Name: "curl"},
		{Status: "TIME_WAIT", Protocol: "TCP", LocalAddr: "10.0.0.1:22", RemoteAddr: "2.2.2.2:22", Name: "sshd"},
	}
	m.mode = "connections"

	m.filterInput = "curl"
	items := m.currentItems()
	assert.Len(t, items, 1)
	assert.Equal(t, "curl", items[0].(Connection).Name)

	m.filterInput = "22"
	items = m.currentItems()
	assert.Len(t, items, 1)
	assert.Equal(t, "sshd", items[0].(Connection).Name)
}

func TestSortPackets(t *testing.T) {
	m := initialModel("any", "")
	m.packets = []Packet{
		{Protocol: "UDP", Length: 10},
		{Protocol: "TCP", Length: 100},
		{Protocol: "TCP", Length: 50},
	}

	m.sortMode = sortPacketProtocol
	sorted := m.sortedPackets(m.packets)
	assert.Equal(t, "TCP", sorted[0].Protocol)
	assert.Equal(t, "TCP", sorted[1].Protocol)
	assert.Equal(t, "UDP", sorted[2].Protocol)

	m.sortMode = sortPacketLength
	sorted = m.sortedPackets(m.packets)
	assert.Equal(t, 100, sorted[0].Length)
	assert.Equal(t, 50, sorted[1].Length)
	assert.Equal(t, 10, sorted[2].Length)
}

func TestSortConnections(t *testing.T) {
	m := initialModel("any", "")
	m.connections = []Connection{
		{Status: "ESTABLISHED", Protocol: "TCP", LocalAddr: "z", RemoteAddr: "z", Name: "zebra"},
		{Status: "TIME_WAIT", Protocol: "UDP", LocalAddr: "a", RemoteAddr: "a", Name: "apple"},
	}

	m.sortMode = sortConnProtocol
	sorted := m.sortedConnections(m.connections)
	assert.Equal(t, "TCP", sorted[0].Protocol)
	assert.Equal(t, "UDP", sorted[1].Protocol)

	m.sortMode = sortConnName
	sorted = m.sortedConnections(m.connections)
	assert.Equal(t, "apple", sorted[0].Name)
	assert.Equal(t, "zebra", sorted[1].Name)
}

func TestUpdateFilterMode(t *testing.T) {
	m := initialModel("any", "")
	m.mode = "tshark"
	m.pktChan = make(chan Packet)
	m.packets = []Packet{
		{Protocol: "TCP", Src: "a", Dst: "b"},
		{Protocol: "UDP", Src: "c", Dst: "d"},
	}

	// Enter filter mode.
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	mm := newM.(model)
	assert.True(t, mm.filtering)

	// Type a filter.
	newM, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	mm = newM.(model)
	newM, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm = newM.(model)
	assert.Equal(t, "ud", mm.filterInput)

	// Exit filter mode.
	newM, _ = mm.Update(tea.KeyMsg{Type: tea.KeyEscape})
	mm = newM.(model)
	assert.False(t, mm.filtering)
}

func TestUpdatePauseKey(t *testing.T) {
	m := initialModel("any", "")
	m.mode = "tshark"
	m.pktChan = make(chan Packet)

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	mm := newM.(model)
	assert.True(t, mm.paused)

	newM, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	mm = newM.(model)
	assert.False(t, mm.paused)
}

func TestUpdateSortKey(t *testing.T) {
	m := initialModel("any", "")
	m.mode = "tshark"
	m.pktChan = make(chan Packet)

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	mm := newM.(model)
	assert.Equal(t, 1, mm.sortMode)
}

func TestUpdateDetailKey(t *testing.T) {
	m := initialModel("any", "")
	m.mode = "tshark"
	m.pktChan = make(chan Packet)
	m.width = 80
	m.height = 24

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm := newM.(model)
	assert.True(t, mm.detailOpen)
	// detail panel is 8 rows, replacing the 6-row footer.
	assert.Equal(t, 8, mm.height-(mm.listRows+3+1))

	newM, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm = newM.(model)
	assert.False(t, mm.detailOpen)
}

func TestUpdateToggleMode(t *testing.T) {
	m := initialModel("any", "")
	m.pktChan = make(chan Packet)
	m.mode = "tshark"

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	mm := newM.(model)
	assert.Equal(t, "connections", mm.mode)

	newM, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	mm = newM.(model)
	assert.Equal(t, "tshark", mm.mode)
}

func TestUpdateScrollKeys(t *testing.T) {
	m := initialModel("any", "")
	m.mode = "tshark"
	m.pktChan = make(chan Packet)
	for i := 0; i < 10; i++ {
		m.addPacket(Packet{Protocol: "TCP", Length: i})
	}
	m.width = 80
	m.height = 24
	m.updateLayout()

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	mm := newM.(model)
	assert.Equal(t, 0, mm.selected)

	newM, _ = mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	mm = newM.(model)
	assert.Equal(t, 9, mm.selected)
}

func TestViewRenders(t *testing.T) {
	m := initialModel("any", "")
	m.width = 80
	m.height = 24
	m.packets = []Packet{{Protocol: "TCP", Src: "a", Dst: "b", Length: 10}}
	m.mode = "tshark"
	m.updateLayout()

	view := m.View()
	assert.Contains(t, view, "netscope")
	assert.Contains(t, view, "TCP")
}

func TestViewWithDetailPanel(t *testing.T) {
	m := initialModel("any", "")
	m.width = 80
	m.height = 24
	m.packets = []Packet{{Protocol: "TCP", Src: "10.0.0.1:80", Dst: "10.0.0.2:443", Length: 100}}
	m.mode = "tshark"
	m.detailOpen = true
	m.updateLayout()

	view := m.View()
	assert.Contains(t, view, "Protocol:")
	assert.Contains(t, view, "Source:")
}

func TestRenderItem(t *testing.T) {
	m := initialModel("any", "")
	p := Packet{Protocol: "TCP", Src: "10.0.0.1:80", Dst: "10.0.0.2:443", Length: 100, Time: time.Second}
	out := m.renderItem(p, false, 80)
	assert.Contains(t, out, "TCP")
	assert.Contains(t, out, "10.0.0.1:80")
}

func TestTickUpdatesStats(t *testing.T) {
	m := initialModel("any", "")
	m.width = 80
	m.height = 24

	newM, cmd := m.Update(tickMsg(time.Now()))
	mm := newM.(model)
	assert.NotNil(t, cmd)
	// Stats update is environment-dependent, just ensure no panic.
	_ = mm.stats.timestamp
}
