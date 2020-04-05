package mqtt

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/hlindberg/mezquit/testutils"
)

func Test_MockConnection_implements_net_Conn(t *testing.T) {
	defer testutils.ShouldNotPanic(t)
	_ = net.Conn(NewMockConnection())
}

func Test_MockConnection_has_hardocded_local_and_remote_addr(t *testing.T) {
	conn := NewMockConnection()
	local := conn.LocalAddr()
	remote := conn.RemoteAddr()
	testutils.CheckEqual("0.0.0.0", local.String(), t)
	testutils.CheckEqual("tcp", local.Network(), t)
	testutils.CheckEqual("0.0.0.0", remote.String(), t)
	testutils.CheckEqual("tcp", remote.Network(), t)
}

// Test that RemoteWrite can be called without error
func Test_MockConnection_Can_do_RemoteWrite(t *testing.T) {
	conn := NewMockConnection()
	n, err := conn.RemoteWrite([]byte("test"))
	testutils.CheckEqual(4, n, t)
	testutils.CheckNotError(err, t)
}

// Test that what is written "remotely" can be read "locally"
func Test_MockConnection_Read_reads_what_was_written_with_RemoteWrite(t *testing.T) {
	conn := NewMockConnection()
	n, err := conn.RemoteWrite([]byte("test"))
	testutils.CheckEqual(4, n, t)
	testutils.CheckNotError(err, t)
	buf := make([]byte, 4)
	n, err = conn.Read(buf)
	testutils.CheckEqual(4, n, t)
	testutils.CheckNotError(err, t)
}

// Test that what is written "locally" can be read "remotely"
func Test_MockConnection_ReadRemote_reads_what_was_written_with_Write(t *testing.T) {
	conn := NewMockConnection()
	n, err := conn.Write([]byte("test"))
	testutils.CheckEqual(4, n, t)
	testutils.CheckNotError(err, t)
	buf := make([]byte, 4)
	n, err = conn.RemoteRead(buf)
	testutils.CheckEqual(4, n, t)
	testutils.CheckNotError(err, t)
}

func Test_MockConnection_Read_waits_for_data_until_close(t *testing.T) {
	conn := NewMockConnection()
	readResult := make(chan error)
	timeout := make(chan bool)
	readEndTime := time.Now()
	closeTime := time.Now().Add(1 * time.Millisecond) // ensure this is after

	go func() {
		delay := 200 * time.Millisecond
		time.Sleep(delay)
		timeout <- true
	}()
	go func() {
		aByte := make([]byte, 1)
		_, err := conn.Read(aByte)
		readEndTime = time.Now()
		readResult <- err
	}()

	// Wait for the timeout
	<-timeout
	conn.Close()

	// Wait for the read
	err := <-readResult

	// Did they occur in the expected order?
	testutils.CheckTrue(readEndTime.After(closeTime), t)
	testutils.CheckTrue(err == io.EOF, t)
}

func Test_MockConnection_Read_waits_until_given_read_deadline(t *testing.T) {
	conn := NewMockConnection()
	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))

	aByte := make([]byte, 1)
	_, err := conn.Read(aByte)
	nerr, ok := err.(net.Error)
	if !ok {
		t.Fatalf("Expected a net.Error but could not convert it")
	}
	testutils.CheckTrue(nerr.Timeout(), t)
}

func Test_MockConnection_Read_returns_amount_read_if_buffer_becomes_empty(t *testing.T) {
	conn := NewMockConnection()
	oneByte := []byte{1}
	n, err := conn.RemoteWrite(oneByte)
	testutils.CheckEqual(1, n, t)
	testutils.CheckNotError(err, t)

	threeBytes := make([]byte, 3)
	n, err = conn.Read(threeBytes)
	testutils.CheckEqual(1, n, t)
	testutils.CheckNotError(err, t)
	testutils.CheckEqual(byte(1), threeBytes[0], t)

	oneByte = []byte{2}
	n, err = conn.RemoteWrite(oneByte)
	testutils.CheckEqual(1, n, t)
	testutils.CheckNotError(err, t)

	n, err = conn.Read(threeBytes)
	testutils.CheckEqual(1, n, t)
	testutils.CheckNotError(err, t)
	testutils.CheckEqual(byte(2), threeBytes[0], t)
}

// TODO: Test Multithreaded reading and writing
// 1. n threads write 1 byte each, n threads read one byte each - all bytes are read
