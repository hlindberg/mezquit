package mqtt

import (
	"bytes"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
)

// EncodeVariableInt Produces a []byte with the integer encoded as a MQTT variable int
func EncodeVariableInt(value int) []byte {
	var data bytes.Buffer

	for {
		encodedByte := byte(value % 128)
		value = value / 128
		// if there are more data to encode, set the top bit of this byte
		if value > 0 {
			encodedByte = (encodedByte | 128)
		}
		data.WriteByte(encodedByte)
		if !(value > 0) {
			break
		}
	}
	return data.Bytes()
}

// DecodeVariableInt Decodes a variable int value in the Reader stream, consumes it and returns the
// value.
func DecodeVariableInt(reader io.Reader) (int, error) {
	multiplier := 1
	value := 0
	for {
		buf := make([]byte, 1)
		reader.Read(buf)
		encodedByte := buf[0]
		value += int((encodedByte & 127)) * multiplier
		multiplier *= 128

		if multiplier > 128*128*128 {
			return 0, fmt.Errorf("Malformed variable length")
		}
		if (encodedByte & 128) == 0 {
			break
		}
	}
	return value, nil
}

// EncodeVariableIntTo encodes a given int into the given Buffer using MQTT variable int and return the written length
func EncodeVariableIntTo(value int, to *bytes.Buffer) int {
	bytes := EncodeVariableInt(value)
	to.Write(bytes)

	if log.IsLevelEnabled(log.DebugLevel) {
		var hexBytes string
		for _, b := range bytes {
			if len(hexBytes) != 0 {
				hexBytes += ", "
			}
			hexBytes += fmt.Sprintf("0x%x", b)
		}
		log.Debugf("Encoded Length %d into %d byte(s): [%s]", value, len(bytes), hexBytes)
	}
	return len(bytes)
}

// EncodeStringTo encodes a given string into the given buffer - 16 bit length + the content
func EncodeStringTo(value string, to *bytes.Buffer) {
	strLength := len(value)
	to.WriteByte(byte(strLength >> 8))
	to.WriteByte(byte(strLength & 0xFF))
	to.WriteString(value)
}

// EncodeBytesTo encodes a given []bytes] into the given buffer - 16 bit length + the content
func EncodeBytesTo(value []byte, to *bytes.Buffer) {
	bytesLength := len(value)
	to.WriteByte(byte(bytesLength >> 8))
	to.WriteByte(byte(bytesLength & 0xFF))
	to.Write(value)
}

// Encode16BitIntTo encodes a given int as 16 bits big endian value into the buffer
//
func Encode16BitIntTo(value int, to *bytes.Buffer) {
	to.WriteByte(byte(value >> 8))
	to.WriteByte(byte(value & 0xFF))
}
