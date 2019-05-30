package backends

import (
	"bufio"
	"net"
	"time"
)

// BackendCommunicator is the connection manager for a backend
type BackendCommunicator interface {
	ReadLine() (string, error)
	WriteLine(data string) error
	Close() error
}

// TCPCommunicator is an implementation of BackendCommunicator over TCP
type TCPCommunicator struct {
	reader       *bufio.Reader
	writer       *bufio.Writer
	conn         net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
}

// NewTCPCommunicator creates a new TCP Communicator
func NewTCPCommunicator(conn net.Conn, readInt, writeInt int) *TCPCommunicator {
	readTimeout := time.Duration(readInt) * time.Second
	writeTimeout := time.Duration(writeInt) * time.Second
	return &TCPCommunicator{
		bufio.NewReader(conn),
		bufio.NewWriter(conn),
		conn,
		readTimeout,
		writeTimeout}
}

// ReadLine reads a line from the connection with the applied timeout
func (t *TCPCommunicator) ReadLine() (string, error) {
	t.conn.SetReadDeadline(time.Now().Add(t.readTimeout))
	return t.reader.ReadString('\n')
}

// WriteLine writes to the connection with the applied timeout
func (t *TCPCommunicator) WriteLine(data string) error {
	t.conn.SetWriteDeadline(time.Now().Add(t.writeTimeout))
	_, err := t.writer.WriteString(data + "\n")
	if err != nil {
		return err
	}
	err = t.writer.Flush()
	return err
}

// Close closes the underlying tcp connection
func (t *TCPCommunicator) Close() error {
	return t.conn.Close()
}
