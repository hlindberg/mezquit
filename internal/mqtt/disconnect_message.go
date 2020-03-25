package mqtt

// NewDisconnectMessage returns a new message of this kind
func NewDisconnectMessage() *GenericMessage {
	return &GenericMessage{fixedHeader: (DisconnectType << 4), body: []byte{}}
}
