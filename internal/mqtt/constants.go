package mqtt

const (
	// Reserved is all zero bits
	Reserved = 0

	// CONTROL MESSAGE TYPES
	// ---------------------

	// ConnectType control message type
	ConnectType = 1

	// ConnAckType control message type
	ConnAckType = 2

	// PublishType control message type
	PublishType = 3

	// CONNECTION PORTS
	// ----------------

	// UnencryptedPortTCP is the standard MQTT port over TCP for unencrypted content
	UnencryptedPortTCP = "1883"

	// Connect bits

	// UserNameFlag is a bit that signals that UserName is in the payload
	UserNameFlag = 1 << 7

	// PasswordFlag is a bit that signals that Password is in the payload
	PasswordFlag = 1 << 6

	// WillRetainFlag is a bit that signals that Will Retention is in the payload
	WillRetainFlag = 1 << 5

	// WillQoSZero sets the Will QoS to 0 (since this is 0 it isn't really needed)
	WillQoSZero = 0

	// WillQoSOne sets the Will QoS to 1 (two bits (3, 4) are set)
	WillQoSOne = 1 << 3

	// WillQoSTwo sets the Will QoS to 1 (two bits (3, 4) are set)
	WillQoSTwo = 2 << 3

	// WillFlag is a bit that signals that Will is in the payload
	WillFlag = 1 << 2

	// CleanSessionFlag is a bit that signals that a clean session is wanted
	CleanSessionFlag = 1 << 1

	// Connack results

	// ConnectionAccepted means it is ok to use connection
	ConnectionAccepted = 0

	// ConnectionRefusedRejectedVersion Protocol version is not accepted
	ConnectionRefusedRejectedVersion = 1

	// ConnectionRefusedRejectedIdentifier Client Identifier is not accepted
	ConnectionRefusedRejectedIdentifier = 2

	// ConnectionRefusedServerUnavailable server is not available
	ConnectionRefusedServerUnavailable = 3

	// ConnectionRefusedBadUserPassword User name or Password is bad
	ConnectionRefusedBadUserPassword = 4

	// ConnectionRefusedNotAuthorized the presented credentials resulted in not being authorized
	ConnectionRefusedNotAuthorized = 5

	// Publish Bits
	// ------

	// QoSZero sets the QoS to 0 (since this is 0 it isn't really needed)
	QoSZero = 0

	// QoSOne sets the QoS to 1 (two bits (3, 4) are set)
	QoSOne = 1 << 1

	// QoSTwo sets the QoS to 1 (two bits (3, 4) are set)
	QoSTwo = 2 << 1

	// NoDupBit sets the DUP bit to 0 (since it is 0 it isn't really needed)
	NoDupBit = 0

	// DupBit sets the DUP bit to 1 (since it is 0 it isn't really needed)
	DupBit = 1 << 3

	// NoRetainBit sets the RETAIN bit to 0 (since it is 0 it isn't really needed)
	NoRetainBit = 0

	// RetainBit sets the RETAIN bit to 1 (since it is 0 it isn't really needed)
	RetainBit = 1
)
