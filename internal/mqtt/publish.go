package mqtt

import (
	"bytes"
	"fmt"
)

// Publish publishes a string message on a MQTT topic
// TODO: Add WILL, QOS
func Publish(broker string, topic string, message string) {

	conn, err := Connect(broker, "H3LLTest")
	if err != nil {
		panic(err)
	}

	// Do a Publish
	// ------------
	var data bytes.Buffer

	data.WriteByte(PublishType<<4 | NoDup | QoSZero | NoRetain)

	topicLength := len(topic)
	if (topicLength &^ 0xffff) > 0 {
		panic(fmt.Sprintf("Topic Length > max 0xffff, got %d", topicLength))
	}
	msgLength := len(message)
	if (msgLength &^ 0xffff) > 0 {
		panic(fmt.Sprintf("Message Length > max 0xffff, got %d", msgLength))
	}

	// Remaining Length is variable - TODO: This is for QoS 0 only as other will need to include 2 bytes for Packet ID
	EncodeVariableIntTo(topicLength+2+msgLength, &data)

	// TopicName - a string (2 bytes length + bytes)
	data.WriteByte(byte((topicLength >> 8)) & 0xFF) // MSB
	data.WriteByte(byte(topicLength & 0xFF))        // LSB
	data.WriteString(topic)

	// (Only if QoS > 1) PacketIdentifier
	// TODO: Does nothing now

	// Message - just bytes, no length
	data.WriteString(message)

	bytes := data.Bytes()
	msgLength = len(bytes)
	n, err := conn.Write(bytes)
	if err != nil {
		panic(err)
	}
	if n != msgLength {
		panic(fmt.Sprintf("Expected to write %d bytes, but wrote %d", msgLength, n))
	}

	// There is no Ack for a QoS 0 Publish

	// Done
	conn.Close()
}
