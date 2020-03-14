package mqtt

// Publish publishes a string message on a MQTT topic
// TODO: Handle Connection options - WILL, QOS
// TODO: Handle Publishing options - QoS
// TODO: Consider having a Connection that Publish is an operation on
//
func Publish(broker string, topic string, message string) {

	conn, err := Connect(broker, ClientName("H3LLTest"))
	if err != nil {
		panic(err)
	}

	publishRequest := NewPublishRequest(
		Message([]byte(message)),
		Topic(topic),
		QoS(0),
		Retain(false),
	)
	publishRequest.WriteTo(conn)

	// Done
	conn.Close()
}
