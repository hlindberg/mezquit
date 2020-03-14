package mqtt

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/lithammer/shortuuid"
)

// ConnectRequest describes a MQTT Connect
//
type ConnectRequest struct {
	options ConnectOptions
}

// remainingLength computes the Remaining Length value to use in the Fixed Header
//
func (r *ConnectRequest) remainingLength() int {

	result := 0
	count := 0
	if r.options.ClientName != "" {
		result += len(r.options.ClientName)
		count++
	}
	if r.options.WillTopic != "" {
		result += len(r.options.WillTopic)
		count++

		// there is always a message if there is a will topic - even if length is 0
		//
		result += len(r.options.WillMessage)
		count++
	}
	if r.options.UserName != "" {
		result += len(r.options.UserName)
		count++
	}
	if r.options.Password != nil {
		result += len(*r.options.Password)
		count++
	}
	// lengths + 2 bytes per included item for its 16 bits length
	return result + count*2
}

func (r *ConnectRequest) connectBits() byte {
	connectBits := byte(0)

	if r.options.CleanSession {
		connectBits |= CleanSessionFlag
	}

	if r.options.WillTopic != "" {
		connectBits |= WillFlag
	}

	if r.options.WillQoS != 0 {
		switch r.options.WillQoS {
		case 1:
			connectBits |= WillQoSOne
		case 2:
			connectBits |= WillQoSTwo
		}
	}

	if r.options.WillRetain {
		connectBits |= WillRetainFlag
	}

	if r.options.UserName != "" {
		connectBits |= UserNameFlag
	}

	if r.options.Password != nil {
		connectBits |= PasswordFlag
	}
	return connectBits
}

// WriteTo writes the ConnectRequest to the given io.Writer
//
func (r *ConnectRequest) WriteTo(writer io.Writer) (n int64, err error) {
	var data bytes.Buffer

	connectBits := r.connectBits()
	keepAlive := r.options.KeepAliveSeconds

	// FIXED CONNECT HEADER
	data.WriteByte(ConnectType<<4 | Reserved)

	// REMAINING LENGTH - VARIABLE HEADER LENGTH (always 10 for connect) + PAYLOAD LENGTH
	EncodeVariableIntTo(10+r.remainingLength(), &data)

	// Connect variable part            Byte   Description
	//                                  ------ ----------------------------------------------
	data.WriteByte(0)                      // (1)    Protocol Name Length MSB
	data.WriteByte(4)                      // (2)    Protocol Name Length LSB
	data.WriteString("MQTT")               // (3-6)  Protocol Name
	data.WriteByte(r.options.Level)        // (7)    Protocol Level - MQTT 3.1.1 is 4, MQTT 5 is 5
	data.WriteByte(connectBits)            // (8)    Connect Bits
	data.WriteByte(byte(keepAlive >> 8))   // (9)    Keep Alive Seconds MSB
	data.WriteByte(byte(keepAlive & 0xFF)) // (9-10) Keep Alive Seconds LSB

	// PAYLOAD
	// A Client ID is required as the first element of the payload.
	// It can (optionally, if broker allows it) be of length 0 to make the server assign the id.
	//
	EncodeStringTo(r.options.ClientName, &data) // (11 - 12 + string)

	// Output rest of optional payload in required order
	//
	if connectBits&WillFlag != 0 {
		EncodeStringTo(r.options.WillTopic, &data)
		EncodeBytesTo(r.options.WillMessage, &data)
	}

	if connectBits&UserNameFlag != 0 {
		EncodeStringTo(r.options.UserName, &data)
	}

	if connectBits&PasswordFlag != 0 {
		EncodeBytesTo(*r.options.Password, &data)
	}

	written, err := writer.Write(data.Bytes())
	return int64(written), err
}

// NewConnectRequest constructs a new ConnectRequest based on a default set of options
// overridden by given options.
//
// For example:
//    request := NewConnectRequest(Level(5), WillTopic("InTheEventOfMyDeath"), WillMessage("Give it all to science"))
//
func NewConnectRequest(options ...ConnectOption) *ConnectRequest {
	opts := DefaultConnectOptions()
	for _, fOpt := range options {
		if err := fOpt(&opts); err != nil {
			log.Fatalf("Connection option apply failure: %s", err)
		}
	}
	return &ConnectRequest{options: opts}
}

