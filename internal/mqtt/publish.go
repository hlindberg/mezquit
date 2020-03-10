package mqtt

import (
	"bytes"
	"fmt"
	"net"
)

// Publish publishes a string message on a MQTT topic
// TODO: Add WILL, QOS
func Publish(broker string, topic string, message string) {

	// hardcoded for now
	// qos := 0
	// will := ""

	// Dial the broker
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", broker, UnencryptedPortTCP))
	if err != nil {
		panic(err)
	}

	// Do a Connect (wait for CONNACK)
	// Need a fixed and variable header
	var data bytes.Buffer
	// FIXED CONNECT HEADER
	data.WriteByte(ConnectType<<4 | Reserved)

	// REMAINING LENGTH - VARIABLE HEADER LENGTH + PAYLOAD LENGTH

	// data.WriteByte(0)          // Remaining Length MSB
	data.WriteByte(10 + 2 + 8) // Remaining Lenght LSB - Variable part 10 bytes, payload 2+8 (length 0,8;)

	// Connect variable part            Byte   Description
	//                                  ------ ----------------------------------------------
	data.WriteByte(0)                // (1)    Protocol Name Length MSB
	data.WriteByte(4)                // (2)    Protocol Name Length LSB
	data.WriteString("MQTT")         // (3-6)  Protocol Name
	data.WriteByte(4)                // (7)    Protocol Level - MQTT 3.1.1 is 4, MQTT 5 is 5
	data.WriteByte(CleanSessionFlag) // (8)    Connect Bits
	data.WriteByte(0)                // (9)    Keep Alive Seconds MSB
	data.WriteByte(10)               // (9-10) Keep Alive Seconds LSB

	// PAYLOAD
	// A Client ID is required as the first element of the payload. It can be of length 0 to make the server assign
	// the id.
	data.WriteByte(0) // (11)  Client ID Length MSB
	data.WriteByte(8) // (12)  Client ID Length LSB
	data.WriteString("H3LLTest")

	// There is no optional payload since the above does not contain user, password, will topics, or anything else
	// that requires a payload. If those were to be set, they should be appended and the total length of
	// variable header and payload needs to be updated (patched).
	//

	// Do the Connect
	// --------------

	written, err := data.WriteTo(conn)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Wrote %d bytes\n", written)

	// Wait for response - and output whatever is received
	//
	response := make([]byte, 4)
	read, err := conn.Read(response)

	if err != nil {
		panic(err)
	}
	fmt.Printf("Read %d bytes\n", read)

	if response[0] != ConnAckType<<4 {
		panic(fmt.Sprintf("Did not get a CONNACK back from Connect - got %d", response[0]))
	}
	if response[1] != 2 {
		panic(fmt.Sprintf("Expected ConnAck length of 2 but got %d", response[1]))
	}

	if response[2] == 1 {
		fmt.Printf("Gotten SP CONNACK flag set in response\n")
	}

	if response[3] != ConnectionAccepted {
		panic(fmt.Sprintf("Did not get ConnectionAccepted return status back - got %d", response[3]))
	}

	fmt.Printf("Got Connection - all is well\n")

	// Do a Publish
	// ------------
	data.Reset()
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
