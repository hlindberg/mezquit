package mqtt

import (
	"fmt"
	"io"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/lithammer/shortuuid"
)

const (
	// INITIAL Session state is before session has been used/connected
	INITIAL = iota

	// CONNECTED Session state is when Session is connected (or thinks it is - it does not know about the state of the actual network connection)
	CONNECTED

	// DISCONNECTING Session state is when Session is in the process of disconnecting (waiting for queues to drain)
	DISCONNECTING

	// DISCONNECTED Session state is when Session has been DISCONNECTED (and it is possible to reconnect)
	DISCONNECTED
)

// Session describes a client session that may span several connects to a MQTT Broker
// It keeps track of package IDs "in flight" and a Client ID.
// It requires one io.Writer and one io.Reader to operate. It does not handle a Network connection - this is
// the responsability of the caller (open/dial, close, reconnect, etc.)
//
type Session struct {
	options   SessionOptions
	inFlight  *inFlight
	stopAfter chan int
	stopped   chan bool
	toBroker  chan MessageWriter
	drained   chan bool
	state     int
	mutex     *sync.RWMutex // mutex for session state changes
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

	// Since go does not have mutex transitions read->write and vice versa a write lock is needed here
	// since there can otherwise be reace conditions in the gap between releasing a read lock and aquiring a write lock.
	// meaning state could have changed. Instead this always aquires a write lock.
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Error if not in INITIAL, or DISCONNECTED state
	if !(s.state == INITIAL || s.state == DISCONNECTED) {
		// i.e. cannot connect when disconnecting (waiting for drains), and also not when already connected
		return fmt.Errorf("Cannot Connect when session is disconnecting or already connected")
	}

	// Create a request (override the client name by appending it - thus overwriting what user gave)
	options = append(options, ClientName(s.options.ClientName))
	connectionRequest := NewConnectRequest(options...)
	s.initInFlight(connectionRequest.IsCleanSession())

	// Send CONNECT
	log.Debugf("Broker <- CONNECT(%s)", connectionRequest.options.ClientName)

	msg := connectionRequest.makeMessage()
	_, err := msg.WriteTo(s.options.Writer)
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

	spFlagSet := false
	if response[2] == 1 {
		spFlagSet = true // TODO: What does this mean to a client?
	}

	if response[3] != ConnectionAccepted {
		// TODO: This should translate the error return status to human readable text, not just include the status as a number
		return fmt.Errorf("Did not get ConnectionAccepted return status back - got %d", response[3])
	}

	log.Debugf("Broker -> CONNACK(sp=%v) received ok", spFlagSet)

	s.state = CONNECTED

	// Start a lister goroutine that listens on the connection and a channel where a timeout can be received
	// signalling that it should stop after that timeout (or 0).
	log.Debugf("Session: starting handleMessages()")
	s.handleMessages()

	// Start a send to broker goroutine that listens on the `toBroker` queue and writes them in order
	//
	log.Debugf("Session: Starting startSendToBroker()")
	s.startSendToBroker()

	return nil
}

// DisconnectWithoutMessage performs flushing of messages just like Disconnect() but does not send a
// DISCONNECT message to the broker.
// This is used to test this scenario.
//
func (s *Session) DisconnectWithoutMessage(timeout int) error {
	log.Debugf("DisconnectWithoutMessage()")
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.state == INITIAL {
		return nil // wasn't connected in the first place - no work to do.
	}
	if s.state != CONNECTED {
		return fmt.Errorf("Session can only be flushed when it is in INITIAL, or CONNECTED state")
	}
	log.Debugf("Session: Stopping messageHandler with Timeout %d", timeout)
	// send stop to the incoming message handler
	s.stopAfter <- timeout

	// Wait for message handler to stop
	_ = <-s.stopped

	// Stop accepting messages to the s.toBroker channel - the queue will be drained
	close(s.toBroker)

	// Wait for outgoing messages to drain
	// TODO: maybe enqueue an error in case the drain fails...
	_ = <-s.drained

	log.Debugf("Session: Queue to broker drained")

	s.state = DISCONNECTED
	return nil // TODO: maybe return an error produced by the drain? Bigger question is handling errors while sending?
}

// Disconnect disconnects the MQTT session from the broker in an orderly fashion by sending a DISCONNECT message
// The `drain` parameter, if set to `true` will ensure that the Session will wait at least the given `timeout` in seconds
// to allow messages in flight to be processed. The disconnect will be sent as soon as the in-flight message set is empty
// or the timeout occurs. If `drain` is set to `false`, processing of incoming ACKS will stop as soon as possible and
// the DISCONNECT is then sent.
//
// TODO: Block posts and subscriptions after Disconnect is called
//
func (s *Session) Disconnect(timeout int) error {
	log.Debugf("Disconnect()")

	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.state == INITIAL {
		return nil // wasn't connected in the first place - no work to do.
	}
	if s.state != CONNECTED {
		return fmt.Errorf("Session can only be disconnected when it is in INITIAL, or CONNECTED state")
	}

	log.Debugf("Session: Stopping messageHandler with Timeout %d", timeout)
	// send stop to the incoming message handler
	s.stopAfter <- timeout

	// Wait for message handler to stop
	_ = <-s.stopped

	log.Debugf("Session: messageHandler() stop signal received")

	log.Debugf("Broker <- DISCONNECT")

	// Enqueue the disconnect to be sent to the broker
	s.toBroker <- NewDisconnectMessage()

	// Stop accepting messages to the s.toBroker channel - the queue will be drained
	close(s.toBroker)

	// Wait for outgoing messages to drain (including the disconnect)
	// TODO: maybe enqueue an error in case the drain fails...
	_ = <-s.drained

	log.Debugf("Session: Queue to broker drained")

	s.state = DISCONNECTED
	return nil // TODO: maybe return an error produced by the drain? Bigger question is handling errors while sending?
}

