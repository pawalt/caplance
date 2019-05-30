package backends

import "net"

// PacketForwarder is an interface for forwarding packets to the appropriate backend
type PacketForwarder interface {
	SendData(data []byte) error
	Close() error
}

// Backend is the data struct for a single backend
type Backend struct {
	name   string          // name of backend
	ip     net.IP          // ip for the balancer to send data to
	Writer PacketForwarder // interface for sending to backend
}

// NewBackend creates a new Backend
func NewBackend(name string, ip net.IP, writer PacketForwarder) *Backend {
	return &Backend{name, ip, writer}
}

// UDPForwarder is an implementation of PacketForwarder that uses UDP as the
// underlying packet encapsulation
type UDPForwarder struct {
	conn net.Conn
}

// NewUDPForwarder creates a new UDP Forwarder
func NewUDPForwarder(conn net.Conn) *UDPForwarder {
	return &UDPForwarder{conn}
}

// SendData sends the desired packet over UDP
func (f *UDPForwarder) SendData(data []byte) error {
	_, err := f.conn.Write(data)
	return err
}

// Close closes the underlying UDP connection
func (f *UDPForwarder) Close() error {
	err := f.conn.Close()
	return err
}
