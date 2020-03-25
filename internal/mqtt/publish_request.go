package mqtt

import (
	"bytes"
	"fmt"
	"log"
)

// PublishRequest describes a MQTT Publish
type PublishRequest struct {
	options PublishOptions
}

// NewPublishRequest creates an instance from default publish options plus given options.
//
func NewPublishRequest(options ...PublishOption) *PublishRequest {
	opts := DefaultPublishOptions()
	for _, fOpt := range options {
		if err := fOpt(&opts); err != nil {
			log.Fatalf("Publish option apply failure: %s", err)
		}
	}
	return &PublishRequest{options: opts}
}

// remainingLength computes the Remaining Length value to use in the Fixed Header
//
func (r *PublishRequest) remainingLength() int {

	result := 0
	lengths := 0

	result += len(r.options.Topic)
	lengths++

	result += len(r.options.Message) // length of message is not separately encoded in the packet (no addition to lenghts here)

	if r.options.QoS > 0 {
		lengths++ // Packet ID 2 bytes must be present
	}

	// lengths + 2 bytes per included item for its 16 bits length
	return result + lengths*2
}

func (r *PublishRequest) fixedHeaderBits() byte {
	result := byte(PublishType << 4)
	if r.options.QoS == 1 {
		result |= QoSOne
	} else if r.options.QoS == 2 {
		result |= QoSTwo
	}
	if r.options.Retain {
		result |= RetainBit
	}
	if r.options.IsDuplicate {
		result |= DupBit
	}
	return result
}

func (r *PublishRequest) makeMessage() *GenericMessage {
	var data bytes.Buffer          // 64 bytes
	data.Grow(r.remainingLength()) // ensure all to be written fits using only one buffer allocation

	// VARIABLE HEADER
	EncodeStringTo(r.options.Topic, &data)

	if r.options.QoS > 0 {
		Encode16BitIntTo(r.options.PacketID, &data)
	}

	// PAYLOAD
	// Message is without preceeding lenght (calculated from the remainder of the "remainingLength")
	data.Write(r.options.Message)
	return &GenericMessage{fixedHeader: r.fixedHeaderBits(), body: data.Bytes()}
}

// PublishOptions contains options for a ConnectRequest
//
type PublishOptions struct {
	Topic       string
	Message     []byte
	QoS         int
	Retain      bool
	IsDuplicate bool // signals that this is a duplicate
	PacketID    int  // 16 bits ID
}

// PublishOption is an Options-modifying-function
type PublishOption func(*PublishOptions) error

// DefaultPublishOptions returns the default options for making a MQTT publish using QoS 0
//
func DefaultPublishOptions() PublishOptions {
	return PublishOptions{QoS: 0, PacketID: 0, IsDuplicate: false}
}

// Message returns a PublishOption for this Message
func Message(msg []byte) PublishOption {
	return func(o *PublishOptions) error {
		o.Message = msg
		return nil
	}
}

// Topic returns a PublishOption for this Topic
func Topic(topic string) PublishOption {
	return func(o *PublishOptions) error {
		o.Topic = topic
		return nil
	}
}

// QoS returns a PublishOption for this QoS
func QoS(value int) PublishOption {
	if value < 0 || value > 2 {
		panic(fmt.Sprintf("QoS must be 0, 1, or 2, got %d", value))
	}
	return func(o *PublishOptions) error {
		o.QoS = value
		return nil
	}
}

// Retain returns a PublishOption for this Retain
func Retain(flag bool) PublishOption {
	return func(o *PublishOptions) error {
		o.Retain = flag
		return nil
	}
}

// IsDuplicate returns a PublishOption indicating this is a duplicate delivery
func IsDuplicate(flag bool) PublishOption {
	return func(o *PublishOptions) error {
		o.IsDuplicate = flag
		return nil
	}
}

// PacketID returns a PublishOption indicating the Packed ID
func PacketID(id int) PublishOption {
	if id < 0 || id > 0xffff {
		panic(fmt.Sprintf("PackedID must be in range 0 - 0xffff, got %x", id))
	}
	return func(o *PublishOptions) error {
		o.PacketID = id
		return nil
	}
}
