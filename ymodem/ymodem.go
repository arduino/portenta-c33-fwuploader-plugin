// portenta-c33-fwuploader-plugin
// Copyright (c) 2023 Arduino LLC.  All right reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

//go:build !darwin

package ymodem

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"time"
)

// ymodem constants
const (
	SOH  byte = 0x01
	STX  byte = 0x02
	EOT  byte = 0x04
	ACK  byte = 0x06
	NAK  byte = 0x15
	POLL byte = 0x43

	ShortPacketPayloadLen = 128
	LongPacketPayloadLen  = 1024
)

// default errors
var (
	ErrInvalidPacket   = errors.New("invalid packet")
	ErrSendingEndBlock = errors.New("failed to send end block")
)

// CRC16 calculate crc16
func CRC16(data []byte) uint16 {
	var u16CRC uint16 = 0

	for _, character := range data {
		part := uint16(character)

		u16CRC = u16CRC ^ (part << 8)
		for i := 0; i < 8; i++ {
			if u16CRC&0x8000 > 0 {
				u16CRC = u16CRC<<1 ^ 0x1021
			} else {
				u16CRC = u16CRC << 1
			}
		}
	}

	return u16CRC
}

// CRC16Constant calculate constant crc16
func CRC16Constant(data []byte, length int) uint16 {
	var u16CRC uint16 = 0

	for _, character := range data {
		part := uint16(character)

		u16CRC = u16CRC ^ (part << 8)
		for i := 0; i < 8; i++ {
			if u16CRC&0x8000 > 0 {
				u16CRC = u16CRC<<1 ^ 0x1021
			} else {
				u16CRC = u16CRC << 1
			}
		}
	}

	for c := 0; c < length-len(data); c++ {
		u16CRC = u16CRC ^ (0x04 << 8)
		for i := 0; i < 8; i++ {
			if u16CRC&0x8000 > 0 {
				u16CRC = u16CRC<<1 ^ 0x1021
			} else {
				u16CRC = u16CRC << 1
			}
		}
	}

	return u16CRC
}

func sendBlock(c io.ReadWriter, block uint16, data []byte) error {
	//send STX
	if _, err := c.Write([]byte{STX}); err != nil {
		return err
	}
	if _, err := c.Write([]byte{uint8(block)}); err != nil {
		return err
	}
	if _, err := c.Write([]byte{255 - uint8(block)}); err != nil {
		return err
	}

	//send data
	var toSend bytes.Buffer
	toSend.Write(data)
	for toSend.Len() < LongPacketPayloadLen {
		toSend.Write([]byte{EOT})
	}

	//calc CRC
	u16CRC := CRC16Constant(data, LongPacketPayloadLen)
	toSend.Write([]byte{uint8(u16CRC >> 8)})
	toSend.Write([]byte{uint8(u16CRC & 0x0FF)})

	sent := 0
	for sent < toSend.Len() {
		n, err := c.Write(toSend.Bytes()[sent:])
		if err != nil {
			return err
		}
		sent += n
	}

	return nil
}

// ModemSend sends given file data to the stream.
func ModemSend(c io.ReadWriter, data []byte, filename string) error {
	oBuffer := make([]byte, 1)

	// Wait for Poll
	if _, err := c.Read(oBuffer); err != nil {
		return err
	}

	// Send zero block with filename and size
	if oBuffer[0] == POLL {
		var send bytes.Buffer
		send.WriteString(filepath.Base(filename))
		send.WriteByte(0x0)
		send.WriteString(fmt.Sprintf("%d", len(data)))
		for send.Len() < LongPacketPayloadLen {
			send.Write([]byte{0x0})
		}

		sendBlock(c, 0, send.Bytes())

		// Wait for ACK
		if _, err := c.Read(oBuffer); err != nil {
			return err
		}

		if oBuffer[0] != ACK {
			return fmt.Errorf("failed to send header block: %x", oBuffer[0])
		}
	}

	// Wait for Poll
	if _, err := c.Read(oBuffer); err != nil {
		return err
	}

	// Send remaining data
	if oBuffer[0] == POLL {
		blocks := uint16(len(data) / LongPacketPayloadLen)
		if len(data) > int(int(blocks)*int(LongPacketPayloadLen)) {
			blocks++
		}

		failed := 0
		var currentBlock uint16 = 0
		for currentBlock < blocks && failed < 10 {
			if int(int(currentBlock+1)*int(LongPacketPayloadLen)) > len(data) {
				sendBlock(c, currentBlock+1, data[int(currentBlock)*int(LongPacketPayloadLen):])
			} else {
				sendBlock(c, currentBlock+1, data[int(currentBlock)*int(LongPacketPayloadLen):(int(currentBlock)+1)*int(LongPacketPayloadLen)])
			}

			if runtime.GOOS == "windows" {
				time.Sleep(50 * time.Millisecond)
			}
			if _, err := c.Read(oBuffer); err != nil {
				return err
			}

			if oBuffer[0] == ACK {
				currentBlock++
			} else {
				failed++
			}
		}
	}

	// Wait for NAK and send EOT
	if _, err := c.Write([]byte{EOT}); err != nil {
		return err
	}

	if _, err := c.Read(oBuffer); err != nil {
		return err
	}

	if oBuffer[0] != NAK {
		return errors.New("didn't get a nak when expected")
	}

	// Send EOT again
	if _, err := c.Write([]byte{EOT}); err != nil {
		return err
	}

	if _, err := c.Read(oBuffer); err != nil {
		return err
	}

	if oBuffer[0] != ACK {
		return ErrSendingEndBlock
	}

	// Wait for POLL
	if _, err := c.Read(oBuffer); err != nil {
		return err
	}

	if oBuffer[0] != POLL {
		return ErrSendingEndBlock
	}

	// Send empty block to signify end
	var zero bytes.Buffer
	for zero.Len() < LongPacketPayloadLen {
		zero.Write([]byte{0x0})
	}

	sendBlock(c, 0, zero.Bytes())

	// Wait for ACK
	if _, err := c.Read(oBuffer); err != nil {
		return err
	}

	if oBuffer[0] != ACK {
		return ErrSendingEndBlock
	}

	return nil
}

