package mqtt

import (
	"bytes"
	"testing"

	"github.com/puppetlabs/scarp/testutils"
)

func Test_ConnectRequest_makeMessage_and_WriteTo(t *testing.T) {

	connectionRequest := NewConnectRequest(ClientName("MqttUnitTest"))
	msg := connectionRequest.makeMessage()
	var buf2 bytes.Buffer
	msg.WriteTo(&buf2)
	testutils.CheckEqual(26, buf2.Len(), t)
}