// DefaultConnectOptions returns the default options for making a MQTT connect using 3.1.1,
// a clean session, and with 10 seconds keep alive. ClientName is set to an empty string
// which may not be honored by all MQTT brokers. Use RandomClientID() function to produce
// a suitable string.
//
func DefaultConnectOptions() ConnectOptions {
	return ConnectOptions{Level: 4, CleanSession: true, KeepAliveSeconds: 10, ClientName: "", WillRetain: false}
}

// RandomClientID returns a random UUID string that can be used as ClientName in a Connection.
// A Short UUID - a Base 57 encoded string is returned.
//
func RandomClientID() string {
	return shortuuid.New()
}

// ConnectOptions contains options for a ConnectRequest
//
type ConnectOptions struct {
	Level            byte // 4 is MQTT: 3.1.1 and 5 is MQTT: 5
	CleanSession     bool // true is "start new session"
	KeepAliveSeconds int  // number of seconds to keep the connection alive
	ClientName       string
	WillTopic        string
	WillMessage      []byte // Only included in request if WillTopic is set to non empty string
	WillQoS          int
	WillRetain       bool
	UserName         string
	Password         *[]byte
}

// ConnectOption is an Options-modifying-function
type ConnectOption func(*ConnectOptions) error

// The noChange function can be used to ignore a value
// TODO: Maybe not very useful for ConnectionOptions
//
func noChangeConnectionOption(_ *ConnectOptions) error {
	return nil
}

// Level returns a ConnectionOption for Level
func Level(level int) ConnectOption {
	if !(level == 0 || level == 4 || level == 5) {
		panic(fmt.Sprintf("Level must be 0 (use default), 4 (use MQTT 3.1.1) or 5 (use MQTT 5), got %d", level))
	}
	if level == 0 {
		return noChangeConnectionOption
	}
	return func(o *ConnectOptions) error {
		o.Level = byte(level)
		return nil
	}
}

// CleanSession returns a ConnectionOption for CleanSession
func CleanSession(flag bool) ConnectOption {
	return func(o *ConnectOptions) error {
		o.CleanSession = flag
		return nil
	}
}

// KeepAliveSeconds returns a ConnectionOption for KeepAliveSeconds
func KeepAliveSeconds(value int) ConnectOption {
	if value < 0 {
		panic("KeepAliveSeconds cannot be negative")
	}
	if value > 0xff {
		panic(fmt.Sprintf("KeepAliveSeconds cannot be larger than 0xff, got %x", value))
	}

	return func(o *ConnectOptions) error {
		o.KeepAliveSeconds = value
		return nil
	}
}

// ClientName returns a ConnectionOption for ClientName
func ClientName(value string) ConnectOption {
	return func(o *ConnectOptions) error {
		o.ClientName = value
		return nil
	}
}

// WillTopic returns a ConnectionOption for WillTopic
func WillTopic(value string) ConnectOption {
	return func(o *ConnectOptions) error {
		o.WillTopic = value
		return nil
	}
}

// WillMessage returns a ConnectionOption for WillTopic
func WillMessage(value []byte) ConnectOption {
	return func(o *ConnectOptions) error {
		o.WillMessage = value
		return nil
	}
}

// WillRetain returns a ConnectionOption for WillRetain
func WillRetain(value bool) ConnectOption {
	return func(o *ConnectOptions) error {
		o.WillRetain = value
		return nil
	}
}

// WillQoS returns a ConnectionOption for WillQoS
func WillQoS(value int) ConnectOption {
	if value < 0 || value > 2 {
		panic(fmt.Sprintf("WillQoS must be 0, 1, or 2, got %d", value))
	}
	return func(o *ConnectOptions) error {
		o.WillQoS = value
		return nil
	}
}

// UserName returns a ConnectionOption for UserName
func UserName(value string) ConnectOption {
	return func(o *ConnectOptions) error {
		o.UserName = value
		return nil
	}
}

// Password returns a ConnectionOption for Password
func Password(value []byte) ConnectOption {
	return func(o *ConnectOptions) error {
		o.Password = &value
		return nil
	}
}
