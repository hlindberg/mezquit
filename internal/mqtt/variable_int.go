package mqtt

import (
	"bytes"
	"fmt"
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

// EncodeVariableIntTo encodes a given int into the given Buffer using MQTT variable int and return the written length
func EncodeVariableIntTo(value int, to *bytes.Buffer) int {
	bytes := EncodeVariableInt(value)
	to.Write(bytes)
	fmt.Printf("Encoded Length %d as (%d) bytes: ", value, len(bytes))
	for _, b := range bytes {
		fmt.Printf("%d", b)
	}
	fmt.Printf("\n")
	return len(bytes)
}
