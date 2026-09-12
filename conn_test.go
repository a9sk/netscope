package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatConnAddr(t *testing.T) {
	assert.Equal(t, "192.168.1.1:80", formatConnAddr("192.168.1.1", 80))
	assert.Equal(t, "[::1]:443", formatConnAddr("::1", 443))
}

func TestProcessNameZero(t *testing.T) {
	assert.Equal(t, "", processName(0))
}

func TestFetchConnections(t *testing.T) {
	// This is environment-dependent; we just check it returns without panic
	// and that counts are non-negative.
	conns, incoming, outgoing, listening, err := fetchConnections()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, incoming, 0)
	assert.GreaterOrEqual(t, outgoing, 0)
	assert.GreaterOrEqual(t, listening, 0)
	assert.GreaterOrEqual(t, len(conns), 0)
}
