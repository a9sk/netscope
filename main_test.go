package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePortFilter(t *testing.T) {
	assert.Equal(t, "", normalizePortFilter(""))
	assert.Equal(t, "port 80", normalizePortFilter("80"))
	assert.Equal(t, "port 443", normalizePortFilter("443"))
	assert.Equal(t, "tcp port 80", normalizePortFilter("tcp port 80"))
	assert.Equal(t, "udp port 53", normalizePortFilter("udp port 53"))
}
