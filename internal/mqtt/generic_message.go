package mqtt

import (
	"bytes"
	"io"
)

// GenericMessage is a generic MQTT message struct with a header byte and all the bytes for the message in a `body`
type GenericMessage struct {
	fixedHeader byte
	body        []byte
}

// WriteTo implements io.WriterTo for GenericMessage
func (m *GenericMessage) WriteTo(writer io.Writer) (int64, error) {
	var data bytes.Buffer // 64 bytes in the first Grow which should be enough unless client ID is very long (not worth optimizing)
	bodyLength := len(m.body)
	data.WriteByte(m.fixedHeader)
	lengthBytes := EncodeVariableInt(bodyLength)
	// EncodeVariableIntTo(bodyLength, &data)
	data.Write(lengthBytes)
	if bodyLength > 0 {
		data.Write(m.body)
	}
	n, err := data.WriteTo(writer)
	return int64(n), err
}

// WriteDupTo sets the DUP bit for applicable messages and then writes to the given writer
// The original message is unchanged
func (m *GenericMessage) WriteDupTo(writer io.Writer) (int64, error) {
	m2 := m
	if m.fixedHeader<<4 == PublishType {
		m2 = &GenericMessage{fixedHeader: m.fixedHeader | DupBit, body: m.body}
	}
	return m2.WriteTo(writer)
}