// startSendToBroker starts a goroutine that reads s.toBroker and sends whatever is posted there
// to the broker. This continues until s.toBroker channel is closed.
//
func (s *Session) startSendToBroker() {
	s.toBroker = make(chan MessageWriter, 100)
	go func() {
		for message := range s.toBroker {
			message.WriteTo(s.options.Writer)
		}
		s.drained <- true // signal all written
	}()
}

// HandleMessages starts go routines that listens for incoming ACK and performs the required housekeeping
// of messages in-flight.
// Note that a client should call `Disconnect` for an orderly disconnect - that will also optionally do a drain with
// a timeout.
//
func (s *Session) handleMessages() {

	go func() {
		timeout := make(chan bool)
		messages := make(chan *GenericMessage, 100)

		go func() {
			fixedHeader := make([]byte, 1)
			for {
				n, err := io.ReadFull(s.options.Reader, fixedHeader)
				if err != nil {
					if err == io.EOF || n == 0 {
						log.Debugf("Read Loop: EOF on broker connection - stopped reading")
						break // stop loop, connection is closed
					}
				}
				msg, err := s.readMessage(fixedHeader[0])
				if err != nil {
					log.Debugf("readMessage() returned error %s", err)
					break // stop loop - don't know what to do...
				}
				messages <- msg
			}

		}()

		for {
			select {
			case cancelTimeout := <-s.stopAfter:
				// When receiving information to stop after a timeout on drain, set a timer that will be selected
				// instead of blocking on a read from the broker
				//
				go func() {
					time.Sleep(time.Duration(cancelTimeout) * time.Second)
					timeout <- true
				}()

			case <-timeout:
				// Asked to stop after timeout - it now timed out, so stop waiting for headers
				s.stopped <- true
				break

			case msg := <-messages:
				// fan out to process specific handlers
				log.Debugf("Message Loop: msg type %x, length %d, bytes: %v", msg.fixedHeader, len(msg.body), msg.body)

				msgType := int(msg.fixedHeader >> 4)
				switch msgType {
				case PublishAckType:
					s.processPublishAck(msg)
				default:
					// TODO: for now panic as this logic will not correctly read the message to clear for next
					panic(fmt.Sprintf("Message Processing Loop: Unhandled message type %d - not yet implemented", msgType))
				}
			}
		}
	}()
}

func (s *Session) readMessage(fixedHeaderByte byte) (*GenericMessage, error) {
	remainingLength, err := DecodeVariableInt(s.options.Reader)
	if err != nil {
		return nil, err
	}
	msg := GenericMessage{fixedHeader: fixedHeaderByte, body: make([]byte, remainingLength)}
	n, err := io.ReadFull(s.options.Reader, msg.body)
	if n != remainingLength {
		err = fmt.Errorf("Expected to read %d bytes remaining length of message but got %d", remainingLength, n)
	}
	return &msg, err
}

func (s *Session) processPublishAck(msg *GenericMessage) {
	if msg.fixedHeader>>4 != PublishAckType {
		panic(fmt.Sprintf("processPublishAck() got generic message of wrong type: %d", msg.fixedHeader>>4))
	}
	body := msg.body
	if len(body) != 2 {
		panic(fmt.Sprintf("PUBACK expects 2 bytes packet ID as the body - got %d", len(body)))
	}
	packetID := int(body[0])<<8 | int(body[1])

	log.Debugf("PUBACK(%d) Received", packetID)

	// Packet is no longer waiting since this is QoS 1
	s.inFlight.releaseWaiting(packetID)

	// Mark packetID as available
	s.inFlight.unsetBit(packetID)
}

// Publish publishes to the connected MQTT broker (Session handles ACKs)
//
func (s *Session) Publish(options ...PublishOption) error {
	s.assertReaderWriter()
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.state != CONNECTED {
		return fmt.Errorf("Publish requires session to be in CONNECTED state")
	}
	var msg MessageWriter
	// Set PacketID if required
	pr := NewPublishRequest(options...)
	if pr.options.QoS > 0 && pr.options.PacketID == 0 {
		pr.options.PacketID = s.inFlight.nextPacketID()
		msg = pr.makeMessage()
		s.inFlight.registerWaiting(pr.options.PacketID, msg)
	} else {
		msg = pr.makeMessage()
	}
	s.toBroker <- msg
	return nil
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

	return &Session{
		options:   opts,
		stopAfter: make(chan int),
		stopped:   make(chan bool),
		drained:   make(chan bool),
		mutex:     &sync.RWMutex{},
		state:     INITIAL,
	}
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
