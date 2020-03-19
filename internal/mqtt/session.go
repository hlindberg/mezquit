package mqtt

import (
	"fmt"
	"io"
	"log"

	"github.com/lithammer/shortuuid"
)

// Session describes a client session that may span several connects to a MQTT Broker
// It keeps track of package IDs "in flight" and a Client ID.
// It requires one io.Writer and one io.Reader to operate. It does not handle a Network connection - this is
// the responsability of the caller (open/dial, close, reconnect, etc.)
//
type Session struct {
	freeNumbers     chan int
	returnedNumbers chan int
	options         SessionOptions
	inFlight        *inFlight
}

func (s *Session) initInFlight(doClean bool) {
	// Clear the InFlight if this is first connect, or explicitly asking for a CleanSession
	if s.inFlight == nil || doClean {
		s.inFlight = newInFlight()
	}
}

// Connect connects to a MQTT broker and returns after having received a CONNACK
// The ClientName ConnectOption should not be included in the ConnectOptions as it is defined by the Session.
// If given as an option here it will be silently overwritten by the name given for the session.
//
// If calling this to continue the session (after an optional ReEstablis()), the CleanSession(false) option
// should be used if QoS > 0 and there is a desire to continue with the same packets "in flight".
//
func (s *Session) Connect(options ...ConnectOption) error {
	s.assertReaderWriter()

	// Create a request (override the client name by appending it - thus overwriting what user gave)
	options = append(options, ClientName(s.options.ClientName))
	connectionRequest := NewConnectRequest(options...)
	s.initInFlight(connectionRequest.IsCleanSession())

	// Send CONNECT
	_, err := connectionRequest.WriteTo(s.options.Writer)
	// TODO: Log the write in debug
	if err != nil {
		// TODO: Log this
		return err
	}

	// Wait for CONNACK
	// SPEC: The first packet sent by a broker on a CONNECT must be a CONNACK (thus it is ok here to wait on one here)
	//
	// TODO: SPEC 3.1.1 states that if CONNACK does not arrive within reasonable time (left open) the client should
	// close the connection. This could be specified in the Session (ConnectTimeOutSec).
	// See here: https://forum.golangbridge.org/t/how-to-set-timeout-for-io-reader/5207
	// If a timeout occurs, the caller must be informed to enable closing the connection and thus stopping a possibly blocked read
	//
	// TODO: When reading, should loop until getting all of the bytes. Timeout should be for getting all of them
	response := make([]byte, 4)
	n, err := s.options.Reader.Read(response)
	if err != nil {
		// TODO: Log this
		return err
	}
	if n != 4 {
		return fmt.Errorf("Expected to read 4 bytes from MQTT Reader but got %d", n)
	}

	if response[0] != ConnAckType<<4 {
		return fmt.Errorf("Did not get a CONNACK back from Connect - got %d", response[0])
	}
	if response[1] != 2 {
		return fmt.Errorf("Expected CONNACK length of 2 but got %d", response[1])
	}

	if response[2] == 1 {
		// TODO: Change to log this at debug level
		// TODO: What is the practical meaning of this? Does a client need to know?
		fmt.Printf("Got SP CONNACK flag set in response\n")
	}

	if response[3] != ConnectionAccepted {
		// TODO: This should translate the error return status to human readable text, not just include the status as a number
		return fmt.Errorf("Did not get ConnectionAccepted return status back - got %d", response[3])
	}

	return nil
}

// Publish publishes to a MQTT broker and if QoS is > 0 handles the ACK messages that follows
func (s *Session) Publish(options ...PublishOption) {
	s.assertReaderWriter()

	publishRequest := NewPublishRequest(options...)
	publishRequest.WriteTo(s.options.Writer)
}

func (s *Session) assertReaderWriter() {
	if s.options.Reader == nil || s.options.Writer == nil {
		panic("Session requires both a Reader and a Writer to operate")
	}
}

// SessionOptions are options applicable to a Session
//
type SessionOptions struct {
	ClientName string
	Reader     io.Reader
	Writer     io.Writer
}

// DefaultSessionOptions returns the defaults options for a session
func DefaultSessionOptions() SessionOptions {
	return SessionOptions{}
}

// SessionOption is an Options-modifying-function
type SessionOption func(*SessionOptions) error

// NewSession creates a session that can be used to connect multiple times to a MQTT broker
// with retained session information.
//
func NewSession(options ...SessionOption) *Session {
	opts := DefaultSessionOptions()
	for _, fOpt := range options {
		if err := fOpt(&opts); err != nil {
			log.Fatalf("Session option apply failure: %s", err)
		}
	}

	return &Session{options: opts}
}

// ReEstablish enables modifying the Input/Output options of an existing Session (i.e. for a new network connection).
// This is only meaningful if QoS > 0 since for 0, a NewSession can be used for each Connect.
//
// TODO: This impl allows also changing the ClientName which is not a good idea).
//
// Example:
//     s.ReEstablish(InputOutput(conn))
//
func (s *Session) ReEstablish(options ...SessionOption) {
	opts := &s.options
	for _, fOpt := range options {
		if err := fOpt(opts); err != nil {
			log.Fatalf("Session option apply failure: %s", err)
		}
	}
}

// ClientID returns a SessionOption for the given clientName
func ClientID(clientName string) SessionOption {
	return func(o *SessionOptions) error {
		o.ClientName = clientName
		return nil
	}
}

// Input returns a SessionOption for the given io.Reader
func Input(reader io.Reader) SessionOption {
	return func(o *SessionOptions) error {
		o.Reader = reader
		return nil
	}
}

// Output returns a SessionOption for the given io.Writer
func Output(writer io.Writer) SessionOption {
	return func(o *SessionOptions) error {
		o.Writer = writer
		return nil
	}
}

// InputOutput returns a SessionOption for the given io.ReadWriter, it sets both Reader and Writer
func InputOutput(rw io.ReadWriter) SessionOption {
	return func(o *SessionOptions) error {
		o.Reader = rw
		o.Writer = rw
		return nil
	}
}

// RandomClientID returns a random UUID string that can be used as ClientName in a Connection.
// A Short UUID - a Base 57 encoded string is returned.
//
func RandomClientID() string {
	return shortuuid.New()
}
