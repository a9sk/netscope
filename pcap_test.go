package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPcapStatusNil(t *testing.T) {
	assert.Equal(t, "", pcapStatus(nil))
}

func TestPickInterface(t *testing.T) {
	name := pickInterface()
	assert.NotEmpty(t, name)
}

func TestStopPcapCaptureNil(t *testing.T) {
	assert.NoError(t, stopPcapCapture(nil))
}
