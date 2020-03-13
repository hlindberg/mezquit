package mqtt

import (
	"fmt"
	"net"
)

// Connect connects to a MQTT broker
// TODO: Add WILL, QOS
func Connect(broker string, options ...ConnectOption) (net.Conn, error) {

	// hardcoded for now
	// qos := 0
	// will := ""

	// Dial the broker
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", broker, UnencryptedPortTCP))
	if err != nil {
		return conn, err
	}

	connectionRequest := NewConnectRequest(options...)

	// Do the Connect
	// --------------

	written, err := connectionRequest.WriteTo(conn)
	// written, err := data.WriteTo(conn)
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
