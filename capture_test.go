package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseTsharkLine(t *testing.T) {
	line := "1.2345\t192.168.1.1\t192.168.1.2\t12345\t80\t\t\t150\tTCP"
	p, ok := parseTsharkLine(line)
	assert.True(t, ok)
	assert.Equal(t, time.Duration(1.2345*float64(time.Second)), p.Time)
	assert.Equal(t, "TCP", p.Protocol)
	assert.Equal(t, "192.168.1.1:12345", p.Src)
	assert.Equal(t, "192.168.1.2:80", p.Dst)
	assert.Equal(t, 150, p.Length)
}

func TestParseTsharkLineUDP(t *testing.T) {
	line := "0.5\t10.0.0.1\t10.0.0.2\t\t\t53\t53\t100\tDNS"
	p, ok := parseTsharkLine(line)
	assert.True(t, ok)
	assert.Equal(t, "10.0.0.1:53", p.Src)
	assert.Equal(t, "10.0.0.2:53", p.Dst)
	assert.Equal(t, "DNS", p.Protocol)
}

func TestParseTsharkLineInvalid(t *testing.T) {
	_, ok := parseTsharkLine("not\tenough\tfields")
	assert.False(t, ok)

	_, ok = parseTsharkLine("\t\t\t\t\t\t\t\t")
	assert.False(t, ok)
}

func TestFormatAddr(t *testing.T) {
	assert.Equal(t, "?", formatAddr("", "", ""))
	assert.Equal(t, "192.168.1.1", formatAddr("192.168.1.1", "", ""))
	assert.Equal(t, "192.168.1.1:443", formatAddr("192.168.1.1", "443", ""))
	assert.Equal(t, "192.168.1.1:53", formatAddr("192.168.1.1", "", "53"))
	assert.Equal(t, "[::1]:443", formatAddr("::1", "443", ""))
}

func TestStartTsharkMissingBinary(t *testing.T) {
	// Use a bogus binary name by temporarily relying on the system path.
	// This test simply exercises the error path when tshark is absent.
	_, err := startTshark("any", "")
	if err == nil {
		t.Skip("tshark is installed; skipping absence test")
	}
	assert.Error(t, err)
}
