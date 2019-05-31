package client

import "net"

// HealthState represents the current state of the client
type HealthState int

const (
	// Unregistered represents the client state before registration
	Unregistered HealthState = 0
	// Registering represents state during registration
	Registering HealthState = 1
	// Active represents state when packets are being forwarded to the client
	Active HealthState = 2
	// Paused represents state when client is registered but packets are not being forwarded
	Paused HealthState = 3
)

// Client holds the current state and configuration for a backend
type Client struct {
	dataIP net.IP
	vip    net.IP
	state  HealthState
}

// NewClient creates a new Client object
func NewClient(dataIP, vip net.IP) *Client {
	return &Client{
		dataIP: dataIP,
		vip:    vip,
		state:  Unregistered}
}

// Start attempts to register and listen for connections
func (c *Client) Start(connectIP net.IP) {
	c.state = Registering
}
