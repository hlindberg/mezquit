package mqtt

import "io"

// MessageWriter can write a MQTT Message, or a Duplicate of a message
type MessageWriter interface {
	io.WriterTo
	WriteDupTo(writer io.Writer) (int64, error)
}
