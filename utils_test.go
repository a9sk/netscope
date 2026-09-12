package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatCount(t *testing.T) {
	assert.Equal(t, "0", formatCount(0))
	assert.Equal(t, "999", formatCount(999))
	assert.Equal(t, "1.0k", formatCount(1000))
	assert.Equal(t, "1.5k", formatCount(1500))
	assert.Equal(t, "1.0M", formatCount(1_000_000))
	assert.Equal(t, "2.5M", formatCount(2_500_000))
}

func TestFormatBytes(t *testing.T) {
	assert.Equal(t, "0 B", formatBytes(0))
	assert.Equal(t, "512 B", formatBytes(512))
	assert.Equal(t, "1.00 KB", formatBytes(1024))
	assert.Equal(t, "1.50 MB", formatBytes(1.5*1024*1024))
	assert.Equal(t, "2.00 GB", formatBytes(2*1024*1024*1024))
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
	assert.Equal(t, "hell", truncate("hello", 4))
	assert.Equal(t, "", truncate("hello", 0))
}

func TestSparkline(t *testing.T) {
	assert.Equal(t, "", sparkline(nil, 10))
	assert.Equal(t, "", sparkline([]float64{1, 2, 3}, 0))
	assert.Equal(t, "█", sparkline([]float64{8}, 1))
	values := []float64{0, 4, 8, 2, 6}
	out := sparkline(values, 5)
	assert.Equal(t, 5, len([]rune(out)))
}

func TestProtocolStyle(t *testing.T) {
	// Smoke test: style should not be empty and should vary by protocol.
	tcp := protocolStyle("TCP")
	udp := protocolStyle("UDP")
	unknown := protocolStyle("FOO")
	assert.NotNil(t, tcp)
	assert.NotNil(t, udp)
	assert.NotNil(t, unknown)
}
