package ymodem

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestYModem_CRC16(t *testing.T) {
	data := []byte{72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100}
	require.Equal(t, uint16(39210), CRC16(data))
}

func TestYModem_CRC16Constant(t *testing.T) {
	data := []byte{72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100}
	require.Equal(t, uint16(43803), CRC16Constant(data, 13))
}
