package mqtt

import (
	"bytes"
	"fmt"
	"net"
)

// Connect connects to a MQTT broker
// TODO: Add WILL, QOS
func Connect(broker string, clientID string) (net.Conn, error) {

	// hardcoded for now
	// qos := 0
	// will := ""

	// Dial the broker
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", broker, UnencryptedPortTCP))
	if err != nil {
		return conn, err
	}

	// Do a Connect (wait for CONNACK)
	// Need a fixed and variable header
	var data bytes.Buffer

	clientIDLength := len(clientID)

	// FIXED CONNECT HEADER
	data.WriteByte(ConnectType<<4 | Reserved)

	// REMAINING LENGTH - VARIABLE HEADER LENGTH + PAYLOAD LENGTH
	EncodeVariableIntTo(10+2+clientIDLength, &data)

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
	data.WriteByte(byte(clientIDLength >> 8))   // (11)  Client ID Length MSB
	data.WriteByte(byte(clientIDLength & 0xff)) // (12)  Client ID Length LSB
	data.WriteString(clientID)

	// There is no optional payload since the above does not contain user, password, will topics, or anything else
	// that requires a payload. If those were to be set, they should be appended and the total length of
	// variable header and payload needs to be updated (patched).
	//

	// Do the Connect
	// --------------

	written, err := data.WriteTo(conn)
	if err != nil {
		return conn, err
	}
	fmt.Printf("Wrote %d bytes\n", written)

	// Wait for response - and output whatever is received
	//
	response := make([]byte, 4)
	_, err = conn.Read(response)

	if err != nil {
		// TODO: Log that read of response failed
		return conn, err
	}

	// TODO: Log this: fmt.Printf("Read %d bytes\n", read)

	if response[0] != ConnAckType<<4 {
		// TODO: Change to error return
		panic(fmt.Sprintf("Did not get a CONNACK back from Connect - got %d", response[0]))
	}
	if response[1] != 2 {
		// TODO: Change to error return
		panic(fmt.Sprintf("Expected ConnAck length of 2 but got %d", response[1]))
	}

	if response[2] == 1 {
		// TODO: Change to log this (debug level)
		fmt.Printf("Got SP CONNACK flag set in response\n")
	}

	if response[3] != ConnectionAccepted {
		// TODO: Change to error return
		panic(fmt.Sprintf("Did not get ConnectionAccepted return status back - got %d", response[3]))
	}

	// TODO: Change this to log debug
	fmt.Printf("Got Connection - all is well\n")

	return conn, nil
}
