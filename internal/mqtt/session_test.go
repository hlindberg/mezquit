package mqtt

import (
	"io"
	"testing"

	"github.com/hlindberg/mezquit/testutils"
)

// immediate and with timeout (to process those waiting)
// with QoS 1 and 2
// EOF / ConnectionClosed
// Reconnect with QoS 2
// Resends
// Wills when not disconnecting with Disconnect ? (difficult, need a reader testing broker)
// Cannot Connect when Connected
// Cannot Disconnect when not Connected (although INITIAL is fine - since it never was connected - does nothing)
//
func Test_Session_Connect_and_Disconnect_QoS_0_immediate(t *testing.T) {
	conn := NewMockConnection()
	_, err := conn.RemoteWrite(testhelperConnectionAccepted())
	testutils.CheckNotError(err, t)

	session := NewSession(ClientID("MqttUnitTest"), Connection(conn))
	err = session.Connect()
	testutils.CheckNotError(err, t)

	// Immediate disconnect
	err = session.Disconnect(0)
	testutils.CheckNotError(err, t)

	// Check that Connect and Disconnect was emitted
	// CONNECT
	theRemoteSide := conn.Remote()
	testhelperConsumeConnect(theRemoteSide, t)

	firstByte, err := theRemoteSide.ReadByte()
	testutils.CheckNotError(err, t)
	testutils.CheckEqual(DisconnectType<<4, int(firstByte), t)
	lengthByte, err := theRemoteSide.ReadByte()
	testutils.CheckNotError(err, t)
	testutils.CheckEqual(byte(0), lengthByte, t)
}

// func Test_Session_ConnectQoS_2(t *testing.T) {
// 	inF := newInFlight()
// 	next := inF.nextPacketID()
// 	testutils.CheckEqual(1, next, t)
// }
// func Test_Session_Disconnect_QoS_2(t *testing.T) {
// 	inF := newInFlight()
// 	next := inF.nextPacketID()
// 	testutils.CheckEqual(1, next, t)
// }

func testhelperConnectionAccepted() []byte {
	connectResponse := make([]byte, 4)
	connectResponse[0] = ConnAckType << 4
	connectResponse[1] = 2 // lenght
	connectResponse[2] = 0 // no SP flag
	connectResponse[3] = ConnectionAccepted

	return connectResponse
}

// Consumes a Connect request from the reader
func testhelperConsumeConnect(reader io.Reader, t *testing.T) {
	t.Helper()
	oneByte := make([]byte, 1)
	n, err := reader.Read(oneByte)
	connFirst := oneByte[0]
	testutils.CheckNotError(err, t)
	testutils.CheckEqual(ConnectType<<4, int(connFirst), t)
	value, err := DecodeVariableInt(reader)
	testutils.CheckEqual(24, value, t)
	connectMessage := make([]byte, value)
	n, err = reader.Read(connectMessage)
	testutils.CheckEqual(value, n, t)

	// TODO: Make checks to assert the Connect request

}