func receivePacket(c io.ReadWriter) ([]byte, error) {
	oBuffer := make([]byte, 1)
	//dBuffer := make([]byte, LONG_PACKET_PAYLOAD_LEN)

	if _, err := c.Read(oBuffer); err != nil {
		return nil, err
	}
	pType := oBuffer[0]

	if pType == EOT {
		return nil, nil
	}

	var packetSize int
	switch pType {
	case SOH:
		packetSize = ShortPacketPayloadLen
	case STX:
		packetSize = LongPacketPayloadLen
	}

	if _, err := c.Read(oBuffer); err != nil {
		return nil, err
	}
	packetCount := oBuffer[0]

	if _, err := c.Read(oBuffer); err != nil {
		return nil, err
	}
	inverseCount := oBuffer[0]

	if inverseCount+packetCount != 255 {
		if _, err := c.Write([]byte{NAK}); err != nil {
			return nil, err
		}
		return nil, ErrInvalidPacket
	}

	received := 0
	var pData bytes.Buffer

	for received < packetSize {
		tempBuffer := make([]byte, packetSize-received)

		n, err := c.Read(tempBuffer)
		if err != nil {
			return nil, err
		}

		received += n
		pData.Write(tempBuffer[:n])
	}

	var crc uint16
	if _, err := c.Read(oBuffer); err != nil {
		return nil, err
	}
	crc = uint16(oBuffer[0])

	if _, err := c.Read(oBuffer); err != nil {
		return nil, err
	}
	crc <<= 8
	crc |= uint16(oBuffer[0])

	crcCalc := CRC16(pData.Bytes())
	if crcCalc != crc {
		if _, err := c.Write([]byte{NAK}); err != nil {
			return nil, err
		}
	}

	if _, err := c.Write([]byte{ACK}); err != nil {
		return nil, err
	}

	return pData.Bytes(), nil
}

// ModemReceive nodoc
func ModemReceive(c io.ReadWriter) (string, []byte, error) {
	var data bytes.Buffer

	// Start Connection
	if _, err := c.Write([]byte{POLL}); err != nil {
		return "", nil, err
	}

	// Read file information
	pktData, err := receivePacket(c)
	if err != nil {
		return "", nil, err
	}

	filenameEnd := bytes.IndexByte(pktData, 0x0)
	filename := string(pktData[0:filenameEnd])

	var filesize int
	fmt.Sscanf(string(pktData[filenameEnd+1:]), "%d", &filesize)

	if _, err := c.Write([]byte{POLL}); err != nil {
		return "", nil, err
	}

	// Read Packets
	for {
		pktData, err := receivePacket(c)
		if err == ErrInvalidPacket {
			continue
		}

		if err != nil {
			return "", nil, err
		}

		// End of Transmission
		if pktData == nil {
			break
		}

		data.Write(pktData)
	}

	// Send NAK to respond to EOT
	if _, err := c.Write([]byte{NAK}); err != nil {
		return "", nil, err
	}

	oBuffer := make([]byte, 1)
	if _, err := c.Read(oBuffer); err != nil {
		return "", nil, err
	}

	// Send ACK to respond to second EOT
	if oBuffer[0] != EOT {
		return "", nil, err
	}

	if _, err := c.Write([]byte{ACK}); err != nil {
		return "", nil, err
	}

	// Second POLL to get remaining file or close
	if _, err := c.Write([]byte{POLL}); err != nil {
		return "", nil, err
	}

	// Get remaining data ( for now assume one file )
	if _, err := receivePacket(c); err != nil {
		return "", nil, err
	}

	return filename, data.Bytes()[0:filesize], nil
}
